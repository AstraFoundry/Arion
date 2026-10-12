import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, readFile, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { Driver, validateResponse } from '../src/protocol.mjs';
import { digest, validateManifest, validateExperiment } from '../src/manifest.mjs';
import { inferenceRequest, plan } from '../src/runner.mjs';
import { buildReport } from '../src/report.mjs';
import { assertObservations } from '../src/e2e.mjs';
import { NORMALIZATION, scoreSample } from '../src/metrics.mjs';

const config = JSON.parse(await readFile(new URL('../../../benchmarks/configs/desktop-factorial.json', import.meta.url), 'utf8'));
const suite = JSON.parse(await readFile(new URL('../../../tests/e2e/desktop-scenarios.json', import.meta.url), 'utf8'));
const sample = { id: 'a', split: 'test', speakerId: 's1', sessionId: 'session1', language: 'en', tags: ['clean'], unit: 'wer', reference: 'one cat', context: '', approvedMemory: [], audio: { path: 'a.wav', sha256: 'a'.repeat(64), durationMs: 1000 } };
const request = { protocol: 1, requestId: '1', cacheState: 'warm', variant: config.variants[0], pipeline: config.pipeline };
// Protocol conformance data deliberately does not execute any speech engine.
const response = { protocol: 1, requestId: '1', execution: 'live', effectiveFeatures: request.variant.features, effectivePipeline: request.pipeline, cacheState: 'warm', rawText: 'one cat', selectedText: 'one cat', finalText: 'one cat', events: [{ text: 'one', ms: 100 }, { text: 'one cat', ms: 1050, final: true }], candidates: [{ id: 'x', text: 'one cat', source: 'test-oracle' }], rankedIds: ['x'], timings: { speechStartMs: 0, speechEndMs: 1000, firstPartialMs: 100, finalMs: 1050, reviewMs: 1060, rankerMs: null, loadMs: 0, computeMs: 400 }, resources: { peakRssBytes: 10000 } };

test('manifest rejects speaker/audio split leakage and reference-derived memory', () => {
  const manifest = { version: 1, license: 'synthetic-testing-only', consent: true, samples: [sample] };
  assert.equal(validateManifest(manifest), manifest);
  assert.throws(() => validateManifest({ ...manifest, samples: [sample, { ...sample, id: 'b', split: 'dev' }] }), /leakage/);
  assert.throws(() => validateManifest({ ...manifest, samples: [{ ...sample, approvedMemory: [{ id: 'm', text: 'cat', approved: true, origin: 'test', sourceSampleIds: ['a'] }] }] }), /test-derived/);
  assert.throws(() => validateManifest({ ...manifest, samples: [{ ...sample, protectedSpans: [{ kind: 'number', text: '42' }] }] }), /protected-span/);
  assert.throws(() => validateManifest({ ...manifest, consent: false }), /consent/);
});

test('factorial plan is deterministic, unique and contains all controlled factors', () => {
  assert.equal(validateExperiment(config), config);
  const first = plan(config, [sample]);
  assert.equal(first.blocks.length, 48);
  assert.equal(first.measuredRequests, 48);
  assert.deepEqual(first, plan(config, [sample]));
  assert.equal(new Set(first.blocks.map(block => `${block.variant.id}:${block.seed}:${block.cacheState}`)).size, 48);
  assert.equal(new Set(config.variants.map(v => digest(v.features))).size, 8);
  assert.throws(() => validateExperiment({ ...config, baseline: 'missing' }), /Baseline/);
  assert.throws(() => validateExperiment({ ...config, variants: [config.variants[0], config.variants[0]] }), /Duplicate/);
});

test('inference boundary withholds labels and removes memory in ablation', () => {
  const withMemory = { ...sample, approvedMemory: [{ id: 'm', text: 'cat', approved: true, origin: 'user_approved', sourceSampleIds: [] }], protectedSpans: [{ kind: 'name', text: 'cat' }], acceptableCandidates: ['one cat'] };
  const payload = inferenceRequest(withMemory, { variant: config.variants[0], seed: 11, cacheState: 'warm' }, config, '/tmp/corpus.json');
  assert.equal(JSON.stringify(payload).includes('reference'), false);
  assert.equal(JSON.stringify(payload).includes('acceptableCandidates'), false);
  assert.equal(JSON.stringify(payload).includes('protectedSpans'), false);
  assert.deepEqual(payload.input.approvedMemory, []);
  assert.equal(payload.input.audio.path, path.resolve('/tmp/a.wav'));
});

test('driver outputs reject mock execution, stale IDs, mismatched features and traces', () => {
  assert.equal(validateResponse(response, request), response);
  assert.throws(() => validateResponse({ ...response, execution: 'fixture' }, request), /live/);
  assert.throws(() => validateResponse({ ...response, requestId: 'old' }, request), /identity/);
  assert.throws(() => validateResponse({ ...response, effectiveFeatures: { ...response.effectiveFeatures, memory: true } }, request), /ablation/);
  assert.throws(() => validateResponse({ ...response, events: [{ text: 'one cat', ms: 1050 }] }, request), /final ASR/);
  assert.throws(() => validateResponse({ ...response, rankedIds: ['invented'] }, request), /permutation/);
  assert.throws(() => validateResponse({ ...response, timings: { ...response.timings, firstPartialMs: 200 } }, request), /event trace/);
  assert.throws(() => validateResponse({ ...response, finalText: 'two cats' }, request), /Correction disabled/);
});

test('reports reject incomplete, duplicated or mismatched candidate observations', () => {
  const cases = [sample, { ...sample, id: 'b', speakerId: 's2' }];
  const rows = plan(config, cases).blocks.flatMap(block => cases.map(s => ({ caseId: s.id, clusterId: s.speakerId, unit: s.unit, language: s.language, tags: s.tags, seed: block.seed, cacheState: block.cacheState, variantId: block.variant.id, candidateHash: 'same-producer', metrics: scoreSample(s, response) })));
  const metadata = { status: 'complete', config, normalization: NORMALIZATION, configHash: digest(config), corpusHash: 'synthetic-metric-test', plan: plan(config, cases), observations: rows.length };
  const report = buildReport(metadata, rows);
  assert.equal(report.observations, 96);
  assert.equal(report.summaries['m0-r0-c0:warm'].overall.final_wer.rate, 0);
  assert.equal(report.rankerPairs[0].status, 'ok');
  assert.throws(() => buildReport({ ...metadata, status: 'running' }, rows), /Incomplete/);
  assert.throws(() => buildReport(metadata, rows.slice(1)), /coverage/);
  assert.throws(() => buildReport(metadata, [rows[0], ...rows.slice(0, -1)]), /Duplicate/);
  const mismatch = rows.map(row => row.variantId === 'm0-r1-c0' ? { ...row, candidateHash: 'changed' } : row);
  assert.equal(buildReport(metadata, mismatch).rankerPairs.find(pair => pair.treatment === 'm0-r1-c0').status, 'candidate_set_mismatch');
});

test('E2E assertions cannot pass with missing or self-reported success alone', () => {
  const scenario = suite.scenarios[0];
  assert.throws(() => assertObservations(scenario, { status: 'passed' }), /assertion/);
  assertObservations(scenario, { editors: { A: { text: 'Hello world' } }, commitCount: 1, network: { outboundRequests: 0 }, session: { state: 'completed' } });
  assert.equal(new Set(suite.scenarios.map(s => s.id)).size, suite.scenarios.length);
});

test('persistent driver transport handles multiple messages and rejects unsolicited output', async t => {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-driver-test-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const driverFile = path.join(directory, 'echo.mjs');
  await writeFile(driverFile, "import {createInterface} from 'node:readline';\nconst lines = createInterface({input:process.stdin});\nlines.on('line',line=>{const r=JSON.parse(line); console.log(JSON.stringify({protocol:1,requestId:r.requestId,execution:'fixture',op:r.op}));});\n");
  const driver = new Driver({ executable: process.execPath, args: [driverFile], timeoutMs: 1000 }, directory);
  t.after(() => driver.close());
  assert.equal((await driver.request({ op: 'hello' })).execution, 'fixture');
  assert.equal((await driver.request({ op: 'infer' })).requestId, '2');
  const brokenFile = path.join(directory, 'broken.mjs');
  await writeFile(brokenFile, "console.log('not JSON'); setTimeout(()=>{},1000);\n");
  const broken = new Driver({ executable: process.execPath, args: [brokenFile], timeoutMs: 1000 }, directory);
  t.after(() => broken.close());
  await assert.rejects(broken.request({ op: 'hello' }), /JSONL/);
});
