// Build and development use identical resource and native compiler settings.
import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { inspectIMEProvisioning } from './lib/ime-provisioning.mjs';
import { prepareCenterBuild } from './lib/center-profile.mjs';
import { macosSigningSettings, requireIMEProvisioningSettings, prepareMacOSSigning, stripAppleCredentials } from './lib/desktop-signing.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const envFile = path.join(root, '.env');
if (existsSync(envFile)) process.loadEnvFile(envFile);
const desktop = path.join(root, 'apps', 'desktop');
const operation = process.argv[2];
if (!['build', 'dev'].includes(operation)) throw new Error('Use desktop-command.mjs build|dev');
if (!['darwin', 'win32'].includes(process.platform)) throw new Error('Native desktop commands require macOS or Windows');
const config = JSON.parse(readFileSync(path.join(desktop, 'desktop.json'), 'utf8'));
const settings = operation === 'build' && process.platform === 'darwin' ? requireIMEProvisioningSettings(macosSigningSettings(process.env)) : null;
let signing;
let center;
const cleanup = () => {
  try { signing?.cleanup(); } finally { center?.cleanup(); }
};
process.once('exit', cleanup);
for (const [signal, status] of [['SIGINT', 130], ['SIGTERM', 143]]) process.once(signal, () => process.exit(status));
try {
  center = prepareCenterBuild(process.env);
  signing = settings ? prepareMacOSSigning(settings) : null;
  if (settings) {
    const info = JSON.parse(execFileSync('plutil', ['-convert', 'json', '-o', '-', path.join(desktop, 'ime/macos/Info.plist')], { encoding: 'utf8' }));
    inspectIMEProvisioning({ profilePath: settings.imeProvisioningProfile, identity: settings.identity, bundleIdentifier: info.CFBundleIdentifier });
  }
  const env = { ...stripAppleCredentials(process.env), ...center.env, ...signing?.env, CGO_ENABLED: '1' };
  if (settings) env.ARION_IME_SIGNED_BUILD = '1';
  else delete env.ARION_IME_SIGNED_BUILD;
  console.log(`Center profile: ${center.profile.url} (${center.profile.authMode}, ${center.fingerprint})`);
  if (process.platform === 'darwin') {
    const minimum = config.macos.minimumSystemVersion;
    env.CC = path.join(root, 'scripts/cc-relocatable.sh');
    env.MACOSX_DEPLOYMENT_TARGET = minimum;
    // Explicit flags also invalidate cgo's cache when the target changes.
    env.CGO_CFLAGS = `-O2 -g -mmacosx-version-min=${minimum}`;
    env.CGO_CXXFLAGS = `-O2 -g -mmacosx-version-min=${minimum}`;
    env.CGO_LDFLAGS = `-mmacosx-version-min=${minimum}`;
  }
  execFileSync(process.execPath, [path.join(root, 'scripts/stage-desktop-resources.mjs')], { cwd: root, env, stdio: 'inherit' });
  execFileSync('go', ['tool', 'mygo', operation], { cwd: desktop, env, stdio: 'inherit' });
  if (operation === 'build') execFileSync(process.execPath, [path.join(root, 'scripts/verify-desktop-package.mjs')], { cwd: root, env, stdio: 'inherit' });
} finally {
  process.removeListener('exit', cleanup);
  cleanup();
}
