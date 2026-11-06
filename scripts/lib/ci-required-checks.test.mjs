import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { bloblessBaseFetch, ensureBaseCommit } from '../ci-native-scope.mjs';
import { evaluateRequiredChecks } from '../ci-required-checks.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

const pull = (patch = {}) => ({
  event: 'pull_request', portable: 'success', scope: 'success', native: 'success',
  nativeOutput: 'true', testsOutput: 'true', windowsUpdater: 'success', windowsRequired: true,
  siteOutput: 'true', siteResult: 'success', ...patch,
});

test('a pull request cannot skip a required native or Windows check', () => {
  assert.deepEqual(evaluateRequiredChecks(pull()), []);
  assert.deepEqual(evaluateRequiredChecks(pull({ nativeOutput: 'false', native: 'skipped', windowsUpdater: 'skipped' })), []);
  assert.ok(evaluateRequiredChecks(pull({ native: 'skipped' })).some(line => /native shell was required/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ nativeOutput: '' })).some(line => /true or false/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ scope: 'failure', native: 'skipped', nativeOutput: '' })).some(line => /scope did not succeed/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ windowsUpdater: 'skipped' })).some(line => /Windows updater/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ testsOutput: '' })).some(line => /tests output/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ portable: 'failure' })).some(line => /portable/.test(line)));
  assert.deepEqual(evaluateRequiredChecks(pull({ nativeOutput: 'false', native: 'success', windowsUpdater: 'skipped' })), []);
});

test('main and a macOS-only release keep their own required jobs', () => {
  assert.deepEqual(evaluateRequiredChecks({
    event: 'push', portable: 'success', scope: 'skipped', native: 'success',
    siteResult: 'success',
    windowsUpdater: 'success', windowsRequired: true,
  }), []);
  assert.deepEqual(evaluateRequiredChecks({
    event: 'workflow_dispatch', portable: 'success', scope: 'skipped', native: 'success',
    siteResult: 'success',
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

test('required site builds cannot silently skip and unrelated PRs can skip them', () => {
  assert.deepEqual(evaluateRequiredChecks(pull({ siteOutput: 'false', siteResult: 'skipped' })), []);
  assert.ok(evaluateRequiredChecks(pull({ siteResult: 'skipped' })).some(line => /site checks were required/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ siteOutput: '' })).some(line => /site output/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ event: 'push', scope: 'skipped', siteOutput: 'false', siteResult: 'skipped' }))
    .some(line => /site checks were required/.test(line)));
  assert.ok(evaluateRequiredChecks(pull({ event: 'workflow_dispatch', scope: 'skipped', siteResult: 'skipped' }))
    .some(line => /site checks were required/.test(line)));
});

test('the required-check CLI rejects a false site scope against the actual PR diff', t => {
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-required-site-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const git = (...args) => execFileSync('git', args, { cwd: directory, encoding: 'utf8' }).trim();
  git('init', '-q');
  git('config', 'user.name', 'scope');
  git('config', 'user.email', 'scope@example.test');
  mkdirSync(path.join(directory, 'scripts'));
  for (const script of ['ci-native-scope.mjs', 'ci-required-checks.mjs']) {
    copyFileSync(path.join(root, 'scripts', script), path.join(directory, 'scripts', script));
  }
  git('add', '.');
  git('commit', '-qm', 'base');
  const base = git('rev-parse', 'HEAD');
  mkdirSync(path.join(directory, 'apps/site'), { recursive: true });
  writeFileSync(path.join(directory, 'apps/site/page.tsx'), 'export default 1;\n');
  git('add', '.');
  git('commit', '-qm', 'site change');
  const env = { ...process.env, EVENT: 'pull_request', PORTABLE: 'success', SCOPE: 'success',
    NATIVE: 'skipped', NATIVE_OUTPUT: 'false', TESTS_OUTPUT: 'true', SITE_OUTPUT: 'false',
    SITE_RESULT: 'skipped', WINDOWS_UPDATER: 'skipped', WINDOWS_REQUIRED: 'true', BASE_SHA: base };
  const result = spawnSync(process.execPath, ['scripts/ci-required-checks.mjs'], { cwd: directory, env, encoding: 'utf8' });
  assert.equal(result.status, 1);
  assert.match(result.stderr, /scope skipped site checks for a site-related diff/);
  const output = execFileSync(process.execPath, ['scripts/ci-native-scope.mjs'], { cwd: directory, env: { ...env, GITHUB_OUTPUT: '' }, encoding: 'utf8' });
  assert.equal(output, 'native=false\ntests=true\nsite=true\n');
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
  assert.equal([...ci.matchAll(/filter: blob:none/g)].length, 6);
  assert.equal([...ci.matchAll(/fetch-depth: 0/g)].length, 6);
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
