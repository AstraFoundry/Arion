import { randomUUID } from 'node:crypto';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { digest, hashFile, readJson } from './manifest.mjs';
import { requireValue as need } from './protocol.mjs';
import { checkedRewriteRows } from './rewrite-runner.mjs';
import { shuffle, pairedBootstrap } from './statistics.mjs';

export async function exportRewriteHuman(runDirectory, output, minRaters = 3, seed = 1729) {
  need(Number.isSafeInteger(minRaters) && minRaters >= 2 && Number.isSafeInteger(seed), 'At least two independent raters and explicit shuffle seed required');
  const { metadata, corpus, rows } = await checkedRewriteRows(runDirectory, { retainCandidates: true });
  const samples = new Map(corpus.samples.map(s => [s.id, s]));
  const tasks = [];
  const mapping = [];
  for (const row of shuffle(rows.filter(r => r.metrics.eligible && r.metrics.accepted), seed)) {
    const taskId = randomUUID();
    const sample = samples.get(row.caseId);
    tasks.push({ taskId, source: sample.sourceText, intendedMeaning: sample.expected.intent, userInstruction: sample.userInstruction, scenario: sample.scenario, context: sample.context.text, output: row.candidateText });
    mapping.push({ taskId, caseId: row.caseId, clusterId: row.clusterId, variantId: row.variantId, seed: row.seed, cacheState: row.cacheState, loadState: row.loadState });
  }
  need(tasks.length > 0, 'No delivered candidates available for human review');
  await mkdir(output, { recursive: false, mode: 0o700 });
  const packageData = { version: 1, kind: 'arion-rewrite-human-v1', minRaters, seed, instructions: 'Rate independently. Meaning: 1 contradicts/invents/omits essential intent, 5 preserves it. Style: 1 fails requested scenario, 5 fits it. Fluency: 1 unreadable, 5 natural. Mark unsupported factual claims and unnecessary refusal independently. Do not use model identity or an automated judge.', tasks };
  await writeFile(path.join(output, 'tasks.json'), JSON.stringify(packageData, null, 2), { mode: 0o600 });
  await writeFile(path.join(output, 'mapping-private.json'), JSON.stringify({ version: 1, kind: packageData.kind, runResultsSha256: metadata.resultsSha256, configHash: metadata.configHash, corpusHash: metadata.corpusHash, minRaters, tasksSha256: await hashFile(path.join(output, 'tasks.json')), mapping }, null, 2), { mode: 0o600 });
  return { tasks: tasks.length, eligibleObservations: rows.filter(r => r.metrics.eligible).length, withheldMapping: 'mapping-private.json' };
}

export async function reportRewriteHuman(runDirectory, reviewDirectory, ratingsFile) {
  const { metadata, rows: runRows } = await checkedRewriteRows(runDirectory);
  const map = await readJson(path.join(reviewDirectory, 'mapping-private.json'));
  const tasks = await readJson(path.join(reviewDirectory, 'tasks.json'));
  const votes = await readJson(ratingsFile);
  need(map.runResultsSha256 === metadata.resultsSha256 && map.configHash === metadata.configHash && map.corpusHash === metadata.corpusHash && map.tasksSha256 === await hashFile(path.join(reviewDirectory, 'tasks.json')), 'Human package/result identity mismatch');
  need(votes.version === 1 && votes.kind === 'arion-rewrite-human-v1' && Array.isArray(votes.ratings), 'Invalid human ratings schema');
  need(tasks.minRaters === map.minRaters && Number.isSafeInteger(map.minRaters) && map.minRaters >= 2, 'Human coverage policy mismatch');
  const expectedRows = runRows.filter(r => r.metrics.eligible && r.metrics.accepted);
  const rowKey = r => `${r.caseId}:${r.variantId}:${r.seed}:${r.cacheState}:${r.loadState}`;
  need(map.mapping.length === expectedRows.length && new Set(map.mapping.map(rowKey)).size === expectedRows.length && map.mapping.every(m => expectedRows.some(r => rowKey(r) === rowKey(m) && r.clusterId === m.clusterId)), 'Human mapping does not cover delivered candidates');
  const index = new Map(map.mapping.map(m => [m.taskId, m]));
  need(index.size === map.mapping.length && tasks.tasks.length === index.size && new Set(tasks.tasks.map(t => t.taskId)).size === index.size && tasks.tasks.every(t => index.has(t.taskId)), 'Human task inventory mismatch');
  const duplicate = new Set();
  const grouped = new Map();
  for (const v of votes.ratings) {
    need(index.has(v.taskId) && typeof v.raterId === 'string' && v.raterId.length > 0 && ['meaning', 'style', 'fluency'].every(k => Number.isInteger(v[k]) && v[k] >= 1 && v[k] <= 5) && typeof v.hallucination === 'boolean' && typeof v.unnecessaryRefusal === 'boolean', 'Invalid human rubric vote');
    const key = `${v.taskId}:${v.raterId}`; need(!duplicate.has(key), 'Duplicate human rater/task'); duplicate.add(key);
    if (!grouped.has(v.taskId)) grouped.set(v.taskId, []);
    grouped.get(v.taskId).push(v);
  }
  need([...index.keys()].every(id => (grouped.get(id)?.length ?? 0) >= map.minRaters), 'Incomplete independent human rating coverage');
  const rows = [...index].map(([id, m]) => {
    const raters = grouped.get(id);
    const scores = Object.fromEntries(['meaning', 'style', 'fluency', 'hallucination', 'unnecessaryRefusal'].map(k => [k, raters.reduce((n, v) => n + Number(v[k]), 0) / raters.length]));
    return { ...m, unit: 'rewrite', scores, raters: raters.length, disagreementRange: Object.fromEntries(['meaning', 'style', 'fluency'].map(k => [k, Math.max(...raters.map(v => v[k])) - Math.min(...raters.map(v => v[k]))])) };
  });
  const mean = (r, metric) => r.length ? r.reduce((n, v) => n + v.scores[metric], 0) / r.length : null;
  const summaries = {};
  const comparisons = [];
  for (const cacheState of metadata.config.cacheStates) for (const loadState of metadata.config.loadStates) {
    const stratum = rows.filter(r => r.cacheState === cacheState && r.loadState === loadState);
    for (const v of metadata.config.variants) {
      const r = stratum.filter(r => r.variantId === v.id);
      summaries[`${v.id}:${cacheState}:${loadState}`] = { scoredCandidates: r.length, deliveredCandidates: expectedRows.filter(row => row.variantId === v.id && row.cacheState === cacheState && row.loadState === loadState).length, eligibleObservations: runRows.filter(row => row.metrics.eligible && row.variantId === v.id && row.cacheState === cacheState && row.loadState === loadState).length, meanScores: Object.fromEntries(['meaning', 'style', 'fluency', 'hallucination', 'unnecessaryRefusal'].map(k => [k, mean(r, k)])), meanDisagreementRange: Object.fromEntries(['meaning', 'style', 'fluency'].map(k => [k, r.length ? r.reduce((n, row) => n + row.disagreementRange[k], 0) / r.length : null])) };
      if (v.id === metadata.config.baseline) continue;
      const a = stratum.filter(r => r.variantId === metadata.config.baseline);
      for (const metric of ['meaning', 'style', 'fluency', 'hallucination']) {
        const matched = a.length === r.length && a.every(row => r.some(other => other.caseId === row.caseId && other.seed === row.seed));
        comparisons.push({ baseline: metadata.config.baseline, treatment: v.id, cacheState, loadState, metric, ...(matched ? pairedBootstrap(a, r, items => mean(items, metric), metadata.config.bootstrap) : { status: 'accepted_output_coverage_mismatch', delta: null, ci95: null }) });
      }
    }
  }
  const report = { version: 1, kind: votes.kind, status: 'complete', tasks: rows.length, minRaters: map.minRaters, ratingsSha256: await hashFile(ratingsFile), runResultsSha256: metadata.resultsSha256, configHash: metadata.configHash, corpusHash: metadata.corpusHash, provenance: metadata.provenance, gitRevision: metadata.gitRevision, gitDirty: metadata.gitDirty, summaries, comparisons, caveat: 'Scores condition on delivered candidates. Coverage mismatches invalidate paired quality comparisons. Report acceptance and hard-safety checks alongside scores. Rater independence and source rights require review; mean ordinal ratings are descriptive.' };
  await writeFile(path.join(reviewDirectory, 'human-summary.json'), JSON.stringify(report, null, 2), { mode: 0o600 });
  return report;
}
