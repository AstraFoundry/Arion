import test from 'node:test';
import assert from 'node:assert/strict';
import { reviewedEvaluation } from '../catalog-evaluation.mjs';
const catalog = { version: 1, bundles: [{ id: 'model-a', evaluations: [] }] };
const input = () => ({ schema: 1, status: 'complete', runId: 'example-run', source: {
  productCommit: 'a'.repeat(40), evaluatorCommit: 'b'.repeat(40), dirty: false,
  corpusHash: 'c'.repeat(64), configHash: 'd'.repeat(64), resultsSha256: 'e'.repeat(64), summarySha256: 'f'.repeat(64),
}, evaluations: [{ bundleId: 'model-a', hardware: 'macos-arm64', split: 'test', observations: 20,
  errorRates: { zh: 0.1 }, stopToReviewP95Ms: 100, firstPartialP95Ms: 40, realTimeFactor: 0.2 }] });
test('reviewed aggregates retain exact identities and omit all raw fields', () => {
  const value = input();
  value.transcript = 'private transcript';
  value.source.privatePath = '/private/recording.wav';
  value.evaluations[0].observationsRaw = ['private reference'];
  const result = reviewedEvaluation(catalog, value);
  assert.equal(result.summary.source.productCommit, value.source.productCommit);
  assert.equal(result.summary.source.evaluatorCommit, value.source.evaluatorCommit);
  assert.equal(result.catalog.bundles[0].evaluations[0].report, 'models/evaluations/example-run.json');
  assert.doesNotMatch(JSON.stringify(result), /private transcript|private reference|recording.wav/);
  assert.deepEqual(catalog.bundles[0].evaluations, []);
  assert.deepEqual(reviewedEvaluation(result.catalog, value), result);
});
test('invalid, dirty, incomplete or unbound aggregates cannot modify the catalog', () => {
  for (const mutate of [v => { v.source.dirty = true; }, v => { delete v.source.productCommit; },
    v => { delete v.source.evaluatorCommit; }, v => { v.status = 'cancelled'; },
    v => { v.runId = '../escape'; }, v => { v.evaluations[0].bundleId = 'unknown'; },
    v => { v.evaluations[0].realTimeFactor = NaN; }, v => { v.evaluations.push(v.evaluations[0]); }]) {
    const value = input();
    mutate(value);
    assert.throws(() => reviewedEvaluation(catalog, value));
    assert.deepEqual(catalog.bundles[0].evaluations, []);
  }
});
