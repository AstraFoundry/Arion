import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import { releaseRebuildPlan } from './release-rebuild.mjs';

const previous = JSON.parse(readFileSync(new URL('../../packages/contracts/releases/staged-release-v1.json', import.meta.url), 'utf8'));
const config = { version: previous.version, release: { product: previous.product, repository: previous.source.repository } };
const channels = { stable: { channel: 'stable', targets: {} }, beta: { channel: 'beta', targets: { 'darwin-arm64': { version: '0.1.0-alpha.5' } } } };
const commit = previous.source.commit;

test('rebuilding archives every original file and document under its original source', () => {
  const plan = releaseRebuildPlan(config, previous, commit, channels);
  assert.equal(plan.recordKey, `releases/${previous.version}/replaced/${commit}/release.json`);
  assert.deepEqual(plan.copies.map(copy => copy.source), [
    'files/Arion.0.1.0-alpha.6.arm64.dmg',
    'files/arion-0.1.0-alpha.6-darwin-arm64.tar.gz',
    'releases/0.1.0-alpha.6/SHA256SUMS',
    'releases/0.1.0-alpha.6/RELEASE-STATUS.md',
  ]);
  for (const copy of plan.copies) assert.ok(copy.destination.startsWith(`releases/${previous.version}/replaced/${commit}/`));
});

test('a rebuild refuses a different original source or release identity', () => {
  for (const expected of [undefined, '', 'abc', '1'.repeat(40)]) {
    assert.throws(() => releaseRebuildPlan(config, previous, expected, channels), /expected replacement commit/);
  }
  for (const modified of [{ ...config, version: '0.1.0' }, { ...config, version: '0.1.0-alpha.7' }, { ...config, release: { ...config.release, repository: 'Other/Product' } }]) {
    assert.throws(() => releaseRebuildPlan(modified, previous, commit, channels), /matching prerelease/);
  }
});

test('a version offered by either update channel cannot be replaced', () => {
  for (const channel of ['stable', 'beta']) {
    const active = structuredClone(channels);
    active[channel].targets['windows-amd64'] = { version: previous.version };
    assert.throws(() => releaseRebuildPlan(config, previous, commit, active), /offered by an update channel/);
  }
});

test('missing or malformed channel evidence never authorizes replacement', () => {
  for (const beta of [undefined, {}, { channel: 'stable', targets: {} }, { channel: 'beta', targets: [] }, { channel: 'beta', targets: { 'darwin-arm64': {} } }]) {
    assert.throws(() => releaseRebuildPlan(config, previous, commit, { ...channels, beta }), /Invalid beta/);
  }
});

test('backup paths cannot escape the exact release prefix', () => {
  const invalid = structuredClone(previous);
  invalid.targets['darwin-arm64'].installer.name = `../${previous.version}/other`;
  assert.throws(() => releaseRebuildPlan(config, invalid, commit, channels), /backup paths/);
});
