import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';
import assert from 'node:assert/strict';
import test from 'node:test';
import { openArchive, sha256 } from '../archive/manifest.mjs';

async function fixture(t) {
  const root = await fs.mkdtemp(path.join(os.tmpdir(), 'arion-archive-'));
  t.after(() => fs.rm(root, { recursive: true, force: true }));
  const bytes = Buffer.from('{"samples":[]}\n');
  const destination = 'benchmarks/corpora/example/corpus.json';
  await fs.mkdir(path.dirname(path.join(root, destination)), { recursive: true });
  await fs.writeFile(path.join(root, destination), bytes);
  const entry = { source: '.workspace/old-corpus.json', destination, bytes: bytes.length, sha256: sha256(bytes) };
  const manifest = { version: 1, kind: 'arion-benchmark-archive', historicalRoot: '/previous-machine/Arion', entries: [entry] };
  const filename = path.join(root, 'archive.json');
  const write = () => fs.writeFile(filename, JSON.stringify(manifest));
  await write();
  return { root, bytes, entry, manifest, filename, write };
}

test('archive reads original absolute and relative identities after relocation without workspace files', async t => {
  const f = await fixture(t), archive = await openArchive(f.filename, f.root);
  assert.deepEqual(await archive.read(f.entry.source), f.bytes);
  assert.deepEqual(await archive.read('/previous-machine/Arion/' + f.entry.source), f.bytes);
  assert.deepEqual(await archive.read(path.join(f.root, f.entry.destination)), f.bytes);
  assert.deepEqual(await archive.verify(), { status: 'passed', files: 1, bytes: f.bytes.length });
});

test('archive rejects changed bytes and does not substitute an original local file', async t => {
  const f = await fixture(t), archive = await openArchive(f.filename, f.root);
  await fs.mkdir(path.join(f.root, '.workspace'));
  await fs.writeFile(path.join(f.root, f.entry.source), f.bytes);
  await fs.writeFile(path.join(f.root, f.entry.destination), 'changed');
  await assert.rejects(archive.read(f.entry.source), /checksum changed/);
  await fs.rm(path.join(f.root, f.entry.destination));
  await assert.rejects(archive.read(f.entry.source), /ENOENT/);
  await assert.rejects(archive.read('.workspace/unmapped.json'), /Unmapped archive path/);
});

test('archive rejects duplicate identities and destinations outside benchmarks', async t => {
  const f = await fixture(t);
  f.manifest.entries.push({ ...f.entry }); await f.write();
  await assert.rejects(openArchive(f.filename, f.root), /duplicate archive identity/);
  f.manifest.entries.pop(); f.entry.destination = '../outside.json'; await f.write();
  await assert.rejects(openArchive(f.filename, f.root), /Invalid/);
  f.entry.destination = '/absolute.json'; await f.write();
  await assert.rejects(openArchive(f.filename, f.root), /Invalid/);
});

test('archive rejects symlinks rather than reading mutable external evidence', { skip: process.platform === 'win32' }, async t => {
  const f = await fixture(t), archive = await openArchive(f.filename, f.root);
  const external = path.join(f.root, 'external.json'); await fs.writeFile(external, f.bytes);
  await fs.rm(path.join(f.root, f.entry.destination));
  await fs.symlink(external, path.join(f.root, f.entry.destination));
  await assert.rejects(archive.read(f.entry.source), /escapes root/);
});

test('phonetic archive option cannot be used to start inference', () => {
  const cli = fileURLToPath(new URL('../phonetic/cli.mjs', import.meta.url));
  const result = spawnSync(process.execPath, [cli, 'run', '--archive', 'missing.json'], { encoding: 'utf8' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /only supported for report replay/);
});
