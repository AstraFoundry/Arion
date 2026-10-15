// Build and development use identical resource and native compiler settings.
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const operation = process.argv[2];
if (!['build', 'dev'].includes(operation)) throw new Error('Use desktop-command.mjs build|dev');
if (!['darwin', 'win32'].includes(process.platform)) throw new Error('Native desktop commands require macOS or Windows');
const config = JSON.parse(readFileSync(path.join(desktop, 'mygo.json'), 'utf8'));
execFileSync(process.execPath, [path.join(root, 'scripts/stage-desktop-resources.mjs')], { cwd: root, stdio: 'inherit' });
const env = { ...process.env, CGO_ENABLED: '1' };
if (process.platform === 'darwin') {
  const minimum = config.macos.minimumSystemVersion;
  env.CC = path.join(root, 'scripts/cc-relocatable.sh');
  env.MACOSX_DEPLOYMENT_TARGET = minimum;
  // Explicit flags also invalidate cgo's cache when the target changes.
  env.CGO_CFLAGS = `-O2 -g -mmacosx-version-min=${minimum}`;
  env.CGO_CXXFLAGS = `-O2 -g -mmacosx-version-min=${minimum}`;
  env.CGO_LDFLAGS = `-mmacosx-version-min=${minimum} -Wl,-rpath,@executable_path/../Resources/lib`;
}
execFileSync('go', ['tool', 'mygo', operation], { cwd: desktop, env, stdio: 'inherit' });
if (operation === 'build') execFileSync(process.execPath, [path.join(root, 'scripts/verify-desktop-package.mjs')], { cwd: root, stdio: 'inherit' });
