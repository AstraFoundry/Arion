// Build and development use identical resource and native compiler settings.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, readFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { developmentSigningIdentity, macOSDesktopProcesses, requireDesktopStopped } from './lib/desktop-development.mjs';
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
if (process.platform === 'darwin' && process.arch !== 'arm64') throw new Error('macOS desktop builds require Apple silicon; Intel macOS is unsupported');
const config = JSON.parse(readFileSync(path.join(desktop, 'desktop.json'), 'utf8'));
const settings = operation === 'build' && process.platform === 'darwin' ? requireIMEProvisioningSettings(macosSigningSettings(process.env)) : null;
let developmentIdentity;
if (operation === 'dev' && process.platform === 'darwin') {
  requireDesktopStopped(macOSDesktopProcesses());
  developmentIdentity = developmentSigningIdentity(process.env);
}
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
  const args = ['tool', 'mygo', operation];
  if (operation === 'dev' && process.platform === 'darwin') {
    const checkout = createHash('sha256').update(root).digest('hex').slice(0, 16);
    const output = path.join(os.homedir(), 'Library', 'Caches', 'Arion', 'desktop-dev', checkout);
    args.push('-out', output, '-sign', developmentIdentity, '-build-only');
    console.log(`Development output: ${output}`);
  }
  execFileSync('go', args, { cwd: desktop, env, stdio: 'inherit' });
  if (operation === 'dev' && process.platform === 'darwin') {
    const checkout = createHash('sha256').update(root).digest('hex').slice(0, 16);
    const directory = path.join(os.homedir(), 'Library', 'Caches', 'Arion', 'desktop-dev', checkout, 'darwin-arm64');
    const bundle = path.join(directory, 'Arion Dev.app');
    const executable = path.join(bundle, 'Contents', 'MacOS', 'Arion Dev');
    // LaunchServices assigns privacy requests to the app instead of the terminal
    // or coding agent responsible for a directly spawned executable.
    requireDesktopStopped(macOSDesktopProcesses());
    const launchArgs = ['--stdout', path.join(directory, 'dev-stdout.log'), '--stderr', path.join(directory, 'dev-stderr.log'), '--env', 'MYGO_ENV=development'];
    for (const key of ['ARION_DATA_DIR', 'ARION_E2E_CONTROL', 'ARION_OFFLINE']) {
      if (env[key] !== undefined) launchArgs.push('--env', `${key}=${env[key]}`);
    }
    execFileSync('open', [...launchArgs, bundle], { stdio: 'inherit' });
    let running;
    for (let attempt = 0; attempt < 50; attempt++) {
      running = macOSDesktopProcesses().find(p => p.executable === executable);
      if (running) break;
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    if (!running) throw new Error(`The requested development app did not stay running. Inspect ${path.join(directory, 'dev-stderr.log')}`);
    console.log(JSON.stringify({ ...running, sha256: createHash('sha256').update(readFileSync(executable)).digest('hex') }));
    console.log('Quit Arion Dev before running make desktop-dev again. macOS development uses explicit rebuilds, without a direct-process live-reload watcher.');
  }
  if (operation === 'build') execFileSync(process.execPath, [path.join(root, 'scripts/verify-desktop-package.mjs')], { cwd: root, env, stdio: 'inherit' });
} finally {
  process.removeListener('exit', cleanup);
  cleanup();
}
