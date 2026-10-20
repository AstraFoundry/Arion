// Scores raw driver results without native code: target-word rank, top-1 and
// selection keys by condition and occurrence, plus timing percentiles.
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const percentile = (values, p) => {
  if (values.length === 0) return null;
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.min(sorted.length - 1, Math.ceil(p / 100 * sorted.length) - 1)];
};
const round = value => value === null ? null : Math.round(value * 1000) / 1000;
const timing = values => ({ count: values.length, p50: round(percentile(values, 50)), p95: round(percentile(values, 95)),
  p99: round(percentile(values, 99)), max: round(Math.max(...values)) });

function score(steps) {
  const n = steps.length;
  const committed = steps.filter(s => s.committed).length;
  const top1 = steps.filter(s => s.committed && s.rank === 0).length;
  const reciprocal = steps.reduce((sum, s) => sum + (s.rank >= 0 ? 1 / (s.rank + 1) : 0), 0);
  const keys = steps.reduce((sum, s) => sum + s.selectionKeys, 0);
  return { count: n, committed, top1, top1Rate: round(top1 / n), mrr: round(reciprocal / n), selectionKeysPerWord: round(keys / n) };
}

export function evaluate(directory) {
  const run = JSON.parse(readFileSync(path.join(directory, 'run.json'), 'utf8'));
  const byCondition = { baseline: [], ranking: [] };
  const times = { baseline: { letter: [], selection: [], bar: [] }, ranking: { letter: [], selection: [], bar: [] } };
  const failures = [];
  for (const entry of run.runs) {
    const result = JSON.parse(readFileSync(path.join(directory, entry.result), 'utf8'));
    if (result.status !== 'completed') { failures.push({ ...entry, failure: result.failure }); continue; }
    const condition = entry.contextRanking ? 'ranking' : 'baseline';
    byCondition[condition].push(...result.steps.map(step => ({ ...step, seed: entry.seed })));
    times[condition].letter.push(...result.letterKeyMs);
    times[condition].selection.push(...result.selectionKeyMs);
    times[condition].bar.push(...result.candidateBarMs);
  }
  const conditions = {};
  for (const [condition, steps] of Object.entries(byCondition)) {
    const targets = steps.filter(s => s.kind === 'target');
    conditions[condition] = {
      targets: score(targets),
      firstOccurrence: score(targets.filter(s => s.occurrence === 1)),
      laterOccurrences: score(targets.filter(s => s.occurrence > 1)),
      contexts: score(steps.filter(s => s.kind === 'context')),
      timingMs: { letterKey: timing(times[condition].letter), selectionKey: timing(times[condition].selection),
        candidateBar: timing(times[condition].bar) },
    };
  }
  // Per-pair later-occurrence outcome, to show which words gain or lose.
  const pairs = {};
  for (const [condition, steps] of Object.entries(byCondition)) {
    for (const s of steps.filter(s => s.kind === 'target' && s.occurrence > 1)) {
      pairs[s.pair] ??= { baseline: [0, 0], ranking: [0, 0] };
      pairs[s.pair][condition][0] += s.committed && s.rank === 0 ? 1 : 0;
      pairs[s.pair][condition][1] += 1;
    }
  }
  const changed = Object.entries(pairs).map(([pair, v]) => ({ pair, baselineTop1: v.baseline[0], rankingTop1: v.ranking[0], of: v.ranking[1] }))
    .filter(p => p.baselineTop1 !== p.rankingTop1);
  return { benchmark: run.benchmark, failures, conditions, changedPairs: changed,
    summary: Object.fromEntries(Object.entries(conditions).map(([c, v]) => [c, {
      laterTop1: v.laterOccurrences.top1Rate, laterMRR: v.laterOccurrences.mrr, laterKeys: v.laterOccurrences.selectionKeysPerWord,
      firstTop1: v.firstOccurrence.top1Rate, letterP95: v.timingMs.letterKey.p95, barP95: v.timingMs.candidateBar.p95 }])) };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  console.log(JSON.stringify(evaluate(path.resolve(process.argv[2])), null, 2));
}
