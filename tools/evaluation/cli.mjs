import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { readJson, loadCorpus, validateExperiment } from './src/manifest.mjs';
import { plan, runBenchmark } from './src/runner.mjs';
import { writeReport } from './src/report.mjs';
import { runScenarios } from './src/e2e.mjs';

const command = process.argv[2];
const options = {};
try {
  for (let i = 3; i < process.argv.length; i += 2) {
    const key = process.argv[i];
    if (!key.startsWith('--') || !process.argv[i + 1] || process.argv[i + 1].startsWith('--') || options[key.slice(2)]) throw new Error('Use unique --name value pairs');
    options[key.slice(2)] = process.argv[i + 1];
  }
  const allowed = { plan: ['config', 'out'], validate: ['corpus'], run: ['config', 'corpus', 'driver', 'out'], report: ['results'], e2e: ['suite', 'driver', 'out'] };
  if (!allowed[command] || Object.keys(options).some(key => !allowed[command].includes(key))) throw new Error('Commands: plan, validate, run, report, e2e; see tools/evaluation/README.md');
  const required = key => { if (!options[key]) throw new Error(`--${key} is required; no inference driver or corpus is supplied by the template`); return path.resolve(options[key]); };
  if (command === 'plan') {
    const config = validateExperiment(await readJson(required('config')));
    const output = required('out');
    await mkdir(path.dirname(output), { recursive: true, mode: 0o700 });
    await writeFile(output, JSON.stringify(plan(config), null, 2), { flag: 'wx', mode: 0o600 });
    console.log(`Plan written: ${config.variants.length} variants, ${config.seeds.length} seeds, ${config.cacheStates.length} cache strata; no inference performed.`);
  } else if (command === 'validate') {
    const corpus = await loadCorpus(required('corpus'));
    console.log(`Validated ${corpus.samples.length} pinned fixtures, split isolation and memory provenance.`);
  } else if (command === 'run') {
    const configFile = required('config');
    const corpusFile = required('corpus');
    const driverFile = required('driver');
    const output = required('out');
    const config = validateExperiment(await readJson(configFile));
    const corpus = await loadCorpus(corpusFile);
    await mkdir(path.dirname(output), { recursive: true, mode: 0o700 });
    const count = await runBenchmark(config, corpus, corpusFile, await readJson(driverFile), path.dirname(driverFile), output);
    console.log(`Completed ${count} live observations. Run report separately.`);
  } else if (command === 'report') {
    const report = await writeReport(required('results'));
    console.log(`Report written for ${report.observations} observations; no transcripts printed.`);
  } else {
    const driverFile = required('driver');
    const suite = await readJson(required('suite'));
    const output = required('out');
    await mkdir(path.dirname(output), { recursive: true, mode: 0o700 });
    const count = await runScenarios(suite, await readJson(driverFile), path.dirname(driverFile), output);
    console.log(`Passed ${count} actual OS scenarios.`);
  }
} catch (error) {
  console.error(`Evaluation failed: ${error.message}`);
  process.exitCode = 1;
}
