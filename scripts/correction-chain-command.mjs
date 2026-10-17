// Research-only chain reproduction; no model acquisition during ordinary checks.
import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'benchmarks/reproduction/correction-chain-v1');
const python = process.env.EVAL_PYTHON || path.join(root, '.workspace/correction-chain/venv/bin/python');
const [operation, ...args] = process.argv.slice(2);
function run(command, argv, cwd = root) {
  const result = spawnSync(command, argv, { cwd, stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}
if (operation === 'env') {
  run('uv', ['venv', path.dirname(path.dirname(python)), '--python', '3.12']);
  run('uv', ['pip', 'sync', '--python', python, path.join(source, 'requirements-resolved.txt')]);
} else if (operation === 'test') {
  run(process.execPath, ['--test', path.join(source, 'report.test.mjs')]);
  run(python, [path.join(source, 'test_correct.py')]);
} else if (operation === 'build') {
  for (const name of ['arion-rewrite-bench', 'arion-llm']) run('go', ['build', '-o', path.join(root, '.workspace/correction-chain', name), `./cmd/${name}`], path.join(root, 'apps/desktop'));
} else if (['report', 'review'].includes(operation)) {
  run(process.execPath, [path.join(source, operation + '.mjs'), ...args]);
} else if (['provision', 'fixtures', 'run'].includes(operation)) {
  const script = { provision: 'provision.py', fixtures: 'build_fixtures.py', run: 'run.py' }[operation];
  run(python, [path.join(source, script), ...args]);
} else {
  throw new Error('Expected env, build, provision, fixtures, run, report or test');
}
