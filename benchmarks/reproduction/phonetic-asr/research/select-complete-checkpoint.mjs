import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { hashFile, loadCorpus } from '../../../tools/evaluation/src/manifest.mjs';
import { requireCoverage } from '../../../tools/evaluation/phonetic/design.mjs';
const base = path.resolve('.workspace/benchmarks/phonetic-asr-v2');
const source = path.join(base, 'correction');
const out = path.join(base, 'completed-17b-view');
const read = async file => JSON.parse(await fs.readFile(file, 'utf8'));
const lines = async file => (await fs.readFile(file, 'utf8')).trim().split('\n').filter(Boolean).map(JSON.parse);
const metadata = await read(path.join(source, 'run.json'));
if (metadata.status !== 'cancelled_by_user' || !metadata.terminationReason.includes('no further Qwen3-4B')) throw new Error('Explicit user-narrowed scope required');
for (const name of ['results', 'warmups']) if (metadata[`${name}HashAtStop`] !== await hashFile(path.join(source, `${name}.jsonl`))) throw new Error('Source observations changed after stop');
const corpus = await loadCorpus(metadata.corpusFile);
const samples = corpus.samples.filter(sample => sample.split === metadata.config.split);
const model = 'qwen3-1.7b';
const rows = (await lines(path.join(source, 'results.jsonl'))).filter(row => ['raw', model].includes(row.model));
requireCoverage(rows, samples, [model], metadata.config.variants);
const warmups = (await lines(path.join(source, 'warmups.jsonl'))).filter(row => row.model === model);
const expected = new Set(corpus.samples.filter(sample => sample.split === 'dev').slice(0, metadata.config.warmupCount).flatMap(sample => metadata.config.variants.map(variant => `${sample.id}/${variant}`)));
if (warmups.length !== expected.size || warmups.some(row => !expected.delete(`${row.caseId}/${row.variant}`)) || expected.size) throw new Error('Warmup coverage mismatch');
await fs.mkdir(out, { mode: 0o700 });
const config = { ...metadata.config, models: [model] };
await fs.writeFile(path.join(out, 'analysis-config.json'), JSON.stringify(config, null, 2) + '\n');
const writeRows = async (name, values) => fs.writeFile(path.join(out, `${name}.jsonl`), values.map(row => JSON.stringify(row)).join('\n') + '\n', { flag: 'wx', mode: 0o600 });
await writeRows('results', rows);
await writeRows('warmups', warmups);
const view = { ...metadata, status: 'complete', config, configHash: await hashFile(path.join(out, 'analysis-config.json')),
  runtimes: { [model]: metadata.runtimes[model] }, modelArtifacts: { [model]: metadata.modelArtifacts[model] },
  derivedAnalysisScope: { kind: 'complete-checkpoint-view', execution: 'No new inference; exact original observations selected after the user cancelled the remaining checkpoint.',
    sourceStatus: metadata.status, sourceRunHash: await hashFile(path.join(source, 'run.json')), sourceResultsHash: metadata.resultsHashAtStop, sourceWarmupsHash: metadata.warmupsHashAtStop,
    sourceCounts: metadata.retainedCounts, excluded: 'All Qwen3-4B observations and warmups; user explicitly cancelled that checkpoint.',
    controllerHash: await hashFile(fileURLToPath(import.meta.url)) },
  resultsHash: await hashFile(path.join(out, 'results.jsonl')), warmupsHash: await hashFile(path.join(out, 'warmups.jsonl')) };
delete view.failure;
delete view.resultsHashAtStop;
delete view.warmupsHashAtStop;
await fs.writeFile(path.join(out, 'run.json'), JSON.stringify(view, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ status: 'complete_checkpoint_view', model, rawControls: rows.filter(row => row.model === 'raw').length, measured: rows.filter(row => row.model === model).length, warmups: warmups.length, originalExecutionStatus: metadata.status }));
