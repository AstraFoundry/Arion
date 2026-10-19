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
const failed = json('preparation-failure/prepared.json');
assert.equal(failed.prepared, false); assert.equal(failed.executed, false);
assert(read('controls/keychain-preparation.log').toString().includes('codesign --display --extract-certificates '));
for (const kind of ['first-lifecycle', 'verified-lifecycle']) {
  const report = json(`${kind}/prepared.json`);
  assert(report.prepared && report.executed && report.passed && report.restored);
  assert.equal(report.results.length, 6);
  assert(report.results.every(result => result.passed && result.exitStatus === 0 && result.interactionPolicyRestored === 1 && !result.secretDataRequestedByHarness));
  assert.equal(report.backup.keyDataExported, false); assert.equal(report.backup.inputSourcesTouched, false);
  for (const identity of report.sourceIdentities) assert(archive.sourceBindings.some(binding => binding.originalPath === identity.path && binding.sha256 === identity.sha256), identity.path);
  assert(archive.sourceBindings.some(binding => binding.originalPath === 'scripts/test-keyboard-keychain.mjs' && binding.sha256 === report.runnerSha256));
  const writer = json(`${kind}/writer.json`), reader = json(`${kind}/reader.json`);
  assert.equal(writer.committed, '事实'); assert.equal(writer.beforeCandidates[0], '实施'); assert.equal(reader.afterRestartCandidates[0], '事实');
  assert.equal(writer.backingEvidence.legacy.status, -25300);
  assert.equal(writer.backingEvidence.dataProtection.status, 0); assert.equal(writer.backingEvidence.dataProtection.sameOwnedReference, 1);
  assert.equal(writer.keyAttributes.accessible, 'cku'); assert.equal(writer.keyAttributes.matchesAfterFirstUnlockThisDeviceOnly, true);
  assert.equal(writer.keyAttributes.accessiblePresent, 1); assert.equal(writer.keyAttributes.synchronizablePresent, 1);
  const missing = json(`${kind}/missing.json`);
  assert.equal(missing.startupSucceeded, false); assert(missing.vaultUnchanged); assert.deepEqual(missing.vaultInventory, writer.vaultInventory);
  const cleanup = json(`${kind}/cleanup.json`);
  assert.equal(cleanup.finalKeyStatus, -25300); assert.equal(cleanup.finalLegacyStatus, -25300); assert.equal(cleanup.finalDataProtectionStatus, -25300);
}
assert.equal(json('first-lifecycle/prepared.json').assurance.deviceBoundProtectionVerification, 'not-established');
assert.equal(json('first-lifecycle/writer.json').keyAttributes.synchronizable, 0);
assert.equal(json('verified-lifecycle/prepared.json').assurance.deviceBoundProtectionVerification, 'verified');
assert.equal(json('verified-lifecycle/writer.json').keyAttributes.synchronizable, false);
assert.equal(json('verified-lifecycle/writer.json').keyAttributes.synchronizableRaw, 0);
for (const binding of archive.sourceBindings) {
  assert(!path.isAbsolute(binding.frozenPath));
  if (binding.frozenPath.startsWith('../')) assert(/^\.\.\/imk-keyboard-v[1-7]\//.test(binding.frozenPath));
  const bytes = read(binding.frozenPath); assert.equal(bytes.length, binding.sizeBytes);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), binding.sha256, binding.originalPath);
}
const profile = json(archive.profile.metadata);
assert.equal(profile.authorization.bundleIdentifier, 'si.astrafoundry.inputmethod.arion');
assert.equal(profile.authorization.entitlements['com.apple.application-identifier'], 'DRLCWL5U4V.si.astrafoundry.inputmethod.arion');
assert.equal(archive.profile.assetIncluded, false);
assert(read(archive.controls.tooling).toString().includes('ℹ pass 87'));
assert(read(archive.controls.tooling).toString().includes('ℹ fail 0'));
assert(read(archive.controls.signedBuild).toString().includes('Verified Developer ID signatures, hardened runtime, notarization tickets and Gatekeeper acceptance.'));
assert(read(archive.controls.signedBuild).toString().includes('Verified darwin-arm64 resources, licenses and architectures'));
assert(![...paths].some(file => /\.(?:provisionprofile|mobileprovision|jpg|png|dylib|o|zip|dmg)$/.test(file)));
console.log(`PASS: ${archive.files.length} SHA-256 mappings and ${archive.sourceBindings.length} exact source bindings; failed preparation and original reporter limitation retained; real Data Protection lifecycle 6/6 verified/restored; tooling 87/87 and signed/notarized package verified; no profile/key assets.`);
