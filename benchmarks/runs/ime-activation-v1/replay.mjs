// Replay retained observations only. Never register, enable or select a source.
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root = path.dirname(fileURLToPath(import.meta.url));
const read = name => JSON.parse(readFileSync(path.join(root, name), 'utf8'));
for (const item of read('manifest.json').files) {
  const bytes = readFileSync(path.join(root, item.path));
  assert.equal(bytes.length, item.bytes, item.path);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), item.sha256, item.path);
}
const native = name => read(`native/${name}.json`);
for (const name of ['existing-activation', 'already-selected', 'reinstall-activate', 'isolated-host-activation-ready']) assert.equal(native(name).state, 3, name);
assert.equal(native('signed-upgrade').state, 2);
assert.equal(native('reinstall-remove').state, 0);
assert.equal(native('reinstall-install').state, 1);
assert.equal(native('cancelled').state, 2);
assert.match(native('cancelled').detail, /canceled/);
for (const key of ['idempotentActivationPassed', 'cancelLeftSelectionUnchanged', 'installedBytesUnchanged']) assert.equal(native('retry-verification')[key], true, key);
assert.match(native('isolated-host-activation').detail, /deadline exceeded/);
assert.equal(native('editor-composing').A.marked, true);
for (const name of ['editor-committed', 'editor-cancelled']) {
  const editor = native(name).A;
  assert.equal(editor.text, '你好');
  assert.equal(editor.edits, 1);
  assert.equal(editor.marked, false);
}
const keyboard = read('checks/keyboard-input.json');
assert.equal(keyboard.passed, true);
assert.equal(keyboard.cases.length, 34);
assert.ok(keyboard.cases.every(item => item.passed));
assert.equal(read('result.json').sourceRestored, native('before').selectedSource);
console.log('Verified archived hashes and known-authorization activation/IMK observations; clean first consent remains unverified.');
