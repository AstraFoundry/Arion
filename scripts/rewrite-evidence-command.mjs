import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const [study, operation, ...args] = process.argv.slice(2);
if (!/^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/.test(study ?? '') || !['run', 'ocr', 'report', 'test_report', 'test_report_v2', 'test_preview', 'test_payload', 'verify'].includes(operation)) {
  throw new Error('Expected a research study name and supported operation');
}
const script = path.join(root, 'benchmarks/reproduction', study, `${operation}.py`);
if (!fs.existsSync(script)) throw new Error(`Missing research operation: ${study}/${operation}`);
const python = process.env.EVAL_PYTHON || (process.platform === 'win32' ? 'python' : 'python3');
const result = spawnSync(python, [script, ...args], { cwd: root, stdio: 'inherit', shell: false });
if (result.error) throw result.error;
process.exit(result.status ?? 1);
