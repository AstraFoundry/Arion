import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { validateRatings } from './report.mjs';
const hash = text => createHash('sha256').update(text).digest('hex');
function fixture() {
  return { original: { rows: [{ caseId: 'case', arm: 'arm', text: 'Recipient: Robert' }] },
    review: { ratings: [{ caseId: 'case', arm: 'arm', textSha256: hash('Recipient: Robert'), fidelity: 'preserved', organization: 2, ready: true, reason: 'Final recipient Robert is preserved in a clear structured form.' }] } };
}
test('semantic annotation accepts structured paraphrases without exact reference matching', () => {
  const { original, review } = fixture();
  assert.equal(validateRatings(original, review).size, 1);
});
test('semantic review rejects changed output bytes', () => {
  const { original, review } = fixture(); original.rows[0].text = 'Recipient: Helen';
  assert.throws(() => validateRatings(original, review), /Changed/);
});
test('semantic review rejects missing and duplicate ratings', () => {
  const { original, review } = fixture(); review.ratings = [];
  assert.throws(() => validateRatings(original, review), /Incomplete/);
  const f = fixture(); f.original.rows.push({ caseId: 'case2', arm: 'arm', text: 'Other' });
  f.review.ratings.push({ ...f.review.ratings[0] });
  assert.throws(() => validateRatings(f.original, f.review), /Duplicate/);
});
test('wrong intent cannot receive ready status merely because prose is fluent', () => {
  const { original, review } = fixture(); review.ratings[0].fidelity = 'degraded';
  assert.throws(() => validateRatings(original, review), /Inconsistent/);
});
