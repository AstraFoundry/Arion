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
const artifact = json(archive.source.artifact);
assert.equal(artifact.architecture, 'arm64');
assert.equal(artifact.parentID, 'si.astrafoundry.inputmethod.numbercontrol');
assert.equal(json('system/notarization.json').status, 'Accepted');
const gatekeeper = json('system/gatekeeper.json');
assert.equal(gatekeeper.status, 0); assert(gatekeeper.stderr.includes('Notarized Developer ID'));
const registration = json('system/registration.json');
assert(Object.values(registration.statuses).every(status => status === 0));
assert.equal(registration.sources.length, 1);
assert.equal(registration.sources[0].id, artifact.parentID);
assert.equal(registration.sources[0].enabled, false); assert.equal(registration.sources[0].selected, false);
const state = json('system/state.json');
assert.equal(state.phase, 'restored'); assert.deepEqual(state.errors, []);
assert.equal(state.hostPID, undefined); assert.equal(state.fixturePID, undefined);
assert(state.after.own.every(source => !source.enabled && !source.selected));
for (const file of ['system/final-source-query.json', 'system/final-arion-query.json']) {
  const final = json(file); assert.equal(final.current, 'com.apple.keylayout.ABC');
  assert.equal(final.learningKeyStatus, -25300); assert.deepEqual(final.own, []);
}
for (const member of artifact.provenance) {
  const frozen = archive.files.find(entry => entry.path === `sample/${member.member}`);
  if (frozen) assert.equal(frozen.sha256, member.sha256);
  else assert(archive.exclusions.some(entry => entry.sourceMember === member.member && entry.sha256 === member.sha256));
}
const original = read('sample/main.m').toString();
assert.equal(read('sample/main-derived.m').toString(), original.replaceAll('NumberInput_1_Connection', artifact.connection));
assert(original.includes('license') && original.includes('Disclaimer: IMPORTANT:'));
assert(read('sample/ReadMe').toString().includes('After this initial installation and log out'));
assert(read('logs/number-control-preparation.log').toString().includes('MACOS_IME_PROVISIONING_PROFILE is required'));
assert(read('logs/number-control-preparation-02.log').toString().includes('"notarization":"Accepted"'));
assert.equal(archive.result.hostLaunchAttempted, false);
assert.equal(archive.result.selectionAttempted, false);
assert.equal(archive.result.typingAttempted, false);
assert.equal(createHash('sha256').update(read(archive.source.readme)).digest('hex'), archive.source.readmeProvenance.sha256);
const controls = archive.finalControls;
assert(read(controls.tooling).toString().includes('ℹ pass 86'));
assert(read(controls.tooling).toString().includes('ℹ fail 0'));
assert(read(controls.unsignedBuild).toString().includes('Built apps/desktop/resources/darwin/ArionInput.app'));
assert(read(controls.unsignedVerify).toString().includes('(unsealed-development)'));
assert(read(controls.signedGate).toString().includes('MACOS_IME_PROVISIONING_PROFILE is required for signed desktop builds.'));
for (const binding of controls.sourceBindings) {
  assert(!path.isAbsolute(binding.frozenPath));
  if (binding.frozenPath.startsWith('../')) assert(/^\.\.\/imk-keyboard-v[1-5]\//.test(binding.frozenPath));
  const bytes = read(binding.frozenPath);
  assert.equal(bytes.length, binding.sizeBytes, binding.originalPath);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), binding.sha256, binding.originalPath);
}
console.log(`PASS: ${archive.files.length} SHA-256 mappings and ${controls.sourceBindings.length} final-source bindings; signed/notarized Apple control disabled/restored; unsigned strict build verified; tooling 86/86; missing-profile signed gate preserved.`);
