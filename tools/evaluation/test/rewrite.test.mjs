import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, writeFile, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import { digest, hashFile } from '../src/manifest.mjs';
import { REWRITE_SCHEMA, validateRewriteCorpus, validateRewriteConfig, loadRewriteCorpus } from '../src/rewrite-manifest.mjs';
import { rewriteRequest, validateRewriteResponse } from '../src/rewrite-protocol.mjs';
import { rewriteMetrics, aggregateRewrite } from '../src/rewrite-metrics.mjs';
import { rewritePlan, runRewrite, checkedRewriteRows, reportRewrite } from '../src/rewrite-runner.mjs';
import { exportRewriteHuman, reportRewriteHuman } from '../src/rewrite-human.mjs';

const original = JSON.parse(await readFile(new URL('../../../benchmarks/configs/desktop-rewrite.json', import.meta.url), 'utf8'));
const smallConfig = () => ({ ...structuredClone(original), variants: original.variants.filter(v => ['rules-c0-m0-s0', 'llm-c0-m0-s0'].includes(v.id)), seeds: [11], warmupCount: 1, cacheStates: ['warm'], loadStates: ['solo'], bootstrap: { iterations: 100, seed: 1729 } });
function sample(id = 'case-a', split = 'test') {
  const sourceText = `${id}: do not send 12 files to Ada.`;
  return { id, split, clusterId: id, sessionId: id, contentId: id, language: 'en', inputKind: 'gold_text', sourceText, sourceSha256: digest(sourceText), tags: ['negation'], annotation: { kind: 'human_reviewed', path: `${id}-annotations.json`, sha256: 'a'.repeat(64) }, asrLoad: { path: `${id}.wav`, sha256: digest(id), durationMs: 1000 }, userInstruction: 'Clean up without changing meaning.', scenario: 'work-chat', context: { text: '', origin: 'user_approved', sourceSampleIds: [] }, approvedMemory: [], controls: [], expected: { references: [sourceText], protectedSpans: [{ kind: 'name', text: 'Ada', minimumOccurrences: 1 }, { kind: 'negation', text: 'not', minimumOccurrences: 1 }, { kind: 'number', text: '12', minimumOccurrences: 1 }], requiredStrings: [], forbiddenStrings: [], permittedNumbers: [], maxOutputChars: null, verbatim: false, refusalAllowed: false, attack: false, intent: 'Do not send the files.' } };
}
const corpusOf = samples => ({ version: 1, kind: REWRITE_SCHEMA, license: 'metric-oracle-only', consent: true, samples });
function response(q) {
  const llm = q.variant.features.engine === 'llm';
  return { protocol: 1, schema: REWRITE_SCHEMA, execution: 'live', requestId: q.requestId, identity: q.identity, effectiveFeatures: q.variant.features, effectivePipeline: q.pipeline, inputHash: q.inputHash, cacheState: q.cacheState, loadState: q.loadState, decision: 'unchanged', modelText: q.input.sourceText, candidateText: q.input.sourceText,
    timings: { requestStartMs: 0, loadMs: 0, prefillStartMs: 1, prefillEndMs: 2, firstTokenMs: llm ? 3 : null, decodeEndMs: 4, validationEndMs: 5, reviewMs: 6 },
    tokenEvents: llm ? [{ ms: 3, tokenId: 10, delta: q.input.sourceText }] : [], usage: { inputTokens: llm ? 8 : 0, outputTokens: llm ? 1 : 0 },
    previewEvents: [{ ms: 6, text: q.input.sourceText, identity: q.identity }], lifecycleEvents: [], commitEvents: [], residency: { reset: false, kvCacheReset: true, modelReloaded: false }, resources: { peakRssBytes: 1000, peakDeviceBytes: null, rssScope: 'process_request_sampled', rssSamplingIntervalMs: 10 }, asr: null };
}
const query = (s, engine = 'llm') => {
  const config = smallConfig(); const variant = config.variants.find(v => v.features.engine === engine);
  return { ...rewriteRequest(s, { seed: 11, cacheState: 'warm', loadState: 'solo', variant }, config, '/private/corpus.json'), requestId: '1' };
};

test('rewrite plan reproduces factorial/precision strata and rejects duplicate conditions', () => {
  validateRewriteConfig(original);
  assert.deepEqual(rewritePlan(original), rewritePlan(original));
  assert.equal(rewritePlan(original, [sample()]).measuredRequests, 216);
  const config = structuredClone(original); config.variants.push(config.variants[0]);
  assert.throws(() => validateRewriteConfig(config), /Unique/);
  config.variants.pop(); config.pipeline.thinking = true;
  assert.throws(() => validateRewriteConfig(config), /non-thinking/);
});

test('rewrite inputs withhold gold intent/reference and physically remove ablated context/memory/scenario', () => {
  const s = sample(); s.context.text = 'Private context'; s.approvedMemory = [{ id: 'm', text: 'Approved name', approved: true, origin: 'user_approved', sourceSampleIds: [] }];
  const q = query(s);
  assert.equal(q.input.context, ''); assert.deepEqual(q.input.approvedMemory, []); assert.equal(q.input.scenario, 'generic');
  assert.equal(q.inputHash, digest(q.input));
  for (const label of ['expected', 'references', 'protectedSpans', 'intent', 'inputKind']) assert.equal(Object.hasOwn(q, label), false);
  assert.equal(JSON.stringify(q).includes('Do not send the files.'), false);
});

test('rewrite split guards reject sibling documents and test-derived memory/context', () => {
  const a = sample('dev-a', 'dev'); const b = sample('test-b', 'test');
  validateRewriteCorpus(corpusOf([a, b]));
  b.contentId = a.contentId; assert.throws(() => validateRewriteCorpus(corpusOf([a, b])), /Cross-split/);
  b.contentId = b.id; a.context = { text: 'leaked', origin: 'dev', sourceSampleIds: [b.id] };
  assert.throws(() => validateRewriteCorpus(corpusOf([a, b])), /Test-derived/);
  a.context = { text: '', origin: 'user_approved', sourceSampleIds: [] }; b.expected.references = ['!!!'];
  assert.throws(() => validateRewriteCorpus(corpusOf([a, b])), /references/);
});

test('rewrite token/preview/cold identity gates cannot attest fabricated events', () => {
  const q = query(sample()); const r = response(q); validateRewriteResponse(r, q);
  r.tokenEvents[0].delta = 'different'; assert.throws(() => validateRewriteResponse(r, q), /Generated token/);
  r.tokenEvents[0].delta = r.modelText; r.previewEvents[0].identity = { ...q.identity, revision: 2 };
  assert.throws(() => validateRewriteResponse(r, q), /native preview/);
  r.previewEvents[0].identity = q.identity; q.cacheState = 'cold'; r.cacheState = 'cold';
  assert.throws(() => validateRewriteResponse(r, q), /release\/reset/);
  r.residency = { reset: true, kvCacheReset: true, modelReloaded: true }; validateRewriteResponse(r, q);
});

test('rewrite cancellation forbids late preview and any target commit', () => {
  const s = sample(); s.controls = [{ type: 'cancel', atMs: 2 }]; const q = query(s); const r = response(q);
  r.decision = 'cancelled'; r.candidateText = null; r.lifecycleEvents = [{ type: 'invalidated', ms: 2 }, { type: 'cancel_ack', ms: 4 }];
  assert.throws(() => validateRewriteResponse(r, q), /Stale preview/);
  r.previewEvents = []; validateRewriteResponse(r, q);
  r.commitEvents = [{ ms: 7 }]; assert.throws(() => validateRewriteResponse(r, q), /never commit/);
});

test('rewrite resources distinguish cumulative high-water marks from sampled request/worker peaks', () => {
  const s = sample(); const q = query(s); const r = response(q);
  delete r.resources.rssScope;
  assert.throws(() => validateRewriteResponse(r, q), /measurement scope/);
  r.resources.rssScope = 'process_high_water';
  assert.throws(() => validateRewriteResponse(r, q), /sampling interval/);
  r.resources.rssSamplingIntervalMs = null;
  validateRewriteResponse(r, q);
  const summary = aggregateRewrite([{ metrics: rewriteMetrics(s, r) }]);
  assert.deepEqual(summary.rssMeasurementScopes, ['process_high_water']);
  assert.equal(summary.rssSamplingIntervalMs.count, 0);
});

test('concurrent rewrite requires ASR duration, overlap and measured queue/sample traces', () => {
  const q = query(sample()); q.loadState = 'concurrent_asr'; q.asrLoad = sample().asrLoad;
  const r = response(q); r.asr = { execution: 'live', audioSha256: q.asrLoad.sha256, model: q.pipeline.asr, startedMs: 0, endedMs: 1000, events: [{ ms: 500, text: 'speech' }], processedSamples: 16000, droppedSamples: 0, maxQueueAgeMs: 3 };
  validateRewriteResponse(r, q);
  r.asr.endedMs = 100; assert.throws(() => validateRewriteResponse(r, q), /overlap/);
  r.asr.endedMs = 1000; r.asr.processedSamples = 0; assert.throws(() => validateRewriteResponse(r, q), /queue\/sample/);
});

// Known text/count oracles, never model performance evidence.
test('rewrite metrics expose changed negation/number, attack success and reject-all coverage', () => {
  const s = sample(); s.expected.attack = true; s.expected.forbiddenStrings = ['ATTACK']; s.expected.requiredStrings = ['Ada'];
  const q = query(s); const r = response(q); r.candidateText = 'Send 13 files. ATTACK'; r.modelText = r.candidateText;
  const m = rewriteMetrics(s, r);
  assert.equal(m.protectedPreserved, 0); assert.equal(m.protectedCount, 3); assert.equal(m.constraintViolations, 2); assert.equal(m.unsupportedNumberCount, 1); assert.equal(m.attackMarkerObserved, true);
  const failed = { ...r, candidateText: null, decision: 'rejected' };
  const report = aggregateRewrite([{ metrics: rewriteMetrics(s, failed) }]);
  assert.equal(report.acceptanceRate, 0); assert.equal(report.protectedDeliveryRecall, 0); assert.equal(report.protectedSpanRecallAccepted, null);
  assert.equal(report.semanticHallucination, 'requires_blinded_human_review');
});

test('cancelled rewrite fixtures cannot lower normal latency or inflate preview stability', () => {
  const a = sample('normal'); const qa = query(a); const ra = response(qa); ra.timings.reviewMs = 100; ra.previewEvents[0].ms = 100;
  const b = sample('cancelled'); b.controls = [{ type: 'cancel', atMs: 2 }]; const qb = query(b); const rb = response(qb);
  rb.decision = 'cancelled'; rb.candidateText = null; rb.previewEvents = []; rb.lifecycleEvents = [{ type: 'invalidated', ms: 2 }, { type: 'cancel_ack', ms: 4 }];
  const summary = aggregateRewrite([{ metrics: rewriteMetrics(a, ra) }, { metrics: rewriteMetrics(b, rb) }]);
  assert.equal(summary.count, 2); assert.equal(summary.eligibleCount, 1); assert.equal(summary.reviewMs.p50, 100); assert.equal(summary.firstTokenMs.count, 1); assert.equal(summary.cancelAckMs.p50, 2);
});

test('rewrite loader checks reviewed annotation/audio hashes and CLI refuses missing drivers/overwrite', async t => {
  const dir = await mkdtemp(path.join(tmpdir(), 'arion-rewrite-loader-')); t.after(() => rm(dir, { recursive: true, force: true }));
  const s = sample();
  for (const a of [s.annotation, s.asrLoad]) { await writeFile(path.join(dir, a.path), 'metadata-oracle-not-audio'); a.sha256 = await hashFile(path.join(dir, a.path)); }
  const file = path.join(dir, 'corpus.json'); await writeFile(file, JSON.stringify(corpusOf([s]))); await loadRewriteCorpus(file);
  await writeFile(path.join(dir, s.annotation.path), 'changed'); await assert.rejects(loadRewriteCorpus(file), /checksum/);
  const cli = fileURLToPath(new URL('../rewrite-cli.mjs', import.meta.url));
  const config = fileURLToPath(new URL('../../../benchmarks/configs/desktop-rewrite.json', import.meta.url));
  const missing = spawnSync(process.execPath, [cli, 'run', '--config', config], { encoding: 'utf8' }); assert.equal(missing.status, 1); assert.match(missing.stderr, /corpus.*required/);
  const args = [cli, 'plan', '--config', config, '--out', path.join(dir, 'plan.json')]; assert.equal(spawnSync(process.execPath, args).status, 0); assert.equal(spawnSync(process.execPath, args).status, 1);
});

test('rewrite runner rejects fixture execution and preserves an unreportable failed run', async t => {
  const dir = await mkdtemp(path.join(tmpdir(), 'arion-rewrite-failure-')); t.after(() => rm(dir, { recursive: true, force: true }));
  const config = smallConfig(); const artifact = path.join(dir, 'artifact.txt'); await writeFile(artifact, 'fixture-not-a-model'); const sha256 = await hashFile(artifact);
  const driverFile = path.join(dir, 'fixture.mjs'); await writeFile(driverFile, "import {createInterface} from 'node:readline'; createInterface({input:process.stdin}).on('line',line=>{const r=JSON.parse(line); console.log(JSON.stringify({protocol:1,requestId:r.requestId,execution:'fixture',schema:r.schema,artifacts:r.artifacts}));});\n");
  const driver = { executable: process.execPath, args: [driverFile], timeoutMs: 1000, artifacts: ['model', 'prompt', 'tokenizer', 'asr'].map(k => ({ id: config.pipeline[k], path: artifact, sha256, license: 'test-only' })) };
  const corpus = corpusOf([sample('dev-a', 'dev'), sample('test-a', 'test')]); const out = path.join(dir, 'run');
  await assert.rejects(runRewrite(config, corpus, path.join(dir, 'corpus.json'), driver, dir, out), /attest live/);
  assert.equal(JSON.parse(await readFile(path.join(out, 'run.json'), 'utf8')).status, 'failed');
  await assert.rejects(reportRewrite(out), /Incomplete/);
  await assert.rejects(runRewrite(config, corpus, path.join(dir, 'corpus.json'), driver, dir, out), /EEXIST/);
});

// Synthetic serialization oracle, not live inference: no driver executes in this test.
test('rewrite report recomputes metrics, enforces coverage and blind human ratings', async t => {
  const dir = await mkdtemp(path.join(tmpdir(), 'arion-rewrite-report-')); t.after(() => rm(dir, { recursive: true, force: true }));
  const config = smallConfig(); const samples = [sample('case-a'), sample('case-b')]; const warmup = sample('dev-a', 'dev'); const corpus = corpusOf([warmup, ...samples]);
  const plan = rewritePlan(config, samples);
  const rows = plan.blocks.flatMap(block => samples.map(s => { const q = { ...rewriteRequest(s, block, config, path.join(dir, 'corpus.json')), requestId: `${s.id}-${block.variant.id}` }; return { caseId: s.id, clusterId: s.clusterId, language: s.language, inputKind: s.inputKind, tags: s.tags, unit: 'rewrite', seed: block.seed, cacheState: block.cacheState, plannedCacheState: block.cacheState, warmup: false, loadState: block.loadState, variantId: block.variant.id, metrics: { protectedPreserved: -100 }, response: response(q) }; }));
  const warmRows = plan.blocks.map(block => { const q = { ...rewriteRequest(warmup, { ...block, cacheState: 'warm' }, config, path.join(dir, 'corpus.json'), true), requestId: `warm-${block.variant.id}` }; return { caseId: warmup.id, clusterId: warmup.clusterId, seed: block.seed, cacheState: 'warm', plannedCacheState: block.cacheState, warmup: true, loadState: block.loadState, variantId: block.variant.id, response: response(q) }; });
  const results = path.join(dir, 'results.jsonl'); const warmups = path.join(dir, 'warmups.jsonl');
  await writeFile(results, rows.map(r => JSON.stringify(r)).join('\n') + '\n'); await writeFile(warmups, warmRows.map(r => JSON.stringify(r)).join('\n') + '\n'); await writeFile(path.join(dir, 'corpus.json'), JSON.stringify(corpus));
  const metadata = { version: 1, kind: REWRITE_SCHEMA, status: 'complete', cleanupConfirmed: true, config, configHash: digest(config), corpusHash: digest(corpus), plan, observations: rows.length, resultsSha256: await hashFile(results), warmupsSha256: await hashFile(warmups) };
  await writeFile(path.join(dir, 'run.json'), JSON.stringify(metadata));
  const report = await reportRewrite(dir); assert.equal(report.summaries['rules-c0-m0-s0:warm:solo'].overall.protectedDeliveryRecall, 1); assert.equal(report.semanticQuality, 'pending_blinded_human_review');
  const review = path.join(dir, 'review'); await exportRewriteHuman(dir, review, 2, 11);
  const tasks = JSON.parse(await readFile(path.join(review, 'tasks.json'), 'utf8')); assert.equal(JSON.stringify(tasks).includes('variantId'), false);
  const ratings = { version: 1, kind: 'arion-rewrite-human-v1', ratings: tasks.tasks.flatMap(task => ['r1', 'r2'].map(raterId => ({ taskId: task.taskId, raterId, meaning: 5, style: 4, fluency: 5, hallucination: false, unnecessaryRefusal: false }))) };
  const ratingsFile = path.join(review, 'ratings.json'); await writeFile(ratingsFile, JSON.stringify(ratings));
  const human = await reportRewriteHuman(dir, review, ratingsFile); assert.equal(human.tasks, 4); assert.equal(human.summaries['llm-c0-m0-s0:warm:solo'].meanScores.meaning, 5);
  ratings.ratings.pop(); await writeFile(ratingsFile, JSON.stringify(ratings)); await assert.rejects(reportRewriteHuman(dir, review, ratingsFile), /Incomplete/);
  rows[0].response.identity.revision = 2; await writeFile(results, rows.map(r => JSON.stringify(r)).join('\n') + '\n');
  await assert.rejects(checkedRewriteRows(dir), /checksum/);
  metadata.resultsSha256 = await hashFile(results); await writeFile(path.join(dir, 'run.json'), JSON.stringify(metadata));
  await assert.rejects(checkedRewriteRows(dir), /Stale/);
  rows[0].response.identity.revision = 1; await writeFile(results, rows.map(r => JSON.stringify(r)).join('\n') + '\n'); metadata.resultsSha256 = await hashFile(results);
  await writeFile(warmups, ''); metadata.warmupsSha256 = await hashFile(warmups); await writeFile(path.join(dir, 'run.json'), JSON.stringify(metadata));
  await assert.rejects(checkedRewriteRows(dir), /warmup coverage/);
});
