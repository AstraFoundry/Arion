// Explicit real-key lifecycle validation. Preparation never runs the harness.
// Execution refuses a pre-existing production key and never exports key bytes.
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildIMELearning } from './lib/build-ime-learning.mjs';
import { macosSigningSettings, signingRunner, stripAppleCredentials } from './lib/desktop-signing.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const usage = 'Usage: node scripts/test-keyboard-keychain.mjs --plan | --prepare | --execute --prepared ABSOLUTE_PATH';
const mode = process.argv[2];
if (!['--plan', '--prepare', '--execute', '--execute-capability'].includes(mode)) throw new Error(usage);
if ((!mode.startsWith('--execute') && process.argv.length !== 3) ||
    (mode.startsWith('--execute') && (process.argv.length !== 5 || process.argv[3] !== '--prepared' || !path.isAbsolute(process.argv[4])))) throw new Error(usage);
if (process.platform !== 'darwin') throw new Error('Real keyboard Keychain validation requires macOS.');
const runtime = path.join(root, 'apps/desktop/resources/darwin/keyboard');
const source = path.join(root, 'apps/desktop/ime/macos');
const config = JSON.parse(readFileSync(path.join(root, 'apps/desktop/desktop.json'), 'utf8'));
const inputInfo = JSON.parse(execFileSync('plutil', ['-convert', 'json', '-o', '-', path.join(source, 'Info.plist')], { encoding: 'utf8' }));
const digest = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const save = (file, value) => writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o600 });
if (mode === '--plan') {
  process.stdout.write(`${JSON.stringify({ phases: ['guard-absence', 'writer', 'fresh-process-reader', 'owned-delete', 'fresh-process-missing', 'cleanup-absence'],
    productionIdentifier: inputInfo.CFBundleIdentifier, backend: 'unchanged production registration and OS Keychain',
    execution: 'explicit --execute only after preparation; no source registration or IMK dispatch',
    ownership: 'abort on existing key; no key data export; exact persistent-reference deletion only',
    interaction: 'noninteractive process-scoped legacy policy plus Data Protection authentication-UI denial; report blocked if interaction needed' }, null, 2)}\n`);
  process.exit(0);
}
if (mode === '--prepare') {
  const envFile = path.join(root, '.env');
  if (existsSync(envFile)) process.loadEnvFile(envFile);
  const settings = macosSigningSettings(process.env);
  if (!settings) throw new Error('Configured Developer ID signing is required; ad hoc signing is not accepted.');
  const arch = { arm64: 'arm64', x64: 'x86_64' }[process.arch];
  if (!arch) throw new Error('Unsupported native test architecture.');
  const runID = randomUUID();
  const directory = path.join(root, '.workspace/tests', `keyboard-keychain-${runID}`);
  mkdirSync(directory, { recursive: true, mode: 0o700 });
  const preparedFile = path.join(directory, 'prepared.json');
  const manifest = { runID, directory, profile: path.join(directory, 'owned-profile'),
    data: path.join(runtime, 'data'), binary: path.join(directory, 'keychain-learning-test'),
    identifier: inputInfo.CFBundleIdentifier, identity: settings.identity,
    prepared: false, executed: false, boundary: 'real OS key lifecycle only; no installed input-source or IMK dispatch',
    backup: { requiredKeyBaseline: 'absent', keyDataExported: false, profileWasAbsent: true, inputSourcesTouched: false } };
  save(preparedFile, manifest);
  const env = stripAppleCredentials(process.env);
  const run = args => execFileSync('xcrun', args, { cwd: root, stdio: 'inherit', env });
  const learning = buildIMELearning({ root, outputDir: directory, arch,
    minimumSystemVersion: config.macos.minimumSystemVersion, testing: false });
  const objects = ['keyboard.m', 'tests/keychain_learning_test.m'].map((member, index) => {
    const object = path.join(directory, `native-${index}.o`);
    run(['clang', '-fobjc-arc', '-O2', '-Wall', '-Wextra', '-Werror', '-target', learning.target,
      '-I', path.join(runtime, 'include'), '-c', path.join(source, member), '-o', object]);
    return object;
  });
  const lib = path.join(directory, 'lib/librime.1.dylib');
  mkdirSync(path.dirname(lib));
  copyFileSync(path.join(runtime, 'lib/librime.1.dylib'), lib);
  run([learning.linker, '-target', learning.target, ...objects, ...learning.objects, lib,
    '-Xlinker', '-rpath', '-Xlinker', '@executable_path/lib', '-framework', 'Cocoa', '-framework', 'LocalAuthentication', ...learning.linkArgs, '-o', manifest.binary]);
  // Use an already available configured identity. Do not import certificates,
  // change the keychain search list, or provision notarization credentials.
  const sign = signingRunner([settings.certificate, settings.certificatePassword, settings.appPassword]);
  sign('codesign', ['--force', '--sign', settings.identity, '--options', 'runtime', '--timestamp', lib]);
  sign('codesign', ['--force', '--sign', settings.identity, '--identifier', manifest.identifier, '--options', 'runtime', '--timestamp', manifest.binary]);
  sign('codesign', ['--verify', '--strict', lib]);
  sign('codesign', ['--verify', '--strict', manifest.binary]);
  const identity = sign('codesign', ['--display', '--verbose=4', manifest.binary]);
  if (!identity.includes(`Identifier=${manifest.identifier}`) || !identity.includes(`TeamIdentifier=${settings.identity.match(/\(([A-Z0-9]{10})\)$/)[1]}`)) throw new Error('Prepared harness signing identity differs from the IMK host.');
  manifest.sourceIdentities = ['keyboard.h', 'keyboard.m', 'learning.h', 'learning.mm', 'learning_crypto.swift', 'tests/keychain_learning_test.m']
    .map(member => ({ path: path.relative(root, path.join(source, member)), sha256: digest(path.join(source, member)) }));
  manifest.binarySha256 = digest(manifest.binary);
  manifest.librarySha256 = digest(lib);
  manifest.prepared = true;
  save(preparedFile, manifest);
  process.stdout.write(`Prepared signed harness without executing it: ${preparedFile}\n`);
  process.exit(0);
}

const preparedFile = process.argv[4];
const manifest = JSON.parse(readFileSync(preparedFile, 'utf8'));
if (!manifest.prepared || manifest.executed || path.dirname(preparedFile) !== manifest.directory ||
    manifest.binarySha256 !== digest(manifest.binary) || manifest.librarySha256 !== digest(path.join(manifest.directory, 'lib/librime.1.dylib'))) throw new Error('Prepared harness is invalid, changed, or already executed.');
if (existsSync(manifest.profile)) throw new Error('Owned test profile must be absent before execution.');
manifest.runnerSha256 = digest(fileURLToPath(import.meta.url));
manifest.executed = true;
manifest.results = [];
save(preparedFile, manifest);
const receipt = path.join(manifest.directory, 'private-ownership-receipt.json');
const execute = phase => {
  const report = path.join(manifest.directory, `${phase}.json`);
  const result = spawnSync(manifest.binary, [phase, manifest.data, manifest.profile, receipt, report, manifest.runID],
    { cwd: root, stdio: 'inherit', env: stripAppleCredentials(process.env) });
  const observation = existsSync(report) ? JSON.parse(readFileSync(report, 'utf8')) : { phase, passed: false, failure: 'phase produced no report' };
  observation.exitStatus = result.status;
  if (result.error) observation.processError = result.error.code;
  if (result.signal) observation.signal = result.signal;
  manifest.results.push(observation);
  save(preparedFile, manifest);
  if (result.status !== 0 || !observation.passed) throw new Error(`Real-key phase failed: ${phase}; inspect ${report}`);
};
let guardPassed = false;
let passed = false;
try {
  execute('guard'); guardPassed = true;
  manifest.capabilityOnly = mode === '--execute-capability';
  for (const phase of manifest.capabilityOnly ? ['capability'] : ['writer', 'reader', 'delete', 'missing']) execute(phase);
  passed = true;
} finally {
  if (guardPassed) {
    try {
      execute('cleanup');
      rmSync(manifest.profile, { recursive: true, force: true });
      rmSync(receipt);
      manifest.restored = true;
    } catch (error) {
      passed = false;
      manifest.restored = false;
      manifest.cleanupError = error.message;
    }
  }
  const writer = manifest.results.find(result => result.phase === 'writer');
  const attributes = writer?.keyAttributes;
  const deviceClassVerified = attributes?.accessiblePresent && attributes.matchesAfterFirstUnlockThisDeviceOnly &&
    attributes.synchronizablePresent && attributes.synchronizable === false &&
    writer?.backingEvidence?.dataProtection?.sameOwnedReference;
  manifest.assurance = {
    lifecyclePassed: passed && !manifest.capabilityOnly,
    capabilityPassed: manifest.capabilityOnly ? passed : undefined,
    deviceBoundProtectionVerification: !attributes ? 'not-observed' : deviceClassVerified ? 'verified' : 'not-established',
    scope: 'Lifecycle success alone does not verify the requested accessibility class or backing-store policy.',
  };
  manifest.passed = passed;
  save(preparedFile, manifest);
  process.stdout.write(`Real-key lifecycle evidence: ${preparedFile}\n`);
}
if (!passed) process.exitCode = 1;
