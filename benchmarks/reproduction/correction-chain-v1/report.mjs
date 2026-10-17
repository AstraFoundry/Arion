// Replay verified model observations; labels remain exclusively evaluator-side.
import fs from 'node:fs';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { errorCounts, distribution } from '../../../tools/evaluation/src/metrics.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
export const hash = file => createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const read = file => JSON.parse(fs.readFileSync(file, 'utf8'));
const lines = file => fs.readFileSync(file, 'utf8').trim().split('\n').filter(Boolean).map(JSON.parse);
const check = (ok, message) => { if (!ok) throw new Error(message); };

export function indexUnique(rows, key) {
  const map = new Map();
  for (const row of rows) {
    check(!map.has(row[key]), `Duplicate ${key}: ${row[key]}`);
    map.set(row[key], row);
  }
  return map;
}

function contains(text, value, language) {
  const normalized = text.normalize('NFKC').toLowerCase();
  const needle = value.normalize('NFKC').toLowerCase();
  if (language !== 'en' && !/^\d+$/.test(needle)) return normalized.includes(needle);
  const escaped = needle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const boundary = /^\d+$/.test(needle) ? '[0-9]' : '[a-z]';
  return new RegExp(`(?<!${boundary})${escaped}(?!${boundary})`, 'u').test(normalized);
}

export function constraints(sample, labels, text, failed = false) {
  if (text === null) return { pass: false, missing: labels.required, forbidden: [], failed: true };
  const matches = value => [value, ...(labels.aliases?.[value] ?? [])].some(alias => contains(text, alias, sample.language));
  const missing = labels.required.filter(value => !matches(value));
  const forbidden = labels.forbidden.filter(matches);
  const silenceViolation = sample.task === 'silence' && text.trim() !== '';
  return { pass: !failed && !silenceViolation && !missing.length && !forbidden.length, missing, forbidden, failed, silenceViolation };
}

function unit(language) { return { zh: 'cer', en: 'wer', mixed: 'mer' }[language]; }
function rate(rows, field) {
  const valid = rows.map(r => r[field]).filter(Boolean);
  const errors = valid.reduce((n, r) => n + r.errors, 0);
  const denominator = valid.reduce((n, r) => n + r.referenceUnits, 0);
  return { cases: rows.length, scored: valid.length, errors, referenceUnits: denominator, rate: denominator ? errors / denominator : null };
}
function summarize(rows) {
  return { count: rows.length, constraintPass: rows.filter(r => r.intent.pass).length,
    failures: rows.filter(r => r.intent.failed).length, rawProposalPass: rows.filter(r => r.rawProposalIntent?.pass).length, verdicts: Object.fromEntries([...new Set(rows.map(r => r.verdict).filter(Boolean))].map(v => [v, rows.filter(r => r.verdict === v).length])), intentOverlap: new Set(rows.map(r => r.language)).size > 1 ? null : rate(rows, 'intentOverlap'),
    silenceEmissions: rows.filter(r => r.intent.silenceViolation).length,
    serialEngineEstimateMs: distribution(rows.map(r => r.serialEngineEstimateMs).filter(Number.isFinite)) };
}

export function buildReport(directory) {
  const run = read(path.join(directory, 'run.json'));
  check(run.status === 'completed', 'Incomplete runs cannot produce a completed report');
  for (const [file, digest] of Object.entries(run.outputHashes)) check(hash(path.join(directory, file)) === digest, `Changed output: ${file}`);
  const fixtures = path.join(root, 'benchmarks/corpora/correction-chain-v1');
  check(hash(path.join(fixtures, 'inputs.json')) === run.inputsSha256, 'Input hash mismatch');
  check(hash(path.join(fixtures, 'labels.json')) === run.labelsSha256, 'Label hash mismatch');
  const samples = read(path.join(fixtures, 'inputs.json')).samples.filter(s => run.caseIds.includes(s.id));
  check(samples.length === run.caseIds.length, 'Case coverage mismatch');
  const labels = read(path.join(fixtures, 'labels.json'));
  const acoustic = indexUnique(lines(path.join(directory, 'acoustic.jsonl')), 'caseId');
  const correction = indexUnique(lines(path.join(directory, 'correction.jsonl')), 'caseId');
  const rewrites = indexUnique(lines(path.join(directory, 'rewrite.jsonl')), 'caseId');
  const inputs = indexUnique(read(path.join(directory, 'rewrite-inputs.json')).samples, 'id');
  check(acoustic.size === samples.length && correction.size === samples.length, 'Stage coverage mismatch');
  check(rewrites.size === inputs.size, 'Rewrite coverage mismatch');
  for (const [id, input] of inputs) check(rewrites.get(id)?.source === input.sourceText, 'Rewrite input/source mismatch');
  const rows = [];
  for (const sample of samples) {
    const a = acoustic.get(sample.id), c = correction.get(sample.id), label = labels[sample.id];
    check(a && c && a.audioSha256 === sample.audio.sha256, 'Acoustic identity mismatch');
    check(c.inputText === a.streaming.text && c.model === run.config.correctionModels[sample.language], 'Correction treatment mismatch');
    for (const arm of run.config.arms) {
      const rewrite = arm.endsWith('_llm') ? rewrites.get(`${sample.id}--${arm}`) : null;
      const base = arm.startsWith('offline') ? a.offline : a.streaming;
      const needsCorrection = arm.includes('correct');
      const failed = needsCorrection && c.truncated || Boolean(rewrite?.truncated);
      const intermediate = needsCorrection ? c.text : base.text;
      if (rewrite) check(rewrite.source === intermediate, 'Broken actual stage lineage');
      if (arm.endsWith('_llm')) check(rewrite || needsCorrection && c.truncated, 'Missing rewrite observation');
      const text = arm.endsWith('_llm') ? rewrite?.text ?? null : intermediate;
      const correctionMs = needsCorrection ? c.parentMs : 0;
      const serialEngineEstimateMs = text === null ? null : base.computeMs + correctionMs + (rewrite?.generationMs ?? 0);
      rows.push({ caseId: sample.id, language: sample.language, task: sample.task, arm, text,
        sourceText: base.text, correctedText: needsCorrection ? c.text : null, proposal: rewrite?.proposal ?? null,
        verdict: rewrite?.verdict ?? null, intent: constraints(sample, label, text, failed),
        rawProposalIntent: rewrite ? constraints(sample, label, rewrite.proposal, rewrite.truncated) : null,
        intentOverlap: text === null ? null : errorCounts(label.references[0], text, unit(sample.language)),
        acousticError: text === null ? null : errorCounts(label.transcript, text, unit(sample.language)),
        serialEngineEstimateMs, correctionMs, rewriteMs: rewrite?.generationMs ?? 0 });
    }
  }
  const arms = {};
  for (const arm of run.config.arms) {
    const selected = rows.filter(r => r.arm === arm);
    arms[arm] = { all: summarize(selected), byLanguage: Object.fromEntries(['zh', 'en', 'mixed'].map(lang => [lang, summarize(selected.filter(r => r.language === lang))])),
      byTask: Object.fromEntries([...new Set(samples.map(s => s.task))].map(task => [task, summarize(selected.filter(r => r.task === task))])) };
    if (['asr', 'asr_correct', 'offline'].includes(arm)) arms[arm].acousticByLanguage = Object.fromEntries(['zh', 'en', 'mixed'].map(lang => [lang, rate(selected.filter(r => r.language === lang), 'acousticError')]));
  }
  const paired = {};
  for (const [base, treatment] of [['asr', 'asr_correct'], ['asr', 'asr_llm'], ['asr_llm', 'asr_correct_llm'], ['asr_llm', 'offline_llm']]) {
    const pairs = samples.map(s => [rows.find(r => r.caseId === s.id && r.arm === base), rows.find(r => r.caseId === s.id && r.arm === treatment)]);
    paired[`${base} -> ${treatment}`] = { cases: pairs.length, intentImproved: pairs.filter(([a, b]) => !a.intent.pass && b.intent.pass).length,
      intentWorsened: pairs.filter(([a, b]) => a.intent.pass && !b.intent.pass).length,
      acousticImproved: base !== 'asr' || treatment !== 'asr_correct' ? null : pairs.filter(([a, b]) => a.acousticError && b.acousticError && b.acousticError.errors < a.acousticError.errors).length,
      acousticWorsened: base !== 'asr' || treatment !== 'asr_correct' ? null : pairs.filter(([a, b]) => a.acousticError && b.acousticError && b.acousticError.errors > a.acousticError.errors).length,
      correctSourceCases: base !== 'asr' || treatment !== 'asr_correct' ? null : pairs.filter(([a]) => a.acousticError?.errors === 0).length,
      correctSourceCorrupted: base !== 'asr' || treatment !== 'asr_correct' ? null : pairs.filter(([a, b]) => a.acousticError?.errors === 0 && (b.acousticError === null || b.acousticError.errors > 0)).length };
  }
  return { kind: run.kind, split: run.split, status: 'completed', runSha256: hash(path.join(directory, 'run.json')),
    inputSha256: run.inputsSha256, labelsSha256: run.labelsSha256, scorerSha256: hash(fileURLToPath(import.meta.url)), metricsSha256: hash(path.join(root, 'tools/evaluation/src/metrics.mjs')), arms, paired,
    correctionTimingMs: distribution([...correction.values()].map(r => r.generationMs)),
    processPeakRssBytes: { acoustic: Math.max(...[...acoustic.values()].map(r => r.peakRssBytes)), correction: Math.max(...[...correction.values()].map(r => r.peakRssBytes)), rewrite: Math.max(...[...rewrites.values()].map(r => r.peakRssBytes || 0)) || null },
    modelLoadMs: { acoustic: run.acousticLoad.loadMs, correction: run.correctionLoad.loadMs, rewrite: null },
    semanticReview: 'pending; string constraints and intent reference overlap are proxies, not semantic correctness',
    limitations: run.config.limits, rows };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [directory, output] = process.argv.slice(2);
  if (!directory || !output) throw new Error('Usage: report.mjs RUN NEW_OUTPUT_JSON');
  const report = buildReport(path.resolve(directory));
  fs.writeFileSync(output, JSON.stringify(report, null, 2) + '\n', { flag: 'wx' });
  console.log(JSON.stringify({ split: report.split, arms: Object.fromEntries(Object.entries(report.arms).map(([k, v]) => [k, v.all.constraintPass])), paired: report.paired }, null, 2));
}
