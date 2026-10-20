import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const python = process.env.EVAL_PYTHON || (process.platform === 'win32' ? 'python' : 'python3');
const script = path.join(root, 'benchmarks/reproduction/rewrite-selector-hybrid-v1/preflight.py');
const result = spawnSync(python, [script, ...process.argv.slice(2)], { cwd: root, stdio: 'inherit', shell: false });
if (result.error) throw result.error;
process.exit(result.status ?? 1);
