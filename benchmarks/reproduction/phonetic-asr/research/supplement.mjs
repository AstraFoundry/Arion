import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { hashFile, loadCorpus } from '../../../tools/evaluation/src/manifest.mjs';
import { distribution } from '../../../tools/evaluation/src/metrics.mjs';
import { pairedBootstrap } from '../../../tools/evaluation/src/statistics.mjs';
import { scoreRow, summarizeRows, requireCoverage } from '../../../tools/evaluation/phonetic/design.mjs';
import { outputContractFailure } from '../../../tools/evaluation/phonetic/preflight.mjs';

const base = path.resolve('.workspace/benchmarks/phonetic-asr-v2');
const read = async file => JSON.parse(await fs.readFile(file, 'utf8'));
const lines = async file => (await fs.readFile(file, 'utf8')).trim().split('\n').filter(Boolean).map(JSON.parse);
const correction = await read(path.join(base, 'completed-17b-view/run.json'));
const primary = await read(path.join(base, 'summary.json'));
const offline = await read(path.join(base, 'offline-control/run.json'));
if (correction.status !== 'complete' || primary.status !== 'complete' || offline.status !== 'complete') throw new Error('Completed real inference and primary report required');
for (const [directory, metadata] of [['completed-17b-view', correction], ['offline-control', offline]]) {
  for (const name of ['results', 'warmups']) {
    if (metadata[`${name}Hash`] !== await hashFile(path.join(base, directory, `${name}.jsonl`))) throw new Error('Changed observations');
  }
  if (metadata.corpusHash !== primary.corpusHash) throw new Error('Corpus mismatch');
}
if (primary.runHash !== await hashFile(path.join(base, 'completed-17b-view/run.json'))) throw new Error('Changed primary run');
const corpus = await loadCorpus(correction.corpusFile);
const samples = corpus.samples.filter(sample => sample.split === correction.config.split);
const sampleIndex = new Map(samples.map(sample => [sample.id, sample]));
const evidenceMeta = await read(path.join(correction.evidenceDirectory, 'run.json'));
if (evidenceMeta.resultsHash !== await hashFile(path.join(correction.evidenceDirectory, 'results.jsonl'))) throw new Error('Changed acoustic evidence');
const evidence = new Map((await lines(path.join(correction.evidenceDirectory, 'results.jsonl'))).map(row => [row.caseId, row.response]));
const rows = await lines(path.join(base, 'completed-17b-view/results.jsonl'));
requireCoverage(rows, samples, correction.config.models, correction.config.variants);
const measured = new Map(rows.map(row => [`${row.caseId}/${row.model}/${row.variant}`, row]));
const makeRow = (sample, response) => ({ caseId: sample.id, unit: sample.unit, language: sample.language,
  clusterId: sample.speakerId, seed: 11, cacheState: 'warm', response,
  metrics: scoreRow(sample, evidence.get(sample.id), response) });
const raw = samples.map(sample => makeRow(sample, measured.get(`${sample.id}/raw/raw`).response));
const summarize = values => ({ ...summarizeRows(values), outputContractFailures: values.filter(row => outputContractFailure(row.response)).length });
const stratified = values => ({ overall: summarize(values), byLanguage: Object.fromEntries(['zh', 'en', 'mixed'].map(language => [language, summarize(values.filter(row => row.language === language))])) });
const compare = (baseline, treatment) => Object.fromEntries(['cer', 'wer', 'mer'].map(unit => {
  const select = values => values.filter(row => row.unit === unit && !row.metrics.silence);
  return [unit, pairedBootstrap(select(baseline), select(treatment), values => {
    const units = values.reduce((sum, row) => sum + row.metrics.corrected.referenceUnits, 0);
    return units ? values.reduce((sum, row) => sum + row.metrics.corrected.errors, 0) / units : null;
  }, { iterations: correction.config.bootstrapIterations, seed: correction.config.orderSeed })];
}));

// Declared before completed test efficacy was inspected. No reference enters routing.
const routing = new Map(samples.map(sample => {
  const scores = evidence.get(sample.id).segments.flatMap(segment => segment.logScores);
  if (scores.some(score => !Number.isFinite(score))) throw new Error('Invalid score');
  return [sample.id, { invoke: scores.length === 0 || scores.some(score => score < correction.threshold), missing: scores.length === 0 }];
}));
const routed = {};
for (const model of correction.config.models) for (const variant of correction.config.variants) {
  const selected = samples.map(sample => makeRow(sample, routing.get(sample.id).invoke ? measured.get(`${sample.id}/${model}/${variant}`).response : measured.get(`${sample.id}/raw/raw`).response));
  const invoked = selected.filter(row => routing.get(row.caseId).invoke);
  routed[`${model}/${variant}`] = { ...stratified(selected), versusRaw: compare(raw, selected),
    invoked: invoked.length, hypotheticalGenerationMs: distribution(invoked.map(row => row.response.generationMs)) };
}
const offlineRows = await lines(path.join(base, 'offline-control/results.jsonl'));
if (offlineRows.length !== samples.length || new Set(offlineRows.map(row => row.caseId)).size !== samples.length || offlineRows.some(row => !sampleIndex.has(row.caseId))) throw new Error('Offline pairing mismatch');
const scoredOffline = offlineRows.map(row => makeRow(sampleIndex.get(row.caseId), { ...row.response, generationMs: 0, truncated: false }));
const failed = await read('.workspace/benchmarks/phonetic-asr-v1/correction/run.json');
const runtimeVersions = Object.fromEntries(Object.entries(correction.runtimes).map(([model, runtime]) => [model, Object.fromEntries(Object.entries(runtime).filter(([key]) => !['id', 'ok'].includes(key)))]));
const responseDiagnostics = Object.fromEntries(correction.config.models.flatMap(model => correction.config.variants.map(variant => {
  const values = rows.filter(row => row.model === model && row.variant === variant);
  const fields = ['low_log_score_threshold', 'token_log_scores', 'text_derived_pinyin', 'audio_derived_ipa', '"asr_text"'];
  return [`${model}/${variant}`, { count: values.length,
    evidenceFieldLiteralEmissions: values.filter(row => fields.some(field => row.response.text.includes(field))).length,
    note: 'Literal evidence-field marker diagnostic, not a complete semantic format validator. All outputs remain scored as emitted.' }];
})));
const summary = {
  responseDiagnostics,
  status: 'complete', kind: 'arion-phonetic-supplement-v1', corpusHash: primary.corpusHash,
  controllerHash: await hashFile(fileURLToPath(import.meta.url)), primaryReportHash: await hashFile(path.join(base, 'summary.json')),
  routing: { policy: 'Invoke correction when any emitted-token log score is below the fixed development quartile; invoke for missing scores; otherwise keep raw ASR. No reference or phone score enters selection.',
    scope: 'Exploratory utterance-level counterfactual using already observed proposals, not uncertain-span phonetic decoding or measured routed runtime.',
    threshold: correction.threshold, samples: samples.length, invoked: [...routing.values()].filter(row => row.invoke).length,
    missingScores: [...routing.values()].filter(row => row.missing).length, treatments: routed },
  offlineControl: { ...stratified(scoredOffline), versusRaw: compare(raw, scoredOffline),
    computeMs: distribution(offlineRows.map(row => row.response.computeMs)), model: offline.model, runtime: offline.runtime,
    runHash: await hashFile(path.join(base, 'offline-control/run.json')), resultsHash: offline.resultsHash, warmupsHash: offline.warmupsHash,
    controllerHash: offline.controllerHash, artifactCount: offline.artifacts.length },
  provenance: { completedCheckpointScope: correction.derivedAnalysisScope, correctionRunHash: primary.runHash, correctionResultsHash: primary.resultsHash,
    acousticRunHash: correction.evidenceRunHash, acousticResultsHash: evidenceMeta.resultsHash,
    preflightRunHash: correction.preflightRunHash, preflightResultsHash: correction.preflightResultsHash,
    preflightCheck: correction.preflightCheck, codeArtifacts: correction.codeArtifacts.map(item => ({ file: path.basename(item.path), sha256: item.sha256 })),
    runtimeVersions, runtimeArtifacts: correction.runtimeArtifacts.length,
    modelArtifacts: Object.fromEntries(Object.entries(correction.modelArtifacts).map(([model, artifacts]) => [model, artifacts.map(item => ({ file: path.basename(item.path), sha256: item.sha256 }))])),
    failedAttempt: { status: failed.status, failure: failed.failure, retainedCounts: failed.retainedCounts, resultsHashAtAbort: failed.resultsHashAtAbort, warmupsHashAtAbort: failed.warmupsHashAtAbort,
      interpretation: 'Interrupted output-contract failure; not evidence of phonetic representation efficacy. Revised prompt was checked on dev before reusing the test corpus.' } },
  limitations: ['Test corpus reused after format diagnosis; exploratory engineering evidence, not untouched confirmatory evaluation.', 'Desktop background workloads were present; timing does not establish an isolated controlled latency ranking.', 'All failures and truncations remain scored; no source substitution repairs failed model output.', 'IPA-only is an ablation of this conservative editor with text-copy demonstrations, not a separately trained phoneme-to-grapheme model.']
};
await fs.writeFile(path.join(base, 'supplemental.json'), JSON.stringify(summary, null, 2) + '\n', { flag: 'wx' });
console.log('Supplemental aggregate complete');
