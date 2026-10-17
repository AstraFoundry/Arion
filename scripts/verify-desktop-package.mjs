// Verify required resources in the packaged app, never mutate signed output.
import { execFileSync } from 'node:child_process';
import { readFileSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { resolveCenter, centerFingerprint, isStandardCenter } from './lib/center-profile.mjs';
import { macosSigningSettings } from './lib/desktop-signing.mjs';
import { verifyMacOSDistribution } from './lib/verify-macos-distribution.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const config = JSON.parse(readFileSync(path.join(root, 'apps/desktop/desktop.json'), 'utf8'));
const pins = JSON.parse(readFileSync(path.join(root, 'scripts/llm-runtime.json'), 'utf8'));
const goos = { darwin: 'darwin', win32: 'windows' }[process.platform];
const arch = process.arch === 'x64' ? 'amd64' : process.arch;
if (!goos) throw new Error('Desktop package verification requires macOS or Windows');
const platform = `${goos}-${arch}`;
const built = path.join(root, 'apps/desktop', config.out, platform);
const resources = goos === 'darwin' ? path.join(built, `${config.name}.app`, 'Contents', 'Resources') : built;
const requireFile = file => {
  if (!statSync(file).isFile() || statSync(file).size === 0) throw new Error(`Missing or empty package file: ${file}`);
};
const systemVersion = value => {
  const [major, minor = 0, patch = 0] = value.split('.').map(Number);
  return major * 1000000 + minor * 1000 + patch;
};
const binaries = goos === 'darwin'
  ? [path.join(built, `${config.name}.app/Contents/MacOS/${config.name}`), path.join(resources, 'lib/libsherpa-onnx-c-api.dylib'), path.join(resources, 'lib/libonnxruntime.dylib'), path.join(resources, 'ArionInput.app/Contents/MacOS/ArionInput')]
  : [path.join(built, `${config.name}.exe`), path.join(resources, 'sherpa-onnx-c-api.dll'), path.join(resources, 'onnxruntime.dll')];
for (const file of ['Arion-LICENSE', 'model-catalog.json', 'go-modules.json']) requireFile(path.join(resources, 'licenses', file));
const notices = JSON.parse(readFileSync(path.join(resources, 'licenses/go-modules.json'), 'utf8'));
for (const record of notices) for (const file of record.licenses) requireFile(path.join(resources, 'licenses', file));
const native = JSON.parse(readFileSync(path.join(resources, 'licenses/native-notices.json'), 'utf8'));
for (const spec of native.files) requireFile(path.join(resources, 'licenses', spec.name));
const spec = pins.platforms[platform];
if (spec) {
  binaries.push(path.join(resources, 'llm', goos === 'windows' ? 'arion-llm.exe' : 'arion-llm'));
  for (const archive of spec.archives) for (const name of Object.values(archive.members)) {
    const file = path.join(resources, 'llm', name);
    requireFile(file);
    if (/\.(dylib|dll)$/.test(name)) binaries.push(file);
  }
}
const expectedCenter = resolveCenter(process.env);
const embeddedCenter = JSON.parse(execFileSync(binaries[0], ['--print-center-profile'], { encoding: 'utf8', env: { PATH: process.env.PATH, SystemRoot: process.env.SystemRoot, HOME: process.env.HOME } }));
if (JSON.stringify(embeddedCenter.profile) !== JSON.stringify(expectedCenter) || embeddedCenter.fingerprint !== centerFingerprint(expectedCenter)) throw new Error('Packaged center profile does not match build configuration');
console.log(`Verified embedded center profile ${embeddedCenter.fingerprint}.`);
for (const file of binaries) {
  requireFile(file);
  if (goos === 'darwin') {
    const architectures = execFileSync('lipo', ['-archs', file], { encoding: 'utf8' }).trim().split(/\s+/);
    if (!architectures.includes(arch === 'amd64' ? 'x86_64' : arch)) throw new Error(`Wrong architecture in ${file}: ${architectures}`);
    const commands = execFileSync('otool', ['-l', file], { encoding: 'utf8' });
    const minimums = [...commands.matchAll(/\bminos ([\d.]+)|cmd LC_VERSION_MIN_MACOSX[\s\S]*?\bversion ([\d.]+)/g)].map(match => match[1] ?? match[2]);
    if (!minimums.length || minimums.some(version => systemVersion(version) > systemVersion(config.macos.minimumSystemVersion))) throw new Error(`Native minimum OS exceeds macOS ${config.macos.minimumSystemVersion}: ${file} (${minimums.join(', ')})`);
  } else {
    const bytes = readFileSync(file);
    if (bytes.toString('ascii', 0, 2) !== 'MZ' || bytes.readUInt16LE(bytes.readUInt32LE(0x3c) + 4) !== 0x8664) throw new Error(`Not a Windows x64 binary: ${file}`);
  }
}
if (goos === 'darwin') {
  const appPlist = path.join(built, `${config.name}.app/Contents/Info.plist`);
  const imePlist = path.join(resources, 'ArionInput.app/Contents/Info.plist');
  const field = (plist, name) => execFileSync('/usr/libexec/PlistBuddy', ['-c', `Print :${name}`, plist], { encoding: 'utf8' }).trim();
  for (const [plist, identifier] of [[appPlist, config.identifier], [imePlist, `${config.identifier}.inputmethod`]]) {
    if (field(plist, 'CFBundleIdentifier') !== identifier || field(plist, 'CFBundleShortVersionString') !== config.version.split('-')[0] || field(plist, 'LSMinimumSystemVersion') !== config.macos.minimumSystemVersion) throw new Error(`Package identity, version or minimum OS mismatch: ${plist}`);
  }
  for (const locale of ['en', 'zh-Hans']) requireFile(path.join(resources, `${locale}.lproj/InfoPlist.strings`));
  execFileSync(process.execPath, [path.join(root, 'scripts/verify-macos-app.mjs')], { stdio: 'inherit' });
  const signing = macosSigningSettings(process.env);
  if (signing) {
    verifyMacOSDistribution({ directory: built, config, platform, binaries, identity: signing.identity, updates: Boolean(process.env.MYGO_UPDATER_PRIVATE_KEY) && isStandardCenter(expectedCenter) });
    console.log('Verified Developer ID signatures, hardened runtime, notarization tickets and Gatekeeper acceptance.');
  }
}
console.log(`Verified ${platform} resources, licenses and architectures; local rewriting ${spec ? 'bundled' : 'unsupported'}.`);
