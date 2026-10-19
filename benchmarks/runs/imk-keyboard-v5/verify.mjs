// Frozen-evidence verification only: no network, runtime, Keychain or OS mutation.
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.dirname(fileURLToPath(import.meta.url));
const read = file => readFileSync(path.join(root, file));
const json = file => JSON.parse(read(file));
const archive = json('archive.json');
const paths = new Set();
for (const entry of archive.files) {
  assert(!path.isAbsolute(entry.path) && !entry.path.split('/').includes('..'));
  assert(!paths.has(entry.path)); paths.add(entry.path);
  const bytes = read(entry.path);
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
const capability = json(archive.keychain.capabilityReport);
assert.equal(capability.passed, false); assert(capability.restored && capability.capabilityOnly);
assert.equal(json('dp-capability/capability.json').addStatus, -34018);
assert.equal(capability.assurance.lifecyclePassed, false);
for (const kind of ['legacy-rejection', 'app-legacy-rejection']) {
  const report = json(`${kind}/prepared.json`);
  assert(report.passed && report.restored && report.legacyRejectionOnly);
  assert.equal(report.assurance.lifecyclePassed, false);
  const rejection = json(`${kind}/legacy-rejection.json`);
  assert(rejection.passed && !rejection.registered);
  assert.equal(rejection.failureCode, 11); assert.equal(rejection.dataProtectionStatus, -25300);
  assert.deepEqual(rejection.vaultInventory, {});
  assert.equal(json(`${kind}/cleanup.json`).finalKeyStatus, -25300);
}
const cleanup = json('app-legacy-rejection/cleanup.json');
assert.equal(cleanup.finalLegacyStatus, -25300); assert.equal(cleanup.finalDataProtectionStatus, -25300);
for (const kind of ['dp-capability', 'legacy-rejection', 'app-legacy-rejection']) {
  const report = json(`${kind}/prepared.json`);
  for (const identity of report.sourceIdentities) {
    const frozen = archive.files.find(entry => entry.path === `${kind}/sources/${identity.path}`);
    assert(frozen, identity.path); assert.equal(frozen.sha256, identity.sha256);
  }
  assert.equal(archive.files.find(entry => entry.path === `${kind}/sources/scripts/test-keyboard-keychain.mjs`).sha256, report.runnerSha256);
  assert.equal(report.backup.keyDataExported, false);
  assert.equal(report.backup.inputSourcesTouched, false);
  assert(report.results.every(result => !result.secretDataRequestedByHarness && result.interactionPolicyRestored === 1));
}
for (const identity of json('app-legacy-rejection/additional-source-identities.json')) {
  const suffix = identity.path.endsWith('.md') ? '.txt' : '';
  assert.equal(archive.files.find(entry => entry.path === `app-legacy-rejection/sources/${identity.path}${suffix}`).sha256, identity.sha256);
}
for (const [file, arch, translation] of [[archive.functional.nativeReport, 'arm64', 'none'], [archive.functional.translatedReport, 'x86_64', 'Rosetta 2']]) {
  const report = json(file);
  assert(report.passed && report.cases.length === 22 && report.cases.every(test => test.passed));
  assert.equal(report.execution.hostArchitecture, 'arm64');
  assert.equal(report.execution.executableArchitecture, arch);
  assert.equal(report.execution.translation, translation);
  assert.equal(report.execution.installedIMEHostDispatch, false);
  assert.equal(report.execution.physicalIntelHost, false);
}
for (const identity of json(archive.functional.identities).sourceIdentities) {
  if (identity.path.endsWith('.dylib')) assert(archive.exclusions.some(item => item.originalPath === identity.path && item.sha256 === identity.sha256));
  else assert.equal(archive.files.find(entry => entry.path === `functional/sources/${identity.path}`).sha256, identity.sha256);
}
assert(read('logs/keychain-strict-synthetic-01.log').toString().includes('Encrypted learning source tests passed.'));
assert(read('logs/keychain-profile-required-01.log').toString().includes('Full Data Protection preparation requires MACOS_IME_PROVISIONING_PROFILE'));
const gui = json(archive.system.report);
assert.equal(gui.mainThread, true); assert.equal(gui.caller, 'NSApplication button action on main thread');
assert(Object.values(gui.statuses).every(status => status === 0));
assert.equal(gui.after.find(source => !source.id.endsWith('.pinyin')).enabled, false);
assert.equal(gui.after.find(source => source.id.endsWith('.pinyin')).enabled, true);
const state = json('system/run-11/state.json');
assert.equal(state.phase, 'restored'); assert.deepEqual(state.errors, []);
assert(state.after.own.length === 2 && state.after.own.every(source => !source.enabled && !source.selected));
const final = json('system/run-11/final-source-query.json');
assert.equal(final.current, 'com.apple.keylayout.ABC'); assert.equal(final.learningKeyStatus, -25300); assert.deepEqual(final.own, []);
assert.equal(json('system/run-11/final-process-query.json').exitStatus, 1);
const ui = json('system/run-11/ui-observations.json');
assert.equal(ui.kind, 'transcribed-actual-tool-observations'); assert.equal(ui.notOriginalScreenshot, true);
assert.equal(ui.actions[0].element, 1); assert.equal(ui.actions[2].element, 3);
assert(ui.actions[2].result.includes('timeoutReached'));
console.log(`PASS: ${archive.files.length} SHA-256 mappings; strict synthetic native/translated 22/22; DP capability failure preserved; two legacy-rejection runs restored; GUI caller still disabled parent and later cleanup verified.`);
