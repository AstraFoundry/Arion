import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const result = spawnSync('go', ['test', './ui/shared/copy'], {
  cwd: path.join(root, 'apps/desktop'),
  stdio: 'inherit',
  env: { ...process.env, CGO_ENABLED: '0' },
});
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
