import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { digest, hashFile, readJson, sha256, verifyDriverArtifacts } from '../src/artifacts.mjs';

test('artifact checks read actual bytes and reject missing, changed and duplicate identities', async t => {
  const directory = await mkdtemp(path.join(tmpdir(), 'arion-artifact-contract-'));
  t.after(() => rm(directory, { recursive: true, force: true }));
  const file = path.join(directory, 'runtime.json');
  const bytes = '{"fixture":true}\n';
  await writeFile(file, bytes);
  const hash = sha256(bytes);
  assert.equal(await hashFile(file), hash);
  assert.deepEqual(await readJson(file), { fixture: true });
  assert.equal(digest({ fixture: true }), sha256(JSON.stringify({ fixture: true })));
  const artifact = { id: 'runtime', path: 'runtime.json', sha256: hash, license: 'synthetic-test' };
  assert.deepEqual(await verifyDriverArtifacts({ artifacts: [artifact] }, directory), { runtime: hash });
  await assert.rejects(verifyDriverArtifacts({ artifacts: [artifact, artifact] }, directory), /identity/);
  await assert.rejects(verifyDriverArtifacts({ artifacts: [{ ...artifact, license: '' }] }, directory), /identity/);
  await writeFile(file, 'changed');
  await assert.rejects(verifyDriverArtifacts({ artifacts: [artifact] }, directory), /checksum/);
  await assert.rejects(verifyDriverArtifacts({ artifacts: [{ ...artifact, path: 'absent' }] }, directory), /ENOENT/);
  await assert.rejects(verifyDriverArtifacts({}, directory), /pin/);
});
