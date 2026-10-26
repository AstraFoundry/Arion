import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const directory = fileURLToPath(new URL('../third_party/mygo/', import.meta.url));
execFileSync('go', ['test', '.', './internal/update', '-short'], {
  cwd: directory, env: { ...process.env, GOWORK: 'off', CGO_ENABLED: '0' }, stdio: 'inherit',
});
if (process.platform === 'win32') {
  execFileSync('go', ['test', './cmd/mygo', '-run', '^TestWindowsInstaller$', '-count=1'], {
    cwd: directory, env: { ...process.env, GOWORK: 'off', CGO_ENABLED: '0' }, stdio: 'inherit',
  });
}
if (process.argv.includes('--native')) {
  if (process.platform !== 'darwin') throw new Error('The packaged updater lifecycle harness requires macOS; run Windows acceptance on Windows.');
  execFileSync('go', ['test', '.', '-run', '^TestPreparedUpdateNativeLifecycle$', '-count=1', '-v'], {
    cwd: directory, env: { ...process.env, GOWORK: 'off', CGO_ENABLED: '0', MYGO_UPDATER_E2E: '1' }, stdio: 'inherit',
  });
}
