import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function verifyDirectory(directory) {
const manifest = JSON.parse(readFileSync(path.join(directory, 'manifest.json')));
for (const [name, identity] of Object.entries(manifest.archivedFiles)) {
  const data = readFileSync(path.join(directory, name));
  assert.equal(data.length, identity.sizeBytes, `${name}: size`);
  assert.equal(createHash('sha256').update(data).digest('hex'), identity.sha256, `${name}: SHA-256`);
}
const observations = readFileSync(path.join(directory, 'observations.jsonl'), 'utf8').trim().split('\n').map(line => JSON.parse(line));
const rows = observations.filter(row => row.kind !== 'load');
assert.equal(rows.length, 7);
for (const row of rows) {
  if (row.kind === 'cancelled') {
    assert.match(row.error, /^CANCELLED:/);
    assert(row.elapsedMs < 1000, 'cancellation acknowledgement');
  } else {
    assert.equal(row.error, '', `${row.kind}: inference error`);
    assert(row.response.Text.length > 0, `${row.kind}: missing generated text`);
    assert(row.elapsedMs < 30000, `${row.kind}: CPU product deadline`);
    assert.equal(row.response.Truncated, false);
  }
}
assert.equal(rows.find(row => row.kind === 'after-cancellation').response.ReusedTokens, 0, 'cancelled native KV must not survive');
const model = JSON.parse(readFileSync(path.join(directory, '../../../models/catalog.json'))).rewriters.find(row => row.id === manifest.model.id);
for (const [name, identity] of Object.entries(manifest.model.files)) {
  assert.equal(identity.sha256, model.files[name].sha256, 'catalog model identity');
  assert.equal(identity.sizeBytes, model.files[name].sizeBytes, 'catalog model size');
}
console.log(`Verified ${Object.keys(manifest.archivedFiles).length} archived files, six actual completions and bounded cancellation; no inference was rerun.`);
return manifest;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  verifyDirectory(path.dirname(fileURLToPath(import.meta.url)));
}
