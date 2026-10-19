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
const registration = json('system/registration.json');
assert(Object.values(registration.statuses).every(status => status === 0));
assert.equal(registration.sources.find(source => source.id === 'si.astrafoundry.inputmethod.arion').enabled, false);
assert.equal(registration.sources.find(source => source.id.endsWith('.pinyin')).enabled, true);
assert(registration.sources.every(source => !source.selected));
const disabled = json('system/disabled.json');
assert(Object.values(disabled.statuses).every(status => status === 0));
assert(disabled.sources.every(source => !source.selected));
const state = json('system/state.json');
assert.equal(state.phase, 'restored'); assert.deepEqual(state.errors, []);
assert.equal(state.baseline.current, 'com.tencent.inputmethod.chatterfly.pinyin');
assert.equal(state.after.current, state.baseline.current);
assert.equal(state.after.learningKeyStatus, -25300);
assert(state.after.own.every(source => !source.enabled && !source.selected));
assert.equal(state.hostPID, undefined); assert.equal(state.fixturePID, undefined);
const final = json('system/final-source-query.json');
assert.equal(final.current, state.baseline.current); assert.deepEqual(final.own, []);
assert.equal(final.learningKeyStatus, -25300);
const host = json('system/final-host-query.json'); assert.equal(host.exitStatus, 1); assert.equal(host.stdout, '');
const artifact = json('system/artifact.json');
assert.equal(artifact.sourceExecutableSHA256, artifact.stagedExecutableSHA256);
const profile = JSON.parse(readFileSync(path.join(root, '../imk-keyboard-v8/controls/profile-validation.json')));
assert.equal(artifact.embeddedProfileSHA256, profile.profileSHA256);
assert.equal(json('system/observations.json').kind, 'derived-from-retained-original-control-outputs');
assert.equal(archive.result.acceptance, 'deferred; not complete');
assert.equal(archive.result.selectionAttempted, false); assert.equal(archive.result.typingAttempted, false); assert.equal(archive.result.hostLaunchAttempted, false);
console.log(`PASS: ${archive.files.length} SHA-256 mappings; exact profile-bearing artifact hashes match; parent disabled after enable0; restored Chatterfly/key absence/later own[]/no host; system acceptance deferred.`);
