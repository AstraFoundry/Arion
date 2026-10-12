import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { hashFile, loadCorpus } from '../src/manifest.mjs';
import { runBenchmark, verifyDriverArtifacts, plan } from '../src/runner.mjs';
import { digest } from '../src/manifest.mjs';
import { NORMALIZATION, scoreSample } from '../src/metrics.mjs';
import { writeReport } from '../src/report.mjs';

const original = JSON.parse(await readFile(new URL('../../../benchmarks/configs/desktop-factorial.json', import.meta.url), 'utf8'));
const cli = new URL('../cli.mjs', import.meta.url);

test('CLI cannot report absent adapters as a passing run or overwrite plans', async t => {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-cli-test-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const missing = spawnSync(process.execPath, [fileURLToPath(cli), 'run', '--config', fileURLToPath(new URL('../../../benchmarks/configs/desktop-factorial.json', import.meta.url))], { encoding: 'utf8' });
  assert.equal(missing.status, 1);
  assert.match(missing.stderr, /corpus.*required/);
  const args = [fileURLToPath(cli), 'plan', '--config', fileURLToPath(new URL('../../../benchmarks/configs/desktop-factorial.json', import.meta.url)), '--out', path.join(directory, 'plan.json')];
  assert.equal(spawnSync(process.execPath, args, { encoding: 'utf8' }).status, 0);
  assert.equal(spawnSync(process.execPath, args, { encoding: 'utf8' }).status, 1);
});

test('real runner refuses fixture drivers and retains a non-reportable failure', async t => {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-run-test-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const artifact = path.join(directory, 'artifact.txt');
  await writeFile(artifact, 'transport-test-only; not a model');
  const hash = await hashFile(artifact);
  const driverFile = path.join(directory, 'fixture.mjs');
  await writeFile(driverFile, "import {createInterface} from 'node:readline';\ncreateInterface({input:process.stdin}).on('line',line=>{const r=JSON.parse(line); console.log(JSON.stringify({protocol:1,requestId:r.requestId,execution:'fixture',artifacts:r.artifacts}));});\n");
  const driverConfig = { executable: process.execPath, args: [driverFile], timeoutMs: 1000, artifacts: [{ id: 'fixture', path: artifact, sha256: hash, license: 'testing-only' }] };
  await verifyDriverArtifacts(driverConfig, directory);
  await assert.rejects(verifyDriverArtifacts({ ...driverConfig, artifacts: [{ ...driverConfig.artifacts[0], sha256: '0'.repeat(64) }] }, directory), /checksum/);
  const sample = (id, split) => ({ id, split, speakerId: id, sessionId: id, language: 'en', tags: [], unit: 'wer', reference: 'test', context: '', approvedMemory: [], audio: { path: artifact, sha256: hash, durationMs: 1 } });
  const corpus = { version: 1, license: 'testing-only', consent: true, samples: [sample('dev-a', 'dev'), sample('dev-b', 'dev'), sample('test-a', 'test')] };
  for (const sample of corpus.samples) {
    const audio = path.join(directory, `${sample.id}.txt`);
    await writeFile(audio, `transport-only-fixture-${sample.id}`);
    sample.audio.path = audio;
    sample.audio.sha256 = await hashFile(audio);
  }
  const manifest = path.join(directory, 'corpus.json');
  await writeFile(manifest, JSON.stringify(corpus));
  await loadCorpus(manifest); // Hash/metadata validation only; decoder is a driver boundary.
  const output = path.join(directory, 'run');
  await assert.rejects(runBenchmark(original, corpus, manifest, driverConfig, directory, output), /Live driver/);
  const failure = JSON.parse(await readFile(path.join(output, 'failure.json'), 'utf8'));
  assert.equal(failure.status, 'failed');
  assert.equal(failure.completedObservations, 0);
  await assert.rejects(writeReport(output));
});

// Serialization oracle only: these rows are synthetic metric inputs, not model runs.
test('report serialization validates file hashes and retains null metric strata', async t => {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-report-test-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const config = { ...original, seeds: [11], cacheStates: ['warm'], bootstrap: { iterations: 100, seed: 1729 } };
  const samples = ['a', 'b'].map(id => ({ id, speakerId: id, unit: 'wer', language: 'en', tags: ['metric-oracle'], reference: 'one', audio: { durationMs: 1000 } }));
  const response = { rawText: 'one', selectedText: 'one', finalText: 'one', candidates: [{ id: 'x', text: 'one', source: 'metric-oracle' }], rankedIds: ['x'], events: [{ text: 'one', ms: 1000, final: true }], timings: { firstPartialMs: null, speechStartMs: 0, speechEndMs: 900, finalMs: 1000, reviewMs: 1000, rankerMs: null, computeMs: 1, loadMs: 0 }, resources: { peakRssBytes: 1000 } };
  const executionPlan = plan(config, samples);
  const rows = executionPlan.blocks.flatMap(block => samples.map(sample => ({ caseId: sample.id, clusterId: sample.speakerId, unit: sample.unit, language: sample.language, tags: sample.tags, seed: block.seed, cacheState: block.cacheState, variantId: block.variant.id, candidateHash: digest(response.candidates), metrics: scoreSample(sample, response), response })));
  const resultsFile = path.join(directory, 'results.jsonl');
  const warmupsFile = path.join(directory, 'warmups.jsonl');
  await writeFile(resultsFile, rows.map(row => JSON.stringify(row)).join('\n') + '\n');
  await writeFile(warmupsFile, '{}\n');
  await writeFile(path.join(directory, 'run.json'), JSON.stringify({ status: 'complete', config, configHash: digest(config), corpusHash: 'metric-oracle', normalization: NORMALIZATION, observations: rows.length, plan: executionPlan, resultsSha256: await hashFile(resultsFile), warmupsSha256: await hashFile(warmupsFile) }));
  const report = await writeReport(directory);
  assert.equal(report.summaries['m0-r0-c0:warm'].overall.final_cer.rate, null);
  assert.equal(report.summaries['m0-r0-c0:warm'].overall.firstPartialMissingCount, 2);
  assert.match(await readFile(path.join(directory, 'summary.md'), 'utf8'), /n\/a/);
  await writeFile(resultsFile, '{}\n');
  await assert.rejects(writeReport(directory), /checksum/);
});
