import test from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { clearProfile, resetOptions, resetPlan } from './reset-desktop.mjs';

const repository = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
function fixture(t) {
  const root = realpathSync(mkdtempSync(path.join(tmpdir(), 'arion-reset-test-')));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const directory = path.join(root, 'Arion');
  mkdirSync(directory);
  const guards = { home: path.join(root, 'home'), repository, configBase: root };
  for (const name of ['models/asr/weights', 'models/rewrite/partial.download', 'keyboard/learning/vault', 'preferences.json', 'keyboard.json', 'vocabulary.bin', 'vocabulary-key.dpapi', 'usage.json', 'preferences.json.old.bak', 'unexpected/future-data']) {
    const file = path.join(directory, name);
    mkdirSync(path.dirname(file), { recursive: true });
    writeFileSync(file, `original:${name}`);
  }
  return { root, directory, guards };
}

test('full reset removes every profile entry, including backups and unknown future files', t => {
  const f = fixture(t);
  const plan = resetPlan(f.directory, resetOptions([]), f.guards);
  assert.ok(Object.values(plan.system).every(Boolean));
  clearProfile(plan);
  assert.deepEqual(resetPlan(f.directory, resetOptions([]), f.guards).remove, []);
});

test('combined exclusions preserve model bytes and preference files while clearing other data', t => {
  const f = fixture(t);
  const options = resetOptions(['keep-models', 'keep-preferences']);
  clearProfile(resetPlan(f.directory, options, f.guards));
  for (const name of ['models/asr/weights', 'models/rewrite/partial.download', 'preferences.json', 'keyboard.json']) {
    assert.equal(readFileSync(path.join(f.directory, name), 'utf8'), `original:${name}`);
  }
  assert.equal(existsSync(path.join(f.directory, 'vocabulary.bin')), false);
});

test('content exclusions retain their keys and keep-system preserves OS scopes and DPAPI blob', t => {
  const f = fixture(t);
  const options = resetOptions(['keep-vocabulary', 'keep-keyboard', 'keep-usage', 'keep-system']);
  assert.ok(Object.values(options.system).every(value => value === false));
  const plan = resetPlan(f.directory, options, f.guards);
  assert.deepEqual(plan.keep, ['keyboard', 'usage.json', 'vocabulary-key.dpapi', 'vocabulary.bin']);
  clearProfile(plan);
  assert.ok(existsSync(path.join(f.directory, 'keyboard/learning/vault')));
});

test('unsafe roots and symlinked profiles are rejected; child links never erase their targets', t => {
  const f = fixture(t);
  for (const directory of ['relative', path.parse(f.root).root, f.root, repository]) {
    assert.throws(() => resetPlan(directory, resetOptions([]), f.guards), /absolute|unsafe/);
  }
  if (process.platform === 'win32') return; // Creating symlinks requires extra host privileges.
  const outside = path.join(f.root, 'outside');
  mkdirSync(outside);
  writeFileSync(path.join(outside, 'secret'), 'untouched');
  symlinkSync(outside, path.join(f.directory, 'linked-data'));
  const alias = path.join(f.root, 'alias');
  symlinkSync(f.directory, alias);
  assert.throws(() => resetPlan(alias, resetOptions([]), f.guards), /symlinked/);
  clearProfile(resetPlan(f.directory, resetOptions([]), f.guards));
  assert.equal(readFileSync(path.join(outside, 'secret'), 'utf8'), 'untouched');
});

test('unknown suffixes fail and dry-run leaves the fixture byte-for-byte intact', t => {
  const f = fixture(t);
  assert.throws(() => resetOptions(['keep-modles']), /Unknown/);
  if (!['darwin', 'win32'].includes(process.platform)) return;
  const out = execFileSync(process.execPath, ['scripts/reset-desktop.mjs', 'keep-models', '--dry-run'], { cwd: repository, env: { ...process.env, ARION_DATA_DIR: f.directory }, encoding: 'utf8' });
  assert.match(out, /Preview only/);
  assert.match(out, /"models"/);
  assert.equal(readFileSync(path.join(f.directory, 'vocabulary.bin'), 'utf8'), 'original:vocabulary.bin');
});

test('Make validates all reset goals before executing any mutation', t => {
  const f = fixture(t);
  for (const args of [['reset', 'keep-modles'], ['reset', 'desktop'], ['reset', 'desktop', 'onboarding', 'keep-models'], ['reset', 'install'], ['reset', 'DRY_RUN=wrong']]) {
    const result = spawnSync('make', args, { cwd: repository, env: { ...process.env, ARION_DATA_DIR: f.directory }, encoding: 'utf8' });
    assert.notEqual(result.status, 0, result.stdout);
    assert.equal(readFileSync(path.join(f.directory, 'vocabulary.bin'), 'utf8'), 'original:vocabulary.bin');
  }
  for (const args of [['-n', 'reset'], ['-n', 'reset', 'keep-models'], ['-n', 'reset', 'desktop', 'onboarding']]) {
    const result = spawnSync('make', args, { cwd: repository, encoding: 'utf8' });
    assert.equal(result.status, 0, result.stderr);
    assert.match(result.stdout, /node scripts\/reset-desktop/);
  }
});
