import { createReadStream } from 'node:fs';
import { createInterface } from 'node:readline';
import { writeFile } from 'node:fs/promises';
import path from 'node:path';
import { aggregate, NORMALIZATION } from './metrics.mjs';
import { pairedBootstrap } from './statistics.mjs';
import { digest, hashFile, readJson, validateExperiment } from './manifest.mjs';
import { requireValue } from './protocol.mjs';

function reportGroups(rows) {
  const groups = { overall: rows };
  for (const language of new Set(rows.map(row => row.language))) groups[`language:${language}`] = rows.filter(row => row.language === language);
  for (const tag of new Set(rows.flatMap(row => row.tags))) groups[`condition:${tag}`] = rows.filter(row => row.tags.includes(tag));
  return Object.fromEntries(Object.entries(groups).map(([key, observations]) => [key, aggregate(observations)]));
}

export function buildReport(metadata, rows) {
  const config = validateExperiment(metadata.config);
  requireValue(metadata.status === 'complete' && metadata.normalization === NORMALIZATION && metadata.configHash === digest(config), 'Incomplete run or configuration/normalization mismatch');
  const cases = metadata.plan.cases;
  requireValue(new Set(cases).size === cases.length && cases.length > 0, 'Invalid planned case inventory');
  requireValue(rows.length === metadata.plan.measuredRequests && rows.length === metadata.observations, 'Observation coverage mismatch');
  const keys = rows.map(row => `${row.variantId}:${row.caseId}:${row.seed}:${row.cacheState}`);
  requireValue(new Set(keys).size === keys.length, 'Duplicate observation');
  requireValue(rows.every(row => config.variants.some(v => v.id === row.variantId) && cases.includes(row.caseId) && config.seeds.includes(row.seed) && config.cacheStates.includes(row.cacheState)), 'Unexpected observation identity');
  const summaries = {};
  const comparisons = [];
  const rankerPairs = [];
  const fields = ['final_cer', 'final_wer', 'final_mer'];
  for (const cacheState of config.cacheStates) {
    const inCache = rows.filter(row => row.cacheState === cacheState);
    const baseline = inCache.filter(row => row.variantId === config.baseline);
    for (const variant of config.variants) {
      const treatment = inCache.filter(row => row.variantId === variant.id);
      requireValue(treatment.length === cases.length * config.seeds.length, 'Missing paired condition observations');
      summaries[`${variant.id}:${cacheState}`] = reportGroups(treatment);
      if (variant.id === config.baseline) continue;
      for (const field of fields) {
        // Restrict a language/unit stratum before resampling so null resamples do
        // not become artificially favorable confidence intervals.
        const unit = field.split('_')[1];
        const a = baseline.filter(row => row.unit === unit && !row.metrics.silence);
        const b = treatment.filter(row => row.unit === unit && !row.metrics.silence);
        comparisons.push({ baseline: config.baseline, treatment: variant.id, cacheState, metric: field, direction: 'lower', ...pairedBootstrap(a, b, group => aggregate(group)[field].rate, config.bootstrap) });
      }
      if (config.pipeline.streamingMetrics) {
        for (const [field, direction] of [['annotatedFirstPartialMs', 'lower'], ['previewFirstPartialMs', 'lower'], ['annotatedStopToReviewMs', 'lower'], ['tailExactMatchRate', 'higher']]) {
          comparisons.push({ baseline: config.baseline, treatment: variant.id, cacheState, metric: `streaming.${field}`, direction, ...pairedBootstrap(baseline, treatment, group => { const value = aggregate(group).streaming[field]; return typeof value === 'object' && value !== null ? value.p95 : value; }, config.bootstrap) });
        }
      }
      comparisons.push({ baseline: config.baseline, treatment: variant.id, cacheState, metric: 'stopToReviewMs.p95', direction: 'lower', ...pairedBootstrap(baseline, treatment, group => aggregate(group).stopToReviewMs.p95, config.bootstrap) });
    }
    // Isolate ranker treatment with memory/correction held fixed.
    for (const off of config.variants.filter(v => v.features.ranker === 'source')) {
      const on = config.variants.find(v => v.features.ranker === 'laya' && v.features.memory === off.features.memory && v.features.correction === off.features.correction);
      if (!on) continue;
      const a = inCache.filter(row => row.variantId === off.id);
      const b = inCache.filter(row => row.variantId === on.id);
      const byKey = new Map(b.map(row => [`${row.caseId}:${row.seed}`, row]));
      const matched = a.every(row => row.candidateHash === byKey.get(`${row.caseId}:${row.seed}`)?.candidateHash);
      rankerPairs.push({ baseline: off.id, treatment: on.id, cacheState, candidatesMatched: matched, metric: 'ranking.top1', direction: 'higher', ...(matched ? pairedBootstrap(a, b, group => aggregate(group).ranking.top1, config.bootstrap) : { status: 'candidate_set_mismatch', delta: null, ci95: null }) });
    }
  }
  return { version: 1, status: 'complete', normalization: NORMALIZATION, observations: rows.length, configHash: metadata.configHash, corpusHash: metadata.corpusHash, summaries, comparisons, rankerPairs, inference: config.design === 'validation' ? 'Single-backend validation; descriptive results without a paired treatment or significance claim.' : 'Exploratory paired speaker-cluster percentile CIs; no family-wise significance claim.' };
}

export async function writeReport(directory) {
  const metadata = await readJson(path.join(directory, 'run.json'));
  requireValue(await hashFile(path.join(directory, 'results.jsonl')) === metadata.resultsSha256 && await hashFile(path.join(directory, 'warmups.jsonl')) === metadata.warmupsSha256, 'Result/warmup checksum mismatch');
  const rows = [];
  const inputLines = createInterface({ input: createReadStream(path.join(directory, 'results.jsonl')), crlfDelay: Infinity });
  for await (const line of inputLines) {
    const { response, ...observation } = JSON.parse(line);
    rows.push(observation); // Raw transcripts need not remain resident for bootstrap.
  }
  const report = buildReport(metadata, rows);
  await writeFile(path.join(directory, 'summary.json'), JSON.stringify(report, null, 2), { mode: 0o600 });
  const lines = ['# Benchmark report', '', report.inference, '', '| Variant / cache | Count | Final CER | Final WER | Final MER | Oracle recall | Top-1 | Review p95 ms | RTF p95 |', '| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |'];
  const show = value => value === null ? 'n/a' : value.toFixed(4);
  for (const [name, strata] of Object.entries(report.summaries)) {
    const s = strata.overall;
    lines.push(`| ${name} | ${s.count} | ${show(s.final_cer.rate)} | ${show(s.final_wer.rate)} | ${show(s.final_mer.rate)} | ${show(s.ranking.oracleRecall)} | ${show(s.ranking.top1)} | ${show(s.stopToReviewMs.p95)} | ${show(s.rtf.p95)} |`);
  }
  lines.push('', 'Per-language/condition results, comparisons where present, protected-span proxies and ranker candidate-set checks are in `summary.json`. Hardware/artifact provenance and exact settings are in `run.json`. Missing first partials and silence outputs remain in their denominator counts.', '');
  await writeFile(path.join(directory, 'summary.md'), lines.join('\n'), { mode: 0o600 });
  return report;
}
