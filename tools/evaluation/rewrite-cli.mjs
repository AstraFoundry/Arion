import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { readJson } from './src/manifest.mjs';
import { loadRewriteCorpus, validateRewriteConfig } from './src/rewrite-manifest.mjs';
import { rewritePlan, runRewrite, reportRewrite } from './src/rewrite-runner.mjs';
import { exportRewriteHuman, reportRewriteHuman } from './src/rewrite-human.mjs';

try {
  const command = process.argv[2];
  const options = {};
  for (let i = 3; i < process.argv.length; i += 2) {
    const key = process.argv[i];
    if (!key.startsWith('--') || !process.argv[i + 1] || process.argv[i + 1].startsWith('--') || Object.hasOwn(options, key.slice(2))) throw new Error('Use unique --name value pairs');
    options[key.slice(2)] = process.argv[i + 1];
  }
  const allowed = { plan: ['config', 'out'], validate: ['corpus'], run: ['config', 'corpus', 'driver', 'out'], report: ['results'], 'human-export': ['results', 'out', 'raters', 'seed'], 'human-report': ['results', 'review', 'ratings'] };
  if (!allowed[command] || Object.keys(options).some(k => !allowed[command].includes(k))) throw new Error('Rewrite commands: plan, validate, run, report, human-export, human-report');
  const required = key => { if (!options[key]) throw new Error(`--${key} is required; no rewrite model/driver/corpus is bundled`); return path.resolve(options[key]); };
  if (command === 'plan') {
    const config = validateRewriteConfig(await readJson(required('config')));
    const output = required('out');
    await mkdir(path.dirname(output), { recursive: true, mode: 0o700 });
    await writeFile(output, JSON.stringify(rewritePlan(config), null, 2), { flag: 'wx', mode: 0o600 });
    console.log(`Rewrite plan: ${config.variants.length} variants; no inference performed.`);
  } else if (command === 'validate') {
    const corpus = await loadRewriteCorpus(required('corpus'));
    console.log(`Validated ${corpus.samples.length} rewrite fixtures; no inference performed.`);
  } else if (command === 'run') {
    const config = validateRewriteConfig(await readJson(required('config')));
    const corpusFile = required('corpus'); const driverFile = required('driver'); const output = required('out');
    const corpus = await loadRewriteCorpus(corpusFile);
    await mkdir(path.dirname(output), { recursive: true, mode: 0o700 });
    const count = await runRewrite(config, corpus, corpusFile, await readJson(driverFile), path.dirname(driverFile), output);
    console.log(`Completed ${count} live rewrite observations; human review remains separate.`);
  } else if (command === 'report') {
    const report = await reportRewrite(required('results'));
    console.log(`Checked ${report.observations} rewrite observations; semantic quality awaits blinded human review.`);
  } else if (command === 'human-export') {
    const output = required('out'); await mkdir(path.dirname(output), { recursive: true, mode: 0o700 });
    const result = await exportRewriteHuman(required('results'), output, options.raters === undefined ? 3 : Number(options.raters), options.seed === undefined ? 1729 : Number(options.seed));
    console.log(`Exported ${result.tasks} blind tasks; mapping-private.json must stay hidden from raters.`);
  } else {
    const report = await reportRewriteHuman(required('results'), required('review'), required('ratings'));
    console.log(`Checked ${report.tasks} human tasks and independent-rater coverage; no raw text printed.`);
  }
} catch (error) {
  console.error(`Rewrite evaluation failed: ${error.message}`); process.exitCode = 1;
}
