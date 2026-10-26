import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const study = path.join(root, 'benchmarks/reproduction/rime-llm-reranker-v1');
const operation = process.argv[2];
let executable = process.env.EVAL_PYTHON || 'python3';
let args;
if (operation === 'prepare-native') {
  executable = process.execPath;
  args = [path.join(study, 'prepare-native.mjs'), ...process.argv.slice(3)];
} else if (['collect', 'run', 'report', 'warmup'].includes(operation)) {
  args = ['-B', path.join(study, 'study.py'), operation, ...process.argv.slice(3)];
} else if (['verify', 'test', 'provision', 'archive', 'audit', 'controls', 'freeze'].includes(operation)) {
  args = ['-B', path.join(study, operation === 'test' ? 'test_report.py' : `${operation}.py`), ...process.argv.slice(3)];
} else {
  throw new Error('Use provision, prepare-native, collect, warmup, run, report, test, archive or verify');
}
const result = spawnSync(executable, args, { cwd: root, shell: false, stdio: 'inherit' });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
