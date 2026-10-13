import { spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
if (!['darwin', 'win32'].includes(process.platform)) {
  console.error('Native E2E requires macOS or Windows with an interactive desktop session.');
  process.exit(2);
}
const desktop = path.join(root, 'apps/desktop');
const output = path.join(root, '.workspace/e2e', process.platform);
const env = { ...process.env, ARION_NATIVE_E2E: '1', ARION_E2E_OUTPUT: output };
const testFlags = ['-count=1', '-timeout=120s', '-v'];

function run(command, args, cwd) {
  const result = spawnSync(command, args, { cwd, stdio: 'inherit', env });
  if (result.error) throw result.error;
  return result.status ?? 1;
}

if (process.platform === 'darwin') {
  process.exitCode = run('go', ['test', '-tags', 'arion_e2e', ...testFlags, './e2e'], desktop);
} else {
  // Go test binaries carry no manifest; Windows reads an external one next to
  // an executable without an embedded manifest (see e2e/windows.manifest).
  const bin = path.join(output, 'bin');
  const exe = path.join(bin, 'e2e.test.exe');
  mkdirSync(bin, { recursive: true });
  const built = run('go', ['test', '-c', '-tags', 'arion_e2e', '-o', exe, './e2e'], desktop);
  if (built !== 0) {
    process.exitCode = built;
  } else {
    copyFileSync(path.join(desktop, 'e2e', 'windows.manifest'), `${exe}.manifest`);
    process.exitCode = run(exe, testFlags.map(flag => flag.replace(/^-/, '-test.')), path.join(desktop, 'e2e'));
  }
}
