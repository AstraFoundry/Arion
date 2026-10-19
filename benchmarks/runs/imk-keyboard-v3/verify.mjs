// Verifies frozen evidence only; no runtime, network, Keychain or UI operation.
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
const root = path.dirname(fileURLToPath(import.meta.url));
const read = file => readFileSync(path.join(root, file), 'utf8');
const json = file => JSON.parse(read(file));
const archive = json('archive.json');
const paths = new Set();
for (const entry of archive.files) {
  assert(!path.isAbsolute(entry.path) && !entry.path.split('/').includes('..'));
  assert(!paths.has(entry.path)); paths.add(entry.path);
  const bytes = readFileSync(path.join(root, entry.path));
  assert.equal(bytes.length, entry.sizeBytes, entry.path);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), entry.sha256, entry.path);
}
function inventory(directory) {
  return readdirSync(directory).flatMap(name => {
    const file = path.join(directory, name);
    return statSync(file).isDirectory() ? inventory(file) : [path.relative(root, file).split(path.sep).join('/')];
  });
}
assert.deepEqual(inventory(root).filter(file => file !== 'archive.json').sort(), [...paths].sort());
const report = json(archive.functional.report);
assert(report.passed && report.cases.length === 22 && report.cases.every(test => test.passed));
assert.equal(report.execution.hostArchitecture, 'arm64');
assert.equal(report.execution.executableArchitecture, 'x86_64');
assert.equal(report.execution.translation, 'Rosetta 2');
assert.equal(report.execution.launcher, '/usr/bin/arch -x86_64');
assert.equal(report.execution.physicalIntelHost, false);
assert.equal(report.execution.installedIMEHostDispatch, false);
for (const identity of json(archive.functional.identities)) {
  if (identity.path.endsWith('.dylib')) {
    assert(archive.exclusions.some(item => item.path === identity.path && item.sha256 === identity.sha256));
  } else {
    const frozen = archive.files.find(item => item.originalPath === identity.path && item.kind === 'identity-matching-attempt-13-source-or-manifest');
    assert(frozen, identity.path); assert.equal(frozen.sha256, identity.sha256);
  }
}
const metadata = json(archive.metadataPatch.report);
assert.equal(metadata.runnerSha256, archive.metadataPatch.sourceSha256);
assert.equal(createHash('sha256').update(read(archive.metadataPatch.source)).digest('hex'), metadata.runnerSha256);
assert.equal(metadata.nativeArm64.translated, false);
assert.equal(metadata.translatedX86_64.translated, true);
assert.equal(metadata.absentOID.oidPresent, false);
assert(metadata.malformedRejected && metadata.commandErrorRejected);
assert(metadata.boundary.includes('Metadata probe verification only'));
const native = json(archive.completedControls.nativeReport);
assert(native.passed && native.cases.length === 22 && native.cases.every(test => test.passed));
assert(read(archive.completedControls.signedDesktopBuild).includes('Verified Developer ID signatures, hardened runtime, notarization tickets and Gatekeeper acceptance.'));
assert(read(archive.completedControls.desktopCheck).includes('Encrypted learning source tests passed.'));
assert(read(archive.completedControls.imeRace).includes('arion.local/desktop/infrastructure/imebridge'));
assert(read(archive.completedControls.i18n).includes('i18n:check'));
const state = json(archive.systemUI.restoration);
assert.equal(state.phase, 'restored'); assert.deepEqual(state.errors, []);
for (const source of [state.baseline, state.after]) {
  assert.equal(source.current, 'com.apple.keylayout.ABC');
  assert.equal(source.learningKeyStatus, -25300);
  assert.deepEqual(source.own, []);
}
assert.equal(archive.systemUI.installButtonExercised, false);
assert.equal(archive.systemUI.installedIMEHostDispatch, false);
console.log(`PASS: ${archive.files.length} SHA-256 mappings; translated x86_64 22/22 and native final-check 22/22; later metadata-only patch kept separate; completed signed/control logs; restored UI attempt without installation acceptance.`);
