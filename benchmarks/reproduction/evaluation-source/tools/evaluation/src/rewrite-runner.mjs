import { appendFile, mkdir, writeFile } from 'node:fs/promises';
import { createReadStream } from 'node:fs';
import { createInterface } from 'node:readline';
import { execFileSync } from 'node:child_process';
import { isDeepStrictEqual } from 'node:util';
import path from 'node:path';
import { Driver, requireValue as need } from './protocol.mjs';
import { digest, hashFile, readJson } from './manifest.mjs';
import { verifyDriverArtifacts } from './runner.mjs';
import { shuffle, pairedBootstrap } from './statistics.mjs';
import { REWRITE_SCHEMA, validateRewriteConfig, validateRewriteCorpus } from './rewrite-manifest.mjs';
import { rewriteRequest, validateRewriteResponse } from './rewrite-protocol.mjs';
import { rewriteMetrics, aggregateRewrite } from './rewrite-metrics.mjs';

export function rewritePlan(config, samples = []) {
  validateRewriteConfig(config);
  const blocks = [];
  for (const seed of config.seeds) for (const cacheState of config.cacheStates) for (const loadState of config.loadStates) {
    blocks.push(...shuffle(config.variants.map(variant => ({ seed, cacheState, loadState, variant })), config.orderSeed + seed + (cacheState === 'cold' ? 1000 : 0) + (loadState === 'concurrent_asr' ? 2000 : 0)));
  }
  return { version: 1, kind: REWRITE_SCHEMA, configHash: digest(config), cases: samples.map(s => s.id), blocks, measuredRequests: blocks.length * samples.length };
}

export async function runRewrite(config, corpus, corpusFile, driverConfig, driverDirectory, output) {
  validateRewriteConfig(config); validateRewriteCorpus(corpus);
  const samples = corpus.samples.filter(s => s.split === config.split);
  const warmups = corpus.samples.filter(s => s.split === (config.split === 'test' ? 'dev' : 'train')).slice(0, config.warmupCount);
  need(samples.length > 0 && warmups.length === config.warmupCount, 'Held-out rewrite cases and separate warmups required');
  const artifacts = await verifyDriverArtifacts(driverConfig, driverDirectory);
  for (const key of ['model', 'prompt', 'tokenizer', 'asr']) need(Object.hasOwn(artifacts, config.pipeline[key]), `Missing pinned rewrite artifact: ${key}`);
  for (const v of config.variants) if (v.pipelineOverrides) need(Object.hasOwn(artifacts, v.pipelineOverrides.model), 'Missing pinned precision-comparison model artifact');
  await mkdir(output, { recursive: false, mode: 0o700 });
  const plan = rewritePlan(config, samples);
  let driver;
  let completedObservations = 0;
  let cleanupAttempted = false;
  let cleanupConfirmed = false;
  const cleanup = async () => {
    cleanupAttempted = true;
    const reply = await driver.request({ op: 'cleanup', purpose: 'rewrite', schema: REWRITE_SCHEMA });
    need(reply.execution === 'live' && reply.schema === REWRITE_SCHEMA && reply.cleanupComplete === true, 'Rewrite native-fixture cleanup was not confirmed');
    cleanupConfirmed = true;
  };
  const metadata = { version: 1, kind: REWRITE_SCHEMA, status: 'running', config, configHash: digest(config), corpusHash: digest(corpus), driverConfigHash: digest(driverConfig), plan, gitRevision: execFileSync('git', ['rev-parse', 'HEAD'], { encoding: 'utf8' }).trim(), gitDirty: execFileSync('git', ['status', '--porcelain'], { encoding: 'utf8' }).trim().length > 0 };
  await writeFile(path.join(output, 'run.json'), JSON.stringify(metadata, null, 2), { mode: 0o600 });
  await writeFile(path.join(output, 'corpus.json'), JSON.stringify(corpus), { mode: 0o600 });
  await writeFile(path.join(output, 'results.jsonl'), '', { mode: 0o600 });
  await writeFile(path.join(output, 'warmups.jsonl'), '', { mode: 0o600 });
  try {
    driver = new Driver(driverConfig, driverDirectory);
    const hello = await driver.request({ op: 'hello', purpose: 'rewrite', schema: REWRITE_SCHEMA, artifacts });
    need(hello.execution === 'live' && hello.schema === REWRITE_SCHEMA && isDeepStrictEqual(hello.artifacts, artifacts), 'Rewrite driver must attest live schema and checked artifacts');
    need(['darwin', 'win32'].includes(hello.hardware?.os) && typeof hello.hardware.cpu === 'string' && typeof hello.hardware.powerMode === 'string' && Number.isFinite(hello.hardware.ramBytes) && hello.hardware.ramBytes > 0, 'Actual desktop hardware provenance required');
    need(['version', 'buildFlags', 'timer', 'provider', 'scheduler', 'tokenizerVersion', 'promptHash'].every(k => typeof hello.runtime?.[k] === 'string' && hello.runtime[k].length > 0), 'Rewrite runtime/provider/scheduler/tokenizer/prompt provenance required');
    need(hello.runtime.provider === config.pipeline.provider && hello.runtime.promptHash === artifacts[config.pipeline.prompt], 'Pinned provider/prompt mismatch');
    need(Array.isArray(hello.capabilities) && ['native-preview', 'rewrite', 'cold-reset', 'invalidate', 'cleanup', ...(config.loadStates.includes('concurrent_asr') ? ['concurrent-asr'] : [])].every(c => hello.capabilities.includes(c)), 'Required rewrite driver capability missing');
    metadata.provenance = hello;
    await writeFile(path.join(output, 'run.json'), JSON.stringify(metadata, null, 2), { mode: 0o600 });
    for (const block of plan.blocks) {
      for (const [measured, items] of [[false, warmups], [true, shuffle(samples, config.orderSeed + block.seed)]]) {
        for (const sample of items) {
          const effectiveBlock = measured ? block : { ...block, cacheState: 'warm' };
          const payload = rewriteRequest(sample, effectiveBlock, config, corpusFile, !measured);
          const started = performance.now();
          const response = await driver.request(payload);
          const elapsed = performance.now() - started;
          validateRewriteResponse(response, { ...payload, requestId: response.requestId });
          need(elapsed + 5 >= response.timings.reviewMs - response.timings.requestStartMs, 'Driver timing exceeds observed elapsed wall time');
          if (block.loadState === 'concurrent_asr') need(elapsed >= sample.asrLoad.durationMs * 0.98 && elapsed + 5 >= response.asr.endedMs - response.asr.startedMs, 'Concurrent ASR returned before real replay');
          const row = { caseId: sample.id, clusterId: sample.clusterId, language: sample.language, inputKind: sample.inputKind, tags: sample.tags, unit: 'rewrite', seed: block.seed, cacheState: effectiveBlock.cacheState, plannedCacheState: block.cacheState, warmup: !measured, loadState: block.loadState, variantId: block.variant.id, response };
          await appendFile(path.join(output, measured ? 'results.jsonl' : 'warmups.jsonl'), `${JSON.stringify(row)}\n`, { mode: 0o600 });
          if (measured) completedObservations++;
        }
      }
    }
    await cleanup();
    metadata.status = 'complete'; metadata.observations = completedObservations; metadata.cleanupConfirmed = cleanupConfirmed;
    metadata.resultsSha256 = await hashFile(path.join(output, 'results.jsonl'));
    metadata.warmupsSha256 = await hashFile(path.join(output, 'warmups.jsonl'));
    await writeFile(path.join(output, 'run.json'), JSON.stringify(metadata, null, 2), { mode: 0o600 });
    return completedObservations;
  } catch (error) {
    if (driver && !cleanupAttempted) {
      try { await cleanup(); } catch (cleanupError) { metadata.cleanupFailure = cleanupError.message; }
    }
    metadata.status = 'failed'; metadata.observations = completedObservations;
    metadata.cleanupConfirmed = cleanupConfirmed;
    await writeFile(path.join(output, 'run.json'), JSON.stringify(metadata, null, 2), { mode: 0o600 });
    await writeFile(path.join(output, 'failure.json'), JSON.stringify({ status: 'failed', completedObservations, reason: error.message }), { mode: 0o600 });
    throw error;
  } finally { driver?.close(); }
}

export async function checkedRewriteRows(directory, { retainCandidates = false } = {}) {
  const metadata = await readJson(path.join(directory, 'run.json'));
  const config = validateRewriteConfig(metadata.config);
  const corpus = validateRewriteCorpus(await readJson(path.join(directory, 'corpus.json')));
  need(metadata.status === 'complete' && metadata.cleanupConfirmed === true && metadata.kind === REWRITE_SCHEMA && metadata.configHash === digest(config) && metadata.corpusHash === digest(corpus), 'Incomplete rewrite run/cleanup or changed corpus/config');
  need(await hashFile(path.join(directory, 'results.jsonl')) === metadata.resultsSha256 && await hashFile(path.join(directory, 'warmups.jsonl')) === metadata.warmupsSha256, 'Rewrite result/warmup checksum mismatch');
  const samples = corpus.samples.filter(s => s.split === config.split);
  const expectedPlan = rewritePlan(config, samples);
  need(isDeepStrictEqual(metadata.plan, expectedPlan), 'Rewrite execution plan changed');
  const byId = new Map(samples.map(s => [s.id, s]));
  const keys = new Set();
  const requestIds = new Set();
  const rows = [];
  for await (const line of createInterface({ input: createReadStream(path.join(directory, 'results.jsonl')), crlfDelay: Infinity })) {
    const row = JSON.parse(line);
    const sample = byId.get(row.caseId);
    const variant = config.variants.find(v => v.id === row.variantId);
    need(sample && variant && row.warmup === false && row.plannedCacheState === row.cacheState && config.seeds.includes(row.seed) && config.cacheStates.includes(row.cacheState) && config.loadStates.includes(row.loadState) && row.clusterId === sample.clusterId && row.language === sample.language && row.inputKind === sample.inputKind && row.unit === 'rewrite' && isDeepStrictEqual(row.tags, sample.tags), 'Unexpected rewrite observation');
    const key = `${row.caseId}:${row.variantId}:${row.seed}:${row.cacheState}:${row.loadState}`;
    need(!keys.has(key), 'Duplicate rewrite observation'); keys.add(key);
    const q = rewriteRequest(sample, { seed: row.seed, cacheState: row.cacheState, loadState: row.loadState, variant }, config, path.join(directory, 'corpus.json'));
    // Audio path relocation does not alter the trace's pinned content identity.
    validateRewriteResponse(row.response, { ...q, requestId: row.response.requestId });
    need(!requestIds.has(row.response.requestId), 'Reused rewrite request identity'); requestIds.add(row.response.requestId);
    row.metrics = rewriteMetrics(sample, row.response);
    if (retainCandidates) row.candidateText = row.response.candidateText;
    delete row.response; // Bootstrap keeps scores/identities, not raw token/preview traces.
    rows.push(row);
  }
  need(rows.length === expectedPlan.measuredRequests && rows.length === metadata.observations, 'Incomplete rewrite observation coverage');
  const warmups = corpus.samples.filter(s => s.split === (config.split === 'test' ? 'dev' : 'train')).slice(0, config.warmupCount);
  need(warmups.length === config.warmupCount, 'Missing separate rewrite warmup fixtures');
  const warmById = new Map(warmups.map(s => [s.id, s]));
  const warmKeys = new Set();
  for await (const line of createInterface({ input: createReadStream(path.join(directory, 'warmups.jsonl')), crlfDelay: Infinity })) {
    const row = JSON.parse(line); const sample = warmById.get(row.caseId); const variant = config.variants.find(v => v.id === row.variantId);
    need(sample && variant && row.warmup === true && row.cacheState === 'warm' && config.cacheStates.includes(row.plannedCacheState) && config.seeds.includes(row.seed) && config.loadStates.includes(row.loadState) && row.clusterId === sample.clusterId, 'Unexpected rewrite warmup observation');
    const key = `${row.caseId}:${row.variantId}:${row.seed}:${row.plannedCacheState}:${row.loadState}`;
    need(!warmKeys.has(key), 'Duplicate rewrite warmup'); warmKeys.add(key);
    const q = rewriteRequest(sample, { seed: row.seed, cacheState: 'warm', loadState: row.loadState, variant }, config, path.join(directory, 'corpus.json'), true);
    validateRewriteResponse(row.response, { ...q, requestId: row.response.requestId });
    need(!requestIds.has(row.response.requestId), 'Reused rewrite warmup request identity'); requestIds.add(row.response.requestId);
  }
  need(warmKeys.size === expectedPlan.blocks.length * config.warmupCount, 'Incomplete rewrite warmup coverage');
  return { metadata, corpus, rows };
}

function groups(rows) {
  const strata = { overall: rows };
  for (const key of ['language', 'inputKind']) for (const value of new Set(rows.map(r => r[key]))) strata[`${key}:${value}`] = rows.filter(r => r[key] === value);
  for (const tag of new Set(rows.flatMap(r => r.tags))) strata[`condition:${tag}`] = rows.filter(r => r.tags.includes(tag));
  return Object.fromEntries(Object.entries(strata).map(([k, r]) => [k, aggregateRewrite(r)]));
}

export async function reportRewrite(directory) {
  const { metadata, rows } = await checkedRewriteRows(directory);
  const config = metadata.config;
  const summaries = {};
  const comparisons = [];
  for (const cacheState of config.cacheStates) for (const loadState of config.loadStates) {
    const strata = rows.filter(r => r.cacheState === cacheState && r.loadState === loadState);
    for (const v of config.variants) summaries[`${v.id}:${cacheState}:${loadState}`] = groups(strata.filter(r => r.variantId === v.id));
    for (const a of config.variants) for (const b of config.variants) {
      const changed = Object.keys(a.features).filter(k => a.features[k] !== b.features[k]);
      const samePipeline = isDeepStrictEqual(a.pipelineOverrides ?? {}, b.pipelineOverrides ?? {});
      const precisionPair = changed.length === 0 && a.features.engine === 'llm' && (a.pipelineOverrides?.precision ?? config.pipeline.precision).startsWith('fp') && (b.pipelineOverrides?.precision ?? config.pipeline.precision).startsWith('int');
      if (!precisionPair && (!samePipeline || changed.length !== 1 || (changed[0] === 'engine' ? a.features.engine !== 'rules' || b.features.engine !== 'llm' : a.features[changed[0]] !== false))) continue;
      for (const [metric, get] of [['acceptanceRate', s => s.acceptanceRate], ['protectedDeliveryRecall', s => s.protectedDeliveryRecall], ['attackMarkerRate', s => s.attackMarkerRate], ['reviewMs.p95', s => s.reviewMs.p95]]) {
        comparisons.push({ baseline: a.id, treatment: b.id, factor: precisionPair ? 'model_precision' : changed[0], cacheState, loadState, metric, ...pairedBootstrap(strata.filter(r => r.variantId === a.id), strata.filter(r => r.variantId === b.id), r => get(aggregateRewrite(r)), config.bootstrap) });
      }
    }
  }
  const contentionComparisons = [];
  if (config.loadStates.includes('solo') && config.loadStates.includes('concurrent_asr')) for (const cacheState of config.cacheStates) for (const v of config.variants) {
    const a = rows.filter(r => r.cacheState === cacheState && r.variantId === v.id && r.loadState === 'solo');
    const b = rows.filter(r => r.cacheState === cacheState && r.variantId === v.id && r.loadState === 'concurrent_asr');
    contentionComparisons.push({ variantId: v.id, cacheState, metric: 'reviewMs.p95', ...pairedBootstrap(a, b, r => aggregateRewrite(r).reviewMs.p95, config.bootstrap) });
  }
  const report = { version: 1, kind: REWRITE_SCHEMA, status: 'complete', observations: rows.length, corpusHash: metadata.corpusHash, configHash: metadata.configHash, provenance: metadata.provenance, gitRevision: metadata.gitRevision, gitDirty: metadata.gitDirty, resultsSha256: metadata.resultsSha256, summaries, comparisons, contentionComparisons, semanticQuality: 'pending_blinded_human_review', inference: 'Exploratory author/content-cluster percentile CIs; repetitions stay in clusters, no multiple-comparison significance claim. Exact strings/numbers/reference overlap are proxies, not semantic quality. Main latency/stability excludes invalidation-control cases. Resource/cancellation/ASR counters include all requests.' };
  await writeFile(path.join(directory, 'rewrite-summary.json'), JSON.stringify(report, null, 2), { mode: 0o600 });
  const lines = ['# Rewrite benchmark', '', report.inference, '', '| Variant / cache / load | Count | Acceptance | Protected delivery | Constraint violation (accepted) | Review p95 ms |', '| --- | ---: | ---: | ---: | ---: | ---: |'];
  const show = n => n === null ? 'n/a' : n.toFixed(4);
  for (const [id, g] of Object.entries(summaries)) { const s = g.overall; lines.push(`| ${id} | ${s.count} | ${show(s.acceptanceRate)} | ${show(s.protectedDeliveryRecall)} | ${show(s.constraintViolationRateAccepted)} | ${show(s.reviewMs.p95)} |`); }
  lines.push('', 'Meaning, hallucination and style are pending blinded human review. Native lifecycle violations abort a run and cannot appear as a passing aggregate. Raw text remains in private corpus/results files.', '');
  await writeFile(path.join(directory, 'rewrite-summary.md'), lines.join('\n'), { mode: 0o600 });
  return report;
}
