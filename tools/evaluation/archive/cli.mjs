import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';
import { parseArgs } from 'node:util';
import assert from 'node:assert/strict';
import { openArchive, sha256 } from './manifest.mjs';
import { verifyEvidence } from './verify.mjs';
import { buildReport } from '../src/report.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const { positionals: [command], values: flags } = parseArgs({ allowPositionals: true, options: { out: { type: 'string' } } });
const archive = await openArchive(path.join(root, 'benchmarks/migration/archive.json'), root);
assert.equal(sha256(await fs.readFile(path.join(root, 'benchmarks/migration/import-plan.json'))), archive.manifest.planSha256,
  'Changed archive import plan');
const verification = await verifyEvidence(archive);
if (command === 'verify') {
  console.log(JSON.stringify(verification, null, 2));
} else if (command === 'report') {
  if (!flags.out) throw new Error('--out must select a new report directory');
  const out = path.resolve(flags.out);
  const benchmarkRoot = path.join(root, 'benchmarks');
  if (out === root || out === benchmarkRoot || out.startsWith(benchmarkRoot + path.sep)) throw new Error('Write regenerated reports outside the immutable benchmark archive');
  await fs.mkdir(path.dirname(out), { recursive: true });
  await fs.mkdir(out);
  const read = async file => JSON.parse(await fs.readFile(file));
  const checks = [];
  // Existing reports gained additional scorer fields over time. Compare every
  // historical field, excluding only the report-generation wall clock.
  function comparePublished(expected, actual, location = '') {
    if (Array.isArray(expected)) {
      assert.equal(actual?.length, expected.length, location);
      expected.forEach((value, i) => comparePublished(value, actual[i], `${location}[${i}]`));
    } else if (expected !== null && typeof expected === 'object') {
      for (const [key, value] of Object.entries(expected)) {
        if (key === 'generated') continue;
        assert.ok(Object.hasOwn(actual, key), `Missing published field ${location}.${key}`);
        comparePublished(value, actual[key], `${location}.${key}`);
      }
    } else assert.deepEqual(actual, expected, location);
  }
  const runNode = args => execFileSync(process.execPath, args, { cwd: root, stdio: 'pipe' });
  const live = [
    ['disfluency-alignment-report-v2', 'disfluency-alignment-v2'],
    ['disfluency-alignment-report', 'disfluency-alignment-v1'],
    ['live-rewrite-report', 'live-rewrite-v1'],
    ['context-ablation-report', 'context-ablation-v1'],
    ['scene-switching-report', 'scene-switching-v1'],
  ];
  for (const [config, published] of live) {
    const output = path.join(out, `${published}.json`);
    runNode(['tools/evaluation/live/report.mjs', `benchmarks/configs/${config}.json`, output]);
    comparePublished(await archive.json(`benchmarks/results/${published}/summary.json`), await read(output));
    checks.push({ study: published, publishedFields: 'equal', output: path.basename(output) });
  }
  const modelMeta = await archive.json('.workspace/benchmarks/models-dev-2/run.json');
  const modelRows = (await archive.rows('.workspace/benchmarks/models-dev-2/results.jsonl')).map(({ response, ...row }) => row);
  const models = buildReport(modelMeta, modelRows);
  assert.deepEqual(models, await archive.json('benchmarks/results/models-dev-2/summary.json'));
  await fs.writeFile(path.join(out, 'models-dev-2.json'), JSON.stringify(models, null, 2) + '\n', { flag: 'wx' });
  checks.push({ study: 'models-dev-2', fullJson: 'equal', output: 'models-dev-2.json' });
  runNode(['tools/evaluation/diagnostic/summarize.mjs', '--runs', 'benchmarks/configs/qwen-archive-report.json', '--out', path.join(out, 'qwen')]);
  assert.deepEqual(await read(path.join(out, 'qwen/summary.json')), await archive.json('benchmarks/results/qwen-pipeline-diagnostic-v1/summary.json'));
  checks.push({ study: 'qwen-pipeline-diagnostic-v1', fullJson: 'equal', output: 'qwen/summary.json' });
  const phoneticOut = path.join(out, 'phonetic-asr-1.7b-v2.json');
  runNode(['tools/evaluation/phonetic/cli.mjs', 'report', '--archive', 'benchmarks/migration/archive.json', '--results',
    'benchmarks/runs/phonetic-asr/phonetic-asr-v2/completed-17b-view', '--out', phoneticOut]);
  assert.deepEqual(await read(phoneticOut), await archive.json('benchmarks/results/phonetic-asr-1.7b-v2/summary.json'));
  checks.push({ study: 'phonetic-asr-1.7b-v2', fullJson: 'equal', output: path.basename(phoneticOut) });
  const result = { status: 'passed', inference: 'not executed', verification, checks,
    limitations: ['Historical extra scorer fields are allowed; all published fields except generation timestamp must match.',
      'Edit-learning native measurements, scene classifier and prefix/OCR subreports are retained and hash-verified; they are not regenerated by this entry point.',
      'Compiled binaries, weights and environments are not included or verified as provisioned inference artifacts.'] };
  await fs.writeFile(path.join(out, 'verification.json'), JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
  console.log(`Verified ${checks.length} report families; wrote ${out}`);
} else throw new Error('Expected verify or report');
