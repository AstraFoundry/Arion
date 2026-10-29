import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, writeFileSync, rmSync, existsSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { withEvidence } from './evidence.mjs';
const archive = fileURLToPath(new URL('../../runs/llm-ime-alignment-v2/', import.meta.url));

test('historical bundle expands original bytes and is cleaned after failed replay', () => {
  let expanded;
  assert.throws(() => withEvidence([archive], ([directory]) => {
    expanded = directory;
    assert.equal(JSON.parse(readFileSync(path.join(directory, 'run/status.json'))).status, 'prepared');
    throw new Error('replay rejected');
  }), /replay rejected/);
  assert.equal(existsSync(expanded), false);
});

test('changed evidence is refused before replay', () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-rank-tamper-'));
  try {
    writeFileSync(path.join(directory, 'bundle.json'), readFileSync(path.join(archive, 'bundle.json')));
    writeFileSync(path.join(directory, 'evidence.tar.gz'), 'changed');
    assert.throws(() => withEvidence([directory], () => assert.fail('tampered evidence replayed')), /bundle changed/);
  } finally { rmSync(directory, { recursive: true, force: true }); }
});
