import fs from 'node:fs/promises';
import { loadCorpus } from '../../../tools/evaluation/src/manifest.mjs';
import { scoreRow, summarizeRows, requireCoverage } from '../../../tools/evaluation/phonetic/design.mjs';
import { outputContractFailure } from '../../../tools/evaluation/phonetic/preflight.mjs';
const base = '.workspace/benchmarks/phonetic-asr-v2';
const read = async file => JSON.parse(await fs.readFile(file, 'utf8'));
const lines = async file => (await fs.readFile(file, 'utf8')).trim().split('\n').filter(Boolean).map(JSON.parse);
const meta = await read(`${base}/correction/run.json`);
const model = process.argv[2] ?? 'qwen3-1.7b';
if (!meta.config.models.includes(model)) throw new Error('Unknown model');
const corpus = await loadCorpus(meta.corpusFile);
const samples = corpus.samples.filter(sample => sample.split === meta.config.split);
const sampleIndex = new Map(samples.map(sample => [sample.id, sample]));
const evidence = new Map((await lines(`${meta.evidenceDirectory}/results.jsonl`)).map(row => [row.caseId, row.response]));
const rows = (await lines(`${base}/correction/results.jsonl`)).filter(row => ['raw', model].includes(row.model));
requireCoverage(rows, samples, [model], meta.config.variants);
const groups = {};
for (const row of rows) {
 const sample = sampleIndex.get(row.caseId);
 Object.assign(row, { unit: sample.unit, metrics: scoreRow(sample, evidence.get(row.caseId), row.response) });
 (groups[`${row.model}/${row.variant}`] ??= []).push(row);
}
const summary = { status: 'complete_model_phase_only', fullMatrixStatus: meta.status, model,
 note: 'Interim complete checkpoint aggregate; not a completed two-model report.',
 groups: Object.fromEntries(Object.entries(groups).map(([key, values]) => [key, { ...summarizeRows(values), outputContractFailures: values.filter(row => row.model !== 'raw' && outputContractFailure(row.response)).length }])) };
await fs.writeFile(`${base}/${model}-completed-phase.json`, JSON.stringify(summary, null, 2), { flag: 'wx' });
for (const [key, value] of Object.entries(summary.groups)) console.log(JSON.stringify({ condition: key,
 rates: Object.fromEntries(Object.entries(value.rates).map(([unit, result]) => [unit, result.rate])), improved: value.improved, worsened: value.worsened, corrupted: value.corrupted, originallyCorrect: value.originallyCorrect, truncated: value.truncated, contractFailures: value.outputContractFailures }));
