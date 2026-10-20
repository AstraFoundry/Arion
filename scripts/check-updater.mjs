import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const directory = fileURLToPath(new URL('../third_party/mygo/', import.meta.url));
execFileSync('go', ['test', '.', './internal/update', '-short'], {
  cwd: directory, env: { ...process.env, GOWORK: 'off', CGO_ENABLED: '0' }, stdio: 'inherit',
});
