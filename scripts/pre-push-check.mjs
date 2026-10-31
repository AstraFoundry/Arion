import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

function run(script, args = []) {
  const result = spawnSync(process.execPath, [path.join(root, script), ...args], { cwd: root, stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}

// Format and the repository lint only. Desktop tests, native E2E, and go test stay in CI.
run('scripts/check-go.mjs', ['format-check']);
run('scripts/check-repository.mjs');
console.log('pre-push: format and repository checks passed');
