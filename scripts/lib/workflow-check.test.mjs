import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtempSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { checkWorkflows } from './workflow-check.mjs';

function fixture(t) {
  const root = mkdtempSync(path.join(os.tmpdir(), 'arion-workflows-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const calls = [];
  const digest = text => createHash('sha256').update(text).digest('hex');
  let downloadStatus = 0;
  let lintStatus = 0;
  let archive = 'archive';
  const run = (command, args, options) => {
    calls.push({ command, args, options });
    if (command === 'curl') {
      if (!downloadStatus) writeFileSync(args[args.indexOf('--output') + 1], archive);
      return { status: downloadStatus };
    }
    if (command === 'tar') {
      writeFileSync(path.join(args[args.indexOf('-C') + 1], 'actionlint'), 'binary');
      return { status: 0 };
    }
    return { status: lintStatus };
  };
  const options = { run, platform: 'linux', arch: 'arm64', artifacts: { 'linux-arm64': { archive: digest('archive'), binary: digest('binary') } } };
  return { root, calls, options, setDownload: value => { downloadStatus = value; }, setLint: value => { lintStatus = value; }, corruptArchive: () => { archive = 'corrupt'; } };
}

test('download once, reuse the verified executable without Go, and propagate lint failures', t => {
  const f = fixture(t);
  assert.equal(checkWorkflows(f.root, f.options), 0);
  assert.deepEqual(f.calls.map(call => path.basename(call.command)), ['curl', 'tar', 'actionlint']);
  f.calls.length = 0;
  f.setLint(1);
  assert.equal(checkWorkflows(f.root, f.options), 1);
  assert.equal(f.calls.length, 1);
  assert.deepEqual(f.calls[0].args, ['-shellcheck=', '-pyflakes=']);
});

test('failed downloads never run lint or leave a usable partial cache', t => {
  const f = fixture(t);
  f.setDownload(2);
  assert.equal(checkWorkflows(f.root, f.options), 2);
  assert.equal(f.calls.length, 1);
  const directory = path.dirname(path.dirname(f.calls[0].args[5]));
  assert.deepEqual(readdirSync(directory), []);
  f.setDownload(0);
  assert.equal(checkWorkflows(f.root, f.options), 0);
});

test('reject corrupted archives before extraction and corrupted cached binaries before execution', t => {
  const f = fixture(t);
  f.corruptArchive();
  assert.throws(() => checkWorkflows(f.root, f.options), /archive SHA-256 mismatch/);
  assert.equal(f.calls.length, 1);
  const good = fixture(t);
  assert.equal(checkWorkflows(good.root, good.options), 0);
  writeFileSync(good.calls.at(-1).command, 'corrupt');
  good.calls.length = 0;
  assert.throws(() => checkWorkflows(good.root, good.options), /Invalid actionlint cache/);
  assert.equal(good.calls.length, 0);
});
