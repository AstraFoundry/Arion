import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const [operation, ...args] = process.argv.slice(2);
const scripts = {
  run: 'context-repair-v1/run.py',
  structure: 'scene-structure-v3/run.py',
  report: 'context-repair-v1/report.py',
  verify: 'context-repair-v1/verify.py',
  test: 'context-repair-v1/test_report.py',
};
if (!Object.hasOwn(scripts, operation)) throw new Error('Expected run, structure, report, verify or test');
const python = process.env.EVAL_PYTHON || (process.platform === 'win32' ? 'python' : 'python3');
const result = spawnSync(python, [path.join(root, 'benchmarks/reproduction', scripts[operation]), ...args], { cwd: root, stdio: 'inherit' });
if (result.error) throw result.error;
process.exit(result.status ?? 1);
