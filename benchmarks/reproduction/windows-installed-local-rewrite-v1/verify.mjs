import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyDirectory } from '../windows-local-rewrite-v1/verify.mjs';

const directory = path.dirname(fileURLToPath(import.meta.url));
const manifest = verifyDirectory(directory);
const log = readFileSync(path.join(directory, 'run.log'), 'utf8');
assert.match(log, /--- PASS: TestNativeWindowsLifecycle \(60\.81s\)/);
assert.equal(manifest.execution.status, 'passed');
for (const [name, identity] of Object.entries(manifest.runtime.binaries)) {
  const canonical = manifest.runtime.canonicalComparison[name];
  assert.equal(canonical.matchesInstalled, true, `${name}: canonical match`);
  assert.equal(canonical.sha256, identity.sha256, `${name}: installed identity`);
  assert.equal(canonical.sizeBytes, identity.sizeBytes, `${name}: installed size`);
}
console.log('Verified installed-runtime/canonical identities and original passing test log; no host files or model inference were accessed.');
