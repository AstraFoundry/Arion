import assert from 'node:assert/strict';
import { mkdtempSync, readdirSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { checkWorkflows } from './workflow-check.mjs';

function fixture(t) {
  const root = mkdtempSync(path.join(os.tmpdir(), 'arion-workflows-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const calls = [];
  let installStatus = 0;
  let lintStatus = 0;
  let version = 'v1.7.11';
  const run = (command, args, options) => {
    calls.push({ command, args, options });
    if (command === 'go' && args[0] === 'install') {
      if (!installStatus) writeFileSync(path.join(options.env.GOBIN, 'actionlint'), 'fixture');
      return { status: installStatus };
    }
    if (command === 'go' && args[0] === 'version') {
      return { status: 0, stdout: `\tmod\tgithub.com/rhysd/actionlint\t${version}\th1:fixture\n` };
    }
    return { status: lintStatus };
  };
  return { root, calls, run, setInstall: value => { installStatus = value; }, setLint: value => { lintStatus = value; }, setVersion: value => { version = value; } };
}

test('provision once, reuse the pinned cache without a module proxy, and propagate lint failures', t => {
  const f = fixture(t);
  const options = { run: f.run, env: { GOPROXY: 'https://proxy.example.test' }, platform: 'linux', arch: 'arm64' };
  assert.equal(checkWorkflows(f.root, options), 0);
  assert.equal(f.calls[0].options.env.GOPROXY, options.env.GOPROXY);
  assert.equal(f.calls[0].options.env.GOWORK, 'off');
  f.calls.length = 0;
  f.setLint(1);
  assert.equal(checkWorkflows(f.root, { ...options, env: { GOPROXY: 'off' } }), 1);
  assert.deepEqual(f.calls.map(call => call.args), [['version', '-m', f.calls[1].command], ['-shellcheck=', '-pyflakes=']]);
});

test('failed provisioning never runs lint or leaves a usable partial cache', t => {
  const f = fixture(t);
  f.setInstall(2);
  const options = { run: f.run, platform: 'linux', arch: 'arm64' };
  assert.equal(checkWorkflows(f.root, options), 2);
  assert.equal(f.calls.length, 1);
  const directory = path.dirname(f.calls[0].options.env.GOBIN);
  assert.deepEqual(readdirSync(directory), []);
  f.setInstall(0);
  assert.equal(checkWorkflows(f.root, options), 0);
});

test('reject a cached tool with a different module version before lint runs', t => {
  const f = fixture(t);
  const options = { run: f.run, platform: 'linux', arch: 'arm64' };
  assert.equal(checkWorkflows(f.root, options), 0);
  f.calls.length = 0;
  f.setVersion('v1.7.10');
  assert.equal(checkWorkflows(f.root, options), 1);
  assert.equal(f.calls.length, 1);
});
