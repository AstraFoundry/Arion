import assert from 'node:assert/strict';
import test from 'node:test';
import { candidateCoverage } from './candidate-ranges.mjs';

function result(input, text, end) {
  return { input, candidateRangeVersion: 1, candidateRangeInspectionMs: 0,
    beforeConfirmation: { page: 0, candidates: [text] },
    candidateRanges: [{ index: 0, text, start: 0, end, type: 'phrase' }] };
}
const protocol = { candidateRangeVersion: 1 };

test('matching text that leaves a suffix is not a full-input hit', () => {
  const actual = result('huochzehan', '火车站', 6);
  const partial = candidateCoverage(actual, protocol, '火车站');
  assert.equal(partial.fullInputRecall, 0);
  assert.equal(partial.fullInputTop1, 0);
  assert.equal(partial.partialOnlyRecall, 1);
  actual.candidateRanges[0].end = 10;
  const full = candidateCoverage(actual, protocol, '火车站');
  assert.equal(full.fullInputRecall, 1);
  assert.equal(full.fullInputTop1, 1);
  assert.equal(full.partialOnlyRecall, 0);
});

test('range observations must match snapshot identity and valid UTF-8 boundaries', () => {
  const original = result('中a', '中', 4);
  for (const mutate of [
    a => { a.candidateRanges[0].end = 1; },
    a => { a.candidateRanges[0].end = 5; },
    a => { a.candidateRanges[0].start = 4; },
    a => { a.candidateRanges[0].index = 1; },
    a => { a.candidateRanges[0].text = 'invented'; },
    a => { a.candidateRanges = []; },
    a => { a.candidateRangeInspectionMs = -1; },
    a => { delete a.candidateRangeVersion; },
  ]) {
    const changed = structuredClone(original); mutate(changed);
    assert.throws(() => candidateCoverage(changed, protocol, '中'));
  }
  assert.equal(candidateCoverage(original, protocol, '中').fullInputRecall, 1);
});

test('historical reports have no inferred range score; new protocol cannot be silently dropped', () => {
  assert.deepEqual(candidateCoverage({}, {}, 'target'), {});
  assert.throws(() => candidateCoverage(result('abc', 'target', 3), {}, 'target'));
  assert.throws(() => candidateCoverage(result('abc', 'target', 3), { candidateRangeVersion: 2 }, 'target'));
});
