import test from 'node:test';
import assert from 'node:assert/strict';
import { aggregate, edits, errorCounts, erasure, normalize, quantile, ranking, scoreSample, tokens } from '../src/metrics.mjs';
import { pairedBootstrap, shuffle } from '../src/statistics.mjs';

// These are metric oracles, not ASR outputs or claims of model quality.
const sample = { reference: 'keep 42', unit: 'wer', audio: { durationMs: 1000 }, protectedSpans: [{ kind: 'number', text: '42' }] };
const response = { rawText: 'keep 42', selectedText: 'keep 42', finalText: 'keep 24', candidates: [{ id: 'a', text: 'keep 42', source: 'fixture' }], rankedIds: ['a'], events: [{ text: 'keep', ms: 100 }, { text: 'keep 42', ms: 1000, final: true }], timings: { firstPartialMs: 100, speechStartMs: 0, speechEndMs: 900, finalMs: 1000, reviewMs: 1050, rankerMs: null, computeMs: 500, loadMs: 0 }, resources: { peakRssBytes: 1024 } };

test('normalization, CER and MER have explicit Unicode units', () => {
  assert.equal(normalize('ＡＢＣ，你好!'), 'abc 你好');
  assert.deepEqual(tokens('打开 API v2。', 'mer'), ['打', '开', 'api', 'v2']);
  assert.equal(errorCounts('甲乙', '甲丙', 'cer').errors, 1);
  assert.deepEqual(tokens('a 😀 b', 'wer'), ['a', 'b']);
});

test('known Levenshtein oracles include insertions above 100%', () => {
  assert.equal(edits([... 'kitten'], [... 'sitting']).errors, 3);
  assert.deepEqual(edits([], ['x', 'y']), { substitutions: 0, deletions: 0, insertions: 2, errors: 2, referenceUnits: 0 });
  assert.equal(errorCounts('a', 'a b c d', 'wer').errors, 3);
  assert.equal(edits(['a', 'b'], []).deletions, 2);
});

test('corpus rates are micro-averaged and silence has its own denominator', () => {
  const metrics = scoreSample(sample, response);
  assert.equal(metrics.harmfulCorrection, true);
  assert.equal(metrics.protectedPreserved, 0);
  const other = { ...metrics, raw: { errors: 0, referenceUnits: 8 }, selected: { errors: 0, referenceUnits: 8 }, final: { errors: 0, referenceUnits: 8 } };
  const summary = aggregate([{ unit: 'wer', metrics }, { unit: 'wer', metrics: other }]);
  assert.equal(summary.final_wer.rate, 0.1);
  assert.equal(summary.final_cer.rate, null);
  assert.equal(summary.corpusRtf, 0.5);
  const silence = scoreSample({ ...sample, reference: '', protectedSpans: [] }, { ...response, rawText: '', selectedText: '', finalText: 'hello' });
  const silentSummary = aggregate([{ unit: 'wer', metrics: silence }]);
  assert.equal(silentSummary.final_wer.rate, null);
  assert.equal(silentSummary.silence.falseEmissionRate, 1);
  assert.equal(scoreSample({ ...sample, reference: '', protectedSpans: [] }, { ...response, finalText: '!!!' }).falseEmission, true);
});

test('normalized erasure counts suffix withdrawal including final revisions', () => {
  const result = erasure([{ text: 'one cat' }, { text: 'one dog' }, { text: 'one dog runs' }], 'wer');
  assert.deepEqual(result, { erased: 1, finalUnits: 3, normalized: 1 / 3 });
  assert.equal(erasure([{ text: 'a b' }, { text: 'a' }], 'wer').normalized, 1);
  assert.equal(erasure([{ text: 'a' }, { text: '' }], 'wer').normalized, null);
});

test('rank accuracy does not hide missing producer oracle candidates', () => {
  const candidates = [{ id: 'a', text: '张维', source: 'lexicon' }, { id: 'b', text: '张伟', source: 'lexicon' }];
  assert.deepEqual(ranking(candidates, ['a', 'b'], ['张伟'], 'cer'), { eligible: 1, oracle: 1, top1: 0, reciprocalRank: 0.5 });
  assert.equal(ranking(candidates, ['a', 'b'], ['王伟'], 'cer').oracle, 0);
  assert.throws(() => ranking(candidates, ['a', 'a'], ['张伟'], 'cer'), /permutation/);
  assert.throws(() => ranking(candidates, ['a', 'invented'], ['张伟'], 'cer'), /permutation/);
});

test('type-7 quantiles interpolate and reject nonfinite values', () => {
  assert.equal(quantile([0, 10], 0.95), 9.5);
  assert.equal(quantile([], 0.95), null);
  assert.throws(() => quantile([NaN], 0.5), /finite/);
});

test('paired cluster bootstrap preserves repeats and known constant treatment delta', () => {
  const baseline = [{ caseId: 'a', clusterId: 's1', seed: 11, cacheState: 'warm', unit: 'wer', value: 2 }, { caseId: 'b', clusterId: 's2', seed: 11, cacheState: 'warm', unit: 'wer', value: 4 }, { caseId: 'a', clusterId: 's1', seed: 23, cacheState: 'warm', unit: 'wer', value: 2 }];
  const treatment = baseline.map(row => ({ ...row, value: row.value - 1 }));
  const mean = rows => rows.reduce((n, row) => n + row.value, 0) / rows.length;
  const result = pairedBootstrap(baseline, treatment, mean, { iterations: 200, seed: 42 });
  assert.ok(Math.abs(result.delta + 1) < 1e-12);
  assert.ok(result.ci95.every(value => Math.abs(value + 1) < 1e-12));
  assert.equal(result.clusters, 2);
  assert.deepEqual(result, pairedBootstrap(baseline, treatment, mean, { iterations: 200, seed: 42 }));
  assert.throws(() => pairedBootstrap(baseline, treatment.slice(1), mean), /Unpaired/);
  assert.equal(pairedBootstrap([baseline[0]], [treatment[0]], mean).status, 'insufficient_clusters');
  assert.deepEqual(shuffle([1, 2, 3, 4], 42), shuffle([1, 2, 3, 4], 42));
});
