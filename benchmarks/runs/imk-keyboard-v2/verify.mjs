// Read-only supplement verification; never starts a runtime or changes input sources.
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
const root = path.dirname(fileURLToPath(import.meta.url));
const read = file => readFileSync(path.join(root, file), 'utf8');
const json = file => JSON.parse(read(file));
const manifest = json('archive.json');
const declared = new Set();
for (const file of manifest.files) {
  assert(!path.isAbsolute(file.path) && !file.path.split('/').includes('..'));
  assert(!declared.has(file.path)); declared.add(file.path);
  const bytes = readFileSync(path.join(root, file.path));
  assert.equal(bytes.length, file.sizeBytes, file.path);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), file.sha256, file.path);
}
function inventory(directory) {
  return readdirSync(directory).flatMap(name => {
    const target = path.join(directory, name);
    return statSync(target).isDirectory() ? inventory(target) : [path.relative(root, target).split(path.sep).join('/')];
  });
}
assert.deepEqual(inventory(root).filter(file => file !== 'archive.json').sort(), [...declared].sort());
const report = json(manifest.functional.report);
assert.equal(report.passed, true); assert.equal(report.cases.length, 22);
assert(report.cases.every(test => test.passed));
for (const identity of json(manifest.functional.identities)) {
  if (identity.path.endsWith('.dylib')) {
    assert(manifest.exclusions.some(file => file.path === identity.path && file.sha256 === identity.sha256));
    continue;
  }
  const copy = manifest.files.find(file => file.originalPath === identity.path && file.kind === 'identity-matching-attempt-12-source-or-manifest');
  assert(copy, identity.path); assert.equal(copy.sha256, identity.sha256);
}
const own = new Set(['si.astrafoundry.inputmethod.arion', 'si.astrafoundry.inputmethod.arion.pinyin']);
const state = json(manifest.system.restoration);
assert.equal(state.phase, 'restored'); assert.deepEqual(state.errors, []);
for (const snapshot of [state.baseline, state.after, json(manifest.system.beforeRestoration)]) {
  assert.equal(snapshot.current, 'com.apple.keylayout.ABC');
  assert.equal(snapshot.learningKeyStatus, -25300);
  for (const source of snapshot.own) assert(own.has(source.id));
}
assert(state.after.own.every(source => !source.enabled && !source.selected));
assert.equal(read(manifest.system.rawSelection).trim(), '-50');
for (const file of [manifest.system.rawRegistration, manifest.system.rawTransition]) {
  assert(read(file).includes('after si.astrafoundry.inputmethod.arion 0'));
  assert(read(file).includes('after si.astrafoundry.inputmethod.arion.pinyin 1'));
}
const observations = json(manifest.system.observations);
assert.equal(observations.status, 'failed-restored');
assert(observations.productionInstall.evidenceType.includes('transcribed'));
assert(observations.gatekeeper.evidenceType.includes('transcribed'));
assert.equal(observations.productionInstall.errorCode, 'TARGET_UNSUPPORTED');
assert.equal(observations.gatekeeper.status, 0);
const log = read(manifest.system.signedBuildLog);
assert(log.includes('✓ Notarized Arion 0.1.0 arm64.dmg'));
assert(log.includes('Verified Developer ID signatures, hardened runtime, notarization tickets and Gatekeeper acceptance.'));
assert.equal(manifest.system.status, 'failed-restored');
console.log(`PASS: ${manifest.files.length} SHA-256 mappings, exact attempt-12 sources, 22/22 functional cases, completed signed build, and failed -50/restored source run-05; no runtime or OS mutation.`);
