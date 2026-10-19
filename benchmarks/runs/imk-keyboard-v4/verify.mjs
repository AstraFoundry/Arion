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
for (const run of ['08', '09']) {
  const registration = json(`system/run-${run}/registration.json`);
  assert(registration.sources.every(source => !source.selected));
  const parent = registration.sources.find(source => !source.id.endsWith('.pinyin'));
  assert.equal(parent.enabled, false);
  assert.equal(registration.sources.find(source => source.id.endsWith('.pinyin')).enabled, true);
  assert.equal(read(`system/run-${run}/select-status.txt`).toString().trim(), '-50');
  const state = json(`system/run-${run}/state.json`);
  assert.equal(state.phase, 'restored'); assert.deepEqual(state.errors, []);
  assert.equal(state.after.current, 'com.apple.keylayout.ABC');
  assert.equal(state.after.learningKeyStatus, -25300);
  assert(state.after.own.every(source => !source.enabled && !source.selected));
}
assert.equal(json('system/run-08/observations.json').status, 'restore-failed');
assert(json('system/run-08/state.json').initialCleanupErrors.length > 0);
assert.equal(json('system/run-08/recovery.json').finalControllerPhase, 'restored');
assert.equal(json('system/run-09/notarization.json').status, 'Accepted');
assert(read('system/run-09/gatekeeper.txt').toString().includes('Notarized Developer ID'));
const control = json('system/run-10/control.json');
assert(control.attempted && control.restored);
assert.equal(control.enableStatus, 0); assert.equal(control.disableStatus, 0);
const enabled = state => state.find(source => source.id === control.source).enabled;
assert.equal(enabled(control.before), false); assert.equal(enabled(control.afterEnable), true);
assert.equal(enabled(control.afterDisable), false);
assert.deepEqual(control.before, control.afterDisable);
for (const key of ['currentBefore', 'currentAfterEnable', 'currentAfterDisable']) assert.equal(control[key], 'com.apple.keylayout.ABC');
const lifecycle = json(archive.keychain.report);
assert(lifecycle.passed && lifecycle.restored && lifecycle.executed && lifecycle.results.length === 6);
assert.equal(lifecycle.assurance.deviceBoundProtectionVerification, 'not-established');
assert.equal(lifecycle.backup.keyDataExported, false);
assert.equal(lifecycle.backup.inputSourcesTouched, false);
for (const result of lifecycle.results) {
  assert(result.passed && result.exitStatus === 0 && !result.secretDataRequestedByHarness);
  assert.equal(result.interactionPolicyRestored, 1);
  assert.equal(json(`keychain/${result.phase}.json`).passed, true);
}
const writer = json('keychain/writer.json');
const reader = json('keychain/reader.json');
assert.equal(writer.committed, '事实');
assert.equal(writer.beforeCandidates[0], '实施'); assert.equal(reader.afterRestartCandidates[0], '事实');
assert.equal(writer.backingEvidence.legacy.status, 0);
assert.equal(writer.backingEvidence.legacy.sameOwnedReference, 1);
assert.equal(writer.backingEvidence.dataProtection.status, -25300);
assert.equal(writer.keyAttributes.accessiblePresent, 0);
assert.equal(writer.keyAttributes.synchronizablePresent, 0);
const missing = json('keychain/missing.json');
assert.equal(missing.startupSucceeded, false); assert(missing.vaultUnchanged);
assert.deepEqual(missing.vaultInventory, writer.vaultInventory);
assert.equal(json('keychain/cleanup.json').finalKeyStatus, -25300);
for (const identity of lifecycle.sourceIdentities) {
  const frozen = archive.files.find(entry => entry.path === `keychain/sources/${identity.path}`);
  assert(frozen, identity.path); assert.equal(frozen.sha256, identity.sha256);
}
assert.equal(archive.files.find(entry => entry.path === 'keychain/sources/scripts/test-keyboard-keychain.mjs').sha256, lifecycle.runnerSha256);
assert(read('keychain/keychain-harness-prepare-01.log').toString().includes('2 errors generated.'));
assert.equal(json('keychain/keychain-policy-assessment-01.json').deviceBoundProtectionVerification, 'not-established');
const metadata = json('metadata/assessment.json');
for (const source of metadata.appleSample.samples) assert.equal(archive.files.find(entry => entry.path === `metadata/${source.file}`).sha256, source.sha256);
assert.equal(archive.files.find(entry => entry.path === 'metadata/Meltype-35f2ce9-Info.plist').sha256, metadata.meltypeFailingRevision.sha256);
assert.equal(archive.system.installedIMEHostDispatch, false);
console.log(`PASS: ${archive.files.length} SHA-256 mappings; run08 failure/recovery preserved; notarized run09 selection failed; first-party TIS control restored; pre-fix real Keychain 6/6 lifecycle without device-bound assurance.`);
