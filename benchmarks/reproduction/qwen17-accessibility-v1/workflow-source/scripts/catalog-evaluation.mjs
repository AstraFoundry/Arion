// Records a completed model-comparison run in models/catalog.json.
// Usage: node scripts/catalog-evaluation.mjs <run-directory>
// The run must be complete with verified result checksums (bench-report);
// only content-free aggregates are copied, into benchmarks/results/<run>/.
import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync, writeFileSync, copyFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const runDir = path.resolve(process.argv[2] ?? '');
const read = name => JSON.parse(readFileSync(path.join(runDir, name), 'utf8'));
const run = read('run.json');
const summary = read('summary.json');
const sha = name => createHash('sha256').update(readFileSync(path.join(runDir, name))).digest('hex');
if (run.status !== 'complete' || summary.status !== 'complete') throw new Error('Run is not complete');
if (sha('results.jsonl') !== run.resultsSha256) throw new Error('Result checksum mismatch');
if (summary.configHash !== run.configHash) throw new Error('Summary belongs to another configuration');

const runId = path.basename(runDir);
const hardware = `${run.provenance.hardware.os}-${run.provenance.hardware.arch}`;
const reportDir = path.join(root, 'benchmarks', 'results', runId);
mkdirSync(reportDir, { recursive: true });
copyFileSync(path.join(runDir, 'summary.json'), path.join(reportDir, 'summary.json'));
const { config, provenance, gitRevision, gitDirty, corpusHash, configHash, observations } = run;
writeFileSync(path.join(reportDir, 'run.json'), JSON.stringify({ runId, status: run.status, config, configHash, corpusHash, gitRevision, gitDirty,
  observations, hardware: provenance.hardware, runtime: provenance.runtime, artifacts: provenance.artifacts, resultsSha256: run.resultsSha256 }, null, 2) + '\n');

const catalogPath = path.join(root, 'models', 'catalog.json');
const catalog = JSON.parse(readFileSync(catalogPath, 'utf8'));
const report = path.relative(root, path.join(reportDir, 'summary.json'));
const rate = (strata, language, unit) => strata[`language:${language}`]?.[`final_${unit}`]?.rate ?? null;
let recorded = 0;
for (const variant of config.variants) {
  const bundleId = variant.pipelineOverrides?.asr;
  const bundle = catalog.bundles.find(b => b.id === bundleId);
  const strata = summary.summaries[`${variant.id}:warm`];
  if (!bundle || !strata) continue;
  const errorRates = Object.fromEntries([['zh', 'cer'], ['en', 'wer'], ['mixed', 'mer']]
    .map(([language, unit]) => [language, rate(strata, language, unit)]).filter(([, value]) => value !== null));
  const overall = strata.overall;
  const evaluation = {
    runId, report, hardware, split: config.split, errorRates,
    stopToReviewP95Ms: overall.stopToReviewMs.p95, firstPartialP95Ms: overall.firstPartialMs.p95,
    realTimeFactor: overall.corpusRtf, observations: overall.count,
  };
  bundle.evaluations = bundle.evaluations.filter(e => !(e.hardware === hardware && e.split === config.split));
  bundle.evaluations.push(evaluation);
  recorded++;
}
writeFileSync(catalogPath, JSON.stringify(catalog, null, 2) + '\n');
console.log(`Recorded ${recorded} bundle evaluations from ${runId} (${hardware}).`);
