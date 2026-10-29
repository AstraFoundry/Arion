// Validate observed engine coverage separately from text-only oracle metrics.
import assert from 'node:assert/strict';

export function candidateCoverage(actual, metadata, reference, normalize = value => value) {
  if (metadata.candidateRangeVersion === undefined) {
    assert.equal(actual.candidateRangeVersion, undefined, 'Range protocol must be declared in frozen metadata.');
    return {};
  }
  assert.equal(metadata.candidateRangeVersion, 1);
  assert.equal(actual.candidateRangeVersion, 1);
  const texts = actual.beforeConfirmation.candidates, ranges = actual.candidateRanges;
  assert.equal(actual.beforeConfirmation.page, 0);
  assert.ok(Array.isArray(ranges) && ranges.length === texts.length && texts.length <= 9);
  assert.ok(Number.isFinite(actual.candidateRangeInspectionMs) && actual.candidateRangeInspectionMs >= 0);
  const bytes = Buffer.byteLength(actual.input);
  const boundaries = new Set([0]);
  let offset = 0;
  for (const character of actual.input) { offset += Buffer.byteLength(character); boundaries.add(offset); }
  for (const [index, candidate] of ranges.entries()) {
    assert.equal(candidate.index, index);
    assert.equal(candidate.text, texts[index]);
    assert.equal(typeof candidate.type, 'string');
    assert.ok(Number.isInteger(candidate.start) && Number.isInteger(candidate.end));
    assert.ok(candidate.start >= 0 && candidate.start < candidate.end && candidate.end <= bytes);
    assert.ok(boundaries.has(candidate.start) && boundaries.has(candidate.end));
  }
  const target = normalize(reference);
  const matches = ranges.filter(candidate => normalize(candidate.text) === target);
  const full = candidate => candidate.start === 0 && candidate.end === bytes;
  const first = ranges[0];
  return {
    fullInputRecall: Number(matches.some(full)),
    partialOnlyRecall: Number(matches.length > 0 && !matches.some(full)),
    fullInputTop1: Number(!!first && normalize(first.text) === target && full(first)),
    candidateRanges: ranges,
  };
}
