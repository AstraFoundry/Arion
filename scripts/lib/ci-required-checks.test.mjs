import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { bloblessBaseFetch, ensureBaseCommit } from '../ci-native-scope.mjs';
import { evaluateRequiredChecks } from '../ci-required-checks.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

const pull = (patch = {}) => ({
  event: 'pull_request', portable: 'success', scope: 'success', native: 'success',
  nativeOutput: 'true', windowsUpdater: 'success', windowsRequired: true, ...patch,
});

test('a pull request cannot skip a required native or Windows check', () => {
  assert.deepEqual(evaluateRequiredChecks(pull()), []);
  assert.deepEqual(evaluateRequiredChecks(pull({ nativeOutput: 'false', native: 'skipped', windowsUpdater: 'skipped' })), []);
  assert.ok(evaluateRequiredChecks(pull({ native: 'skipped' })).some(line => /native shell was required/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ nativeOutput: '' })).some(line => /true or false/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ scope: 'failure', native: 'skipped', nativeOutput: '' })).some(line => /scope did not succeed/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ windowsUpdater: 'skipped' })).some(line => /Windows updater/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ portable: 'failure' })).some(line => /portable/.test(line)));
  assert.deepEqual(evaluateRequiredChecks(pull({ nativeOutput: 'false', native: 'success', windowsUpdater: 'skipped' })), []);
});

test('main and a macOS-only release keep their own required jobs', () => {
  assert.deepEqual(evaluateRequiredChecks({
    event: 'push', portable: 'success', scope: 'skipped', native: 'success',
    windowsUpdater: 'success', windowsRequired: true,
  }), []);
  assert.deepEqual(evaluateRequiredChecks({
    event: 'workflow_dispatch', portable: 'success', scope: 'skipped', native: 'success',
    windowsUpdater: 'skipped', windowsRequired: false,
  }), []);
  assert.ok(evaluateRequiredChecks({
    event: 'push', portable: 'success', scope: 'skipped', native: 'skipped',
    windowsUpdater: 'skipped', windowsRequired: true,
  }).some(line => /native shell/.test(line)));
  assert.ok(evaluateRequiredChecks({
    event: 'push', portable: 'success', scope: 'success', native: 'success',
    windowsUpdater: 'success', windowsRequired: true,
  }).some(line => /scope runs only/.test(line)));
});

test('base history is fetched blobless, and an existing commit is not fetched again', t => {
  const base = 'a'.repeat(40);
  assert.deepEqual(bloblessBaseFetch(base), ['fetch', '--no-tags', '--filter=blob:none', 'origin', base]);
  assert.equal(bloblessBaseFetch(base).includes('--depth=1'), false);
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-scope-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  execFileSync('git', ['init', '-q'], { cwd: directory });
  execFileSync('git', ['config', 'user.email', 'scope@example.test'], { cwd: directory });
  execFileSync('git', ['config', 'user.name', 'scope'], { cwd: directory });
  execFileSync('git', ['commit', '--allow-empty', '-q', '-m', 'base'], { cwd: directory });
  const sha = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: directory, encoding: 'utf8' }).trim();
  let fetched = false;
  ensureBaseCommit(sha, directory, () => { fetched = true; });
  assert.equal(fetched, false);
  assert.throws(() => ensureBaseCommit('b'.repeat(40), directory, () => { throw new Error('fetch failed'); }), /fetch failed/);
});

test('CI keeps blobless history, main-only native saves, and the required-check gate', () => {
  const ci = readFileSync(path.join(root, '.github/workflows/ci.yml'), 'utf8');
  const release = readFileSync(path.join(root, '.github/workflows/release.yml'), 'utf8');
  assert.equal([...ci.matchAll(/go-version: '([^']+)'/g)].map(match => match[1]).every(version => version === '1.27.3'), true);
  assert.equal([...ci.matchAll(/go-version: '([^']+)'/g)].length, 3);
  assert.equal([...ci.matchAll(/filter: blob:none/g)].length, 5);
  assert.equal([...ci.matchAll(/fetch-depth: 0/g)].length, 5);
  assert.doesNotMatch(ci, /fetch-depth: 1/);
  assert.match(ci, /actions\/cache\/restore@v4/);
  assert.match(ci, /actions\/cache\/save@v4/);
  assert.match(ci, /github\.ref == 'refs\/heads\/main'/);
  assert.match(ci, /ci-cache-report\.mjs/);
  assert.match(ci, /ci-required-checks\.mjs/);
  assert.match(ci, /windows-updater:/);
  assert.doesNotMatch(ci, /cache: false/);
  assert.match(release, /filter: blob:none/);
  assert.doesNotMatch(release, /fetch-depth: 1/);
  const scope = readFileSync(path.join(root, 'scripts/ci-native-scope.mjs'), 'utf8');
  assert.match(scope, /--no-renames/);
  assert.match(scope, /blob:none/);
  assert.doesNotMatch(scope, /--depth=1/);
});
