// Aggregate explicit semantic annotations; no automatic semantic oracle is used.
import fs from 'node:fs';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const read = file => JSON.parse(fs.readFileSync(file, 'utf8'));
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
const fileHash = file => hash(fs.readFileSync(file));
const assert = (ok, message) => { if (!ok) throw new Error(message); };
export function validateRatings(original, review) {
  const outputs = new Map(original.rows.map(r => [`${r.caseId}/${r.arm}`, r]));
  assert(outputs.size === original.rows.length, 'Duplicate original output');
  assert(review.ratings.length === outputs.size, 'Incomplete semantic review');
  const seen = new Set();
  for (const rating of review.ratings) {
    const key = `${rating.caseId}/${rating.arm}`;
    assert(!seen.has(key), 'Duplicate semantic rating'); seen.add(key);
    const output = outputs.get(key);
    assert(output && hash(Buffer.from(output.text)) === rating.textSha256, 'Changed or unknown reviewed text');
    assert(['preserved', 'degraded', 'uncertain'].includes(rating.fidelity), 'Unknown fidelity');
    assert([0, 1, 2].includes(rating.organization), 'Unknown organization rating');
    assert(rating.ready === (rating.fidelity === 'preserved' && rating.organization >= 1), 'Inconsistent readiness');
    assert(typeof rating.reason === 'string' && rating.reason.length > 20, 'Concrete rationale required');
  }
  return outputs;
}
function aggregate(rows) {
  return { cases: rows.length, preserved: rows.filter(r => r.fidelity === 'preserved').length,
    degraded: rows.filter(r => r.fidelity === 'degraded').length, uncertain: rows.filter(r => r.fidelity === 'uncertain').length,
    ready: rows.filter(r => r.ready).length,
    organization: Object.fromEntries([0, 1, 2].map(score => [score, rows.filter(r => r.organization === score).length])) };
}
export function report() {
  const originalPath = path.join(root, 'benchmarks/results/correction-chain-v1/summary.json');
  const directory = path.join(root, 'benchmarks/results/correction-chain-semantic-v1');
  const review = read(path.join(directory, 'ratings.json'));
  const original = read(originalPath);
  assert(fileHash(originalPath) === review.summarySha256, 'Original summary hash mismatch');
  assert(fileHash(path.join(root, 'benchmarks/reproduction/correction-chain-semantic-v1/rubric.json')) === review.rubricSha256, 'Rubric hash mismatch');
  const outputs = validateRatings(original, review);
  const arms = [...new Set(original.rows.map(r => r.arm))];
  const paired = {};
  for (const [base, treatment] of [['asr', 'asr_correct'], ['asr_llm', 'asr_correct_llm'], ['asr_llm', 'offline_llm']]) {
    const pairs = review.ratings.filter(r => r.arm === base).map(a => [a, review.ratings.find(b => b.caseId === a.caseId && b.arm === treatment)]);
    paired[`${base} -> ${treatment}`] = {
      fidelityGained: pairs.filter(([a, b]) => a.fidelity !== 'preserved' && b.fidelity === 'preserved').map(([a]) => a.caseId),
      fidelityLost: pairs.filter(([a, b]) => a.fidelity === 'preserved' && b.fidelity !== 'preserved').map(([a]) => a.caseId),
      readyGained: pairs.filter(([a, b]) => !a.ready && b.ready).map(([a]) => a.caseId),
      readyLost: pairs.filter(([a, b]) => a.ready && !b.ready).map(([a]) => a.caseId),
    };
  }
  const reclassified = review.ratings.map(r => ({ ...r, oldPass: outputs.get(`${r.caseId}/${r.arm}`).intent.pass }))
    .filter(r => r.oldPass !== r.ready);
  return { version: 1, kind: 'agent-semantic-reassessment-v1', originalSummarySha256: review.summarySha256,
    ratingsSha256: fileHash(path.join(directory, 'ratings.json')), rubricSha256: review.rubricSha256,
    scorerSha256: fileHash(fileURLToPath(import.meta.url)), reviewer: review.reviewer, independence: review.independence,
    arms: Object.fromEntries(arms.map(arm => [arm, {
      all: aggregate(review.ratings.filter(r => r.arm === arm)),
      spokenOnly: aggregate(review.ratings.filter(r => r.arm === arm && !r.caseId.includes('silence'))),
      byLanguage: Object.fromEntries(['zh', 'en', 'mixed'].map(language => [language, aggregate(review.ratings.filter(r => r.arm === arm && outputs.get(`${r.caseId}/${r.arm}`).language === language))]))
    }])), paired, reclassified,
    sensitivity: 'The pool-request spelling and phonetic database name judgments are subjective boundary cases; see per-case reasons. No population significance or independent human agreement is established.' };
}
if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const output = process.argv[2];
  if (!output) throw new Error('Usage: report.mjs NEW_OUTPUT_JSON');
  const value = report(); fs.writeFileSync(output, JSON.stringify(value, null, 2) + '\n', { flag: 'wx' });
  console.log(JSON.stringify({ arms: Object.fromEntries(Object.entries(value.arms).map(([arm, result]) => [arm, result.all])), paired: value.paired }, null, 2));
}
