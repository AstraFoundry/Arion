// Model-free, read-only verification of original evidence and frozen sources.
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';

const root = path.dirname(fileURLToPath(import.meta.url));
const archive = JSON.parse(readFileSync(path.join(root, 'archive.json'), 'utf8'));
const declared = new Set();
for (const file of archive.files) {
  assert(!path.isAbsolute(file.path) && !file.path.split('/').includes('..'), 'archive path escapes root');
  assert(!declared.has(file.path), `duplicate mapping: ${file.path}`);
  declared.add(file.path);
  const bytes = readFileSync(path.join(root, file.path));
  assert.equal(bytes.length, file.sizeBytes, `size mismatch: ${file.path}`);
  assert.equal(createHash('sha256').update(bytes).digest('hex'), file.sha256, `digest mismatch: ${file.path}`);
}
function inventory(directory) {
  return readdirSync(directory).flatMap(name => {
    const file = path.join(directory, name);
    return statSync(file).isDirectory() ? inventory(file) : [path.relative(root, file).split(path.sep).join('/')];
  });
}
assert.deepEqual(inventory(root).filter(file => file !== 'archive.json').sort(), [...declared].sort(), 'unmapped archive file');
for (const attempt of archive.attempts) {
  if (!attempt.report) { assert.equal(attempt.status, 'compile-failed'); continue; }
  const report = JSON.parse(readFileSync(path.join(root, attempt.report), 'utf8'));
  assert.equal(report.cases.length, attempt.total);
  assert.equal(report.cases.filter(test => test.passed).length, attempt.passed);
  assert.equal(report.passed, attempt.status === 'passed');
}
const final = archive.attempts.find(attempt => attempt.id === 'typing-attempt-11');
assert.equal(final.passed, 20); assert.equal(final.total, 20); assert(final.exactExecutedHarnessArchived);
const recorded = JSON.parse(readFileSync(path.join(root, 'engine/typing-attempt-11-identities.json'), 'utf8'));
for (const identity of recorded) {
  const match = archive.files.find(file => file.originalPath === identity.path && file.sourceRelation === 'matches completed attempt-11 recorded identity');
  if (identity.path.endsWith('.dylib')) {
    assert(archive.exclusions.some(file => file.path === identity.path && file.sha256 === identity.sha256));
  } else {
    assert(match, `missing exact executed source: ${identity.path}`);
    assert.equal(match.sha256, identity.sha256);
  }
}
const ownIDs = new Set(['si.astrafoundry.inputmethod.arion', 'si.astrafoundry.inputmethod.arion.pinyin']);
for (const run of archive.systemRuns) {
  assert(run.status.startsWith('failed'), 'failed OS acceptance cannot become passing');
  const state = JSON.parse(readFileSync(path.join(root, run.state), 'utf8'));
  assert.equal(state.phase, 'restored'); assert.deepEqual(state.errors, []);
  for (const snapshot of [state.baseline, state.after]) {
    assert.equal(snapshot.current, 'com.apple.keylayout.ABC');
    for (const own of snapshot.own) assert(ownIDs.has(own.id), 'non-allowlisted source retained');
  }
}
console.log(`PASS: ${archive.files.length} SHA-256 mappings, 11 historical attempts, 20/20 final functional cases, 2 failed/restored OS attempts; no runtime or workspace required.`);
