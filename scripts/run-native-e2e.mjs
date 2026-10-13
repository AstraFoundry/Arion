import { spawnSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
if (!['darwin', 'win32'].includes(process.platform)) {
  console.error('Native E2E requires macOS or Windows with an interactive desktop session.');
  process.exit(2);
}
const result = spawnSync('go', ['test', '-tags', 'arion_e2e', '-count=1', '-timeout=120s', '-v', './e2e'], {
  cwd: path.join(root, 'apps/desktop'), stdio: 'inherit',
  env: { ...process.env, ARION_NATIVE_E2E: '1', ARION_E2E_OUTPUT: path.join(root, '.workspace/e2e', process.platform) },
});
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
