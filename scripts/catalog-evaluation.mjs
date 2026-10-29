import assert from 'node:assert/strict';
import { readFile, mkdir, writeFile, rename } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const digest = /^[a-f0-9]{64}$/;
const revision = /^[a-f0-9]{40}$/;

export function reviewedEvaluation(catalog, input) {
  assert.equal(input.schema, 1);
  assert.equal(input.status, 'complete');
  assert.match(input.runId, /^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/);
  assert.match(input.source?.productCommit, revision);
  assert.match(input.source?.evaluatorCommit, revision);
  assert.equal(input.source.dirty, false, 'Dirty runs cannot become new published catalog evaluations.');
  for (const key of ['corpusHash', 'configHash', 'resultsSha256', 'summarySha256']) assert.match(input.source[key], digest);
  assert.ok(Array.isArray(input.evaluations) && input.evaluations.length > 0);
  const report = `models/evaluations/${input.runId}.json`;
  const bundles = structuredClone(catalog.bundles);
  const seen = new Set();
  const evaluations = input.evaluations.map(value => {
    assert.ok(!seen.has(value.bundleId), 'Repeated model bundle.');
    seen.add(value.bundleId);
    const bundle = bundles.find(item => item.id === value.bundleId);
    assert.ok(bundle, 'Unknown model bundle.');
    assert.match(value.hardware, /^[A-Za-z0-9._-]+$/);
    assert.ok(['dev', 'test'].includes(value.split));
    assert.ok(Number.isSafeInteger(value.observations) && value.observations > 0);
    for (const key of ['stopToReviewP95Ms', 'firstPartialP95Ms', 'realTimeFactor']) assert.ok(Number.isFinite(value[key]) && value[key] >= 0);
    assert.ok(value.errorRates && typeof value.errorRates === 'object');
    const errorRates = {};
    for (const [language, rate] of Object.entries(value.errorRates)) {
      assert.ok(['zh', 'en', 'mixed'].includes(language) && Number.isFinite(rate) && rate >= 0);
      errorRates[language] = rate;
    }
    const evaluation = { runId: input.runId, report, hardware: value.hardware, split: value.split, errorRates,
      stopToReviewP95Ms: value.stopToReviewP95Ms, firstPartialP95Ms: value.firstPartialP95Ms,
      realTimeFactor: value.realTimeFactor, observations: value.observations };
    bundle.evaluations = bundle.evaluations.filter(item => !(item.hardware === evaluation.hardware && item.split === evaluation.split));
    bundle.evaluations.push(evaluation);
    return { bundleId: value.bundleId, ...evaluation };
  });
  const source = Object.fromEntries(['productCommit', 'evaluatorCommit', 'dirty', 'corpusHash', 'configHash', 'resultsSha256', 'summarySha256'].map(key => [key, input.source[key]]));
  return { catalog: { ...catalog, bundles }, report, summary: { schema: 1, status: 'complete', runId: input.runId, source, evaluations } };
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  assert.ok(process.argv.length === 4 && process.argv[3] === '--reviewed', 'Use node scripts/catalog-evaluation.mjs <reviewed-summary.json> --reviewed.');
  const catalogPath = path.join(root, 'models/catalog.json');
  const result = reviewedEvaluation(JSON.parse(await readFile(catalogPath, 'utf8')), JSON.parse(await readFile(path.resolve(process.argv[2]), 'utf8')));
  const reportPath = path.join(root, result.report);
  const reportBytes = `${JSON.stringify(result.summary, null, 2)}\n`;
  await mkdir(path.dirname(reportPath), { recursive: true });
  try {
    await writeFile(reportPath, reportBytes, { flag: 'wx' });
  } catch (error) {
    if (error.code !== 'EEXIST') throw error;
    assert.equal(await readFile(reportPath, 'utf8'), reportBytes, 'The published run ID already has different content.');
  }
  const temporary = `${catalogPath}.tmp`;
  await writeFile(temporary, `${JSON.stringify(result.catalog, null, 2)}\n`, { flag: 'wx' });
  await rename(temporary, catalogPath);
  console.log(`Imported ${result.summary.evaluations.length} reviewed aggregate model evaluations.`);
}
