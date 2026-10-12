import { appendFile, mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { isDeepStrictEqual } from 'node:util';
import { Driver, requireValue, validateResponse } from './protocol.mjs';
import { digest, hashFile } from './manifest.mjs';
import { NORMALIZATION, scoreSample } from './metrics.mjs';
import { streamingMetrics } from './streaming.mjs';
import { shuffle } from './statistics.mjs';

export function plan(config, cases = []) {
  const blocks = [];
  for (const seed of config.seeds) {
    for (const cacheState of config.cacheStates) {
      const items = config.variants.map(variant => ({ seed, cacheState, variant }));
      blocks.push(...shuffle(items, config.orderSeed + seed + (cacheState === 'cold' ? 1000 : 0)));
    }
  }
  return { version: 1, configHash: digest(config), normalization: NORMALIZATION, blocks, cases: cases.map(c => c.id), measuredRequests: blocks.length * cases.length };
}

export function inferenceRequest(sample, block, config, corpusFile, warmup = false) {
  // Ground truth and protected-span labels never cross the inference boundary.
  return {
    op: 'infer', warmup, sampleId: sample.id, seed: block.seed,
    cacheState: block.cacheState, variant: block.variant, pipeline: { ...config.pipeline, ...block.variant.pipelineOverrides },
    input: {
      audio: { ...sample.audio, path: path.resolve(path.dirname(corpusFile), sample.audio.path) },
      context: sample.context,
      approvedMemory: block.variant.features.memory ? sample.approvedMemory : [],
    },
  };
}

export async function verifyDriverArtifacts(driverConfig, directory) {
  requireValue(Array.isArray(driverConfig.artifacts) && driverConfig.artifacts.length > 0, 'Driver must pin local model/runtime artifacts');
  const ids = new Set();
  const artifacts = {};
  for (const artifact of driverConfig.artifacts) {
    requireValue(typeof artifact.id === 'string' && !ids.has(artifact.id) && typeof artifact.path === 'string' && /^[a-f0-9]{64}$/.test(artifact.sha256) && typeof artifact.license === 'string' && artifact.license.length > 0, 'Invalid artifact identity/hash/license');
    ids.add(artifact.id);
    requireValue(await hashFile(path.resolve(directory, artifact.path)) === artifact.sha256, `Artifact checksum mismatch for ${artifact.id}`);
    artifacts[artifact.id] = artifact.sha256;
  }
  return artifacts;
}

export async function runBenchmark(config, corpus, corpusFile, driverConfig, driverDirectory, output) {
  const cases = corpus.samples.filter(sample => sample.split === config.split);
  if (config.pipeline.streamingMetrics) requireValue(cases.every(sample => sample.reference === '' || (sample.speechIntervals?.length && sample.referenceTurns?.length)), 'Streaming evaluation requires reviewed speech timestamps');
  requireValue(cases.length > 0, 'Requested split is empty');
  const warmups = corpus.samples.filter(sample => sample.split === (config.split === 'test' ? 'dev' : 'train')).slice(0, config.warmupCount);
  requireValue(warmups.length === config.warmupCount, 'Separate non-test warmup fixtures required');
  requireValue(!warmups.some(s => cases.some(c => c.id === s.id)), 'Warmup fixtures overlap measured split');
  const artifactHashes = await verifyDriverArtifacts(driverConfig, driverDirectory);
  // Refuse overwriting completed or incomplete results.
  await mkdir(output, { recursive: false, mode: 0o700 });
  const executionPlan = plan(config, cases);
  const driver = new Driver(driverConfig, driverDirectory);
  let completedObservations = 0;
  try {
    const hello = await driver.request({ op: 'hello', purpose: 'benchmark', artifacts: artifactHashes });
    requireValue(hello.execution === 'live' && isDeepStrictEqual(hello.artifacts, artifactHashes), 'Live driver artifact attestation mismatch');
    requireValue(typeof hello.hardware?.os === 'string' && typeof hello.hardware?.cpu === 'string' && typeof hello.hardware?.powerMode === 'string' && Number.isFinite(hello.hardware?.ramBytes) && typeof hello.runtime?.version === 'string' && typeof hello.runtime?.buildFlags === 'string' && typeof hello.runtime?.timer === 'string', 'Hardware/runtime/timer provenance missing');
    if (config.pipeline.preview === 'native') requireValue(['darwin', 'win32'].includes(hello.hardware.os), 'Native preview measurements require actual macOS or Windows');
    const metadata = {
      version: 1, status: 'running', normalization: NORMALIZATION, config, configHash: digest(config),
      corpusHash: digest(corpus), driverConfigHash: digest(driverConfig), provenance: hello,
      gitRevision: execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(),
      gitDirty: execFileSync('git', ['status', '--porcelain'], { encoding: 'utf8' }).trim().length > 0,
      plan: executionPlan,
    };
    await writeFile(path.join(output, 'run.json'), JSON.stringify(metadata, null, 2), { mode: 0o600 });
    for (const block of executionPlan.blocks) {
      // Warm requests require resident models; cold requests must reset model state
      // per sample and include load cost. Warmup records are retained separately.
      for (const sample of warmups) {
        const payload = inferenceRequest(sample, { ...block, cacheState: 'warm' }, config, corpusFile, true);
        const started = performance.now();
        const response = await driver.request(payload);
        requireValue(performance.now() - started >= sample.audio.durationMs * 0.98, 'Realtime driver returned before audio could be replayed');
        validateResponse(response, { ...payload, requestId: response.requestId });
        await appendFile(path.join(output, 'warmups.jsonl'), `${JSON.stringify({ sampleId: sample.id, block, response })}\n`, { mode: 0o600 });
      }
      for (const sample of shuffle(cases, config.orderSeed + block.seed)) {
        const payload = inferenceRequest(sample, block, config, corpusFile);
        const started = performance.now();
        const response = await driver.request(payload);
        requireValue(performance.now() - started >= sample.audio.durationMs * 0.98, 'Realtime driver returned before audio could be replayed');
        validateResponse(response, { ...payload, requestId: response.requestId });
        const row = {
          caseId: sample.id, clusterId: sample.speakerId, language: sample.language, tags: sample.tags,
          unit: sample.unit, seed: block.seed, cacheState: block.cacheState, variantId: block.variant.id,
          candidateHash: digest(response.candidates), metrics: { ...scoreSample(sample, response), streaming: streamingMetrics(sample, response, payload.pipeline) }, response,
        };
        completedObservations++;
        await appendFile(path.join(output, 'results.jsonl'), `${JSON.stringify(row)}\n`, { mode: 0o600 });
      }
    }
    metadata.status = 'complete';
    metadata.observations = completedObservations;
    metadata.resultsSha256 = await hashFile(path.join(output, 'results.jsonl'));
    metadata.warmupsSha256 = await hashFile(path.join(output, 'warmups.jsonl'));
    await writeFile(path.join(output, 'run.json'), JSON.stringify(metadata, null, 2), { mode: 0o600 });
    return completedObservations;
  } catch (error) {
    await writeFile(path.join(output, 'failure.json'), JSON.stringify({ status: 'failed', completedObservations, reason: error.message }), { mode: 0o600 });
    throw error;
  } finally { driver.close(); }
}
