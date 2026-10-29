// Expand hash-pinned historical evidence only for the duration of replay.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { existsSync, mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { hash, read } from '../imk-quality-v1/evaluate.mjs';

export function withEvidence(directories, replay) {
  const temporary = [];
  try {
    const expanded = directories.map(directory => {
      const root = path.resolve(directory);
      if (!existsSync(path.join(root, 'bundle.json'))) return root;
      const bundle = read(path.join(root, 'bundle.json'));
      assert.equal(bundle.file, 'evidence.tar.gz');
      const archive = path.join(root, bundle.file);
      assert.equal(hash(archive), bundle.sha256, 'Historical evidence bundle changed');
      const members = execFileSync('tar', ['-tzf', archive], { encoding: 'utf8' }).trim().split('\n');
      assert.equal(members.length, bundle.files);
      assert.equal(new Set(members).size, members.length);
      for (const member of members) {
        assert.ok(!path.isAbsolute(member) && !member.split('/').some(part => ['', '.', '..'].includes(part)));
      }
      const destination = mkdtempSync(path.join(os.tmpdir(), 'arion-rank-evidence-'));
      temporary.push(destination);
      execFileSync('tar', ['-xzf', archive, '-C', destination]);
      return destination;
    });
    return replay(expanded);
  } finally {
    for (const directory of temporary) rmSync(directory, { recursive: true, force: true });
  }
}
