import { spawnSync } from 'node:child_process';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const args = process.argv.slice(2);
const config = {};
for (let i = 0; i < args.length; i += 2) {
  const name = args[i].replace(/^--/, '');
  if (!['out', 'runtime', 'model'].includes(name) || !args[i + 1] || name in config) throw new Error('Expected --out NEWDIR --runtime PATH --model PATH');
  config[name] = path.resolve(args[i + 1]);
}
for (const name of ['out', 'runtime', 'model']) if (!config[name]) throw new Error(`Missing --${name}`);
mkdirSync(config.out);
const plan = [
  ['development', 'baseline'], ['development', 'no-prior'], ['development', 'no-screen'], ['development', 'candidate'],
  ['validation', 'candidate'], ['validation', 'baseline'],
  ['native-pairs', 'baseline'], ['native-pairs', 'candidate'],
];
writeFileSync(path.join(config.out, 'plan.json'), JSON.stringify({ config, plan, limits: 'Serial paired text-stage generation. Actual native input origins stay fixture-specific. Nonrandom arm order and development-host drift limit latency attribution.' }, null, 2) + '\n', { flag: 'wx' });
const results = [];
for (const [corpus, arm] of plan) {
  const command = [path.join(root, 'scripts/rewrite-stage-command.mjs'), 'run', '--arm', arm,
    '--cases', path.join(root, `benchmarks/reproduction/rewrite-stage-preservation-v1/${corpus}.json`),
    '--out', path.join(config.out, `${corpus}-${arm}`), '--runtime', config.runtime, '--model', config.model];
  const start = performance.now();
  const result = spawnSync(process.execPath, command, { cwd: root, stdio: 'inherit', shell: false });
  results.push({ corpus, arm, command, code: result.status, error: result.error?.message, elapsedMs: performance.now() - start });
  writeFileSync(path.join(config.out, 'progress.json'), JSON.stringify(results, null, 2) + '\n');
  if (result.status !== 0) process.exit(result.status ?? 1);
}
writeFileSync(path.join(config.out, 'complete.json'), JSON.stringify({ status: 'completed', arms: results.length, quality: 'Every output requires manual review; command completion does not pass model quality.' }, null, 2) + '\n', { flag: 'wx' });
