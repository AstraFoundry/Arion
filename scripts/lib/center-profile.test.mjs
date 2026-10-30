import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { resolveCenter, standardCenter, centerFingerprint, prepareCenterBuild, centerBuildConfig, checkOfficialCenter } from './center-profile.mjs';

const device = { ARION_CENTER_URL: 'https://sync.example.org/prefix/', ARION_CENTER_AUTH_MODE: 'device' };
const oidc = { ...device, ARION_CENTER_AUTH_MODE: 'oidc', ARION_CENTER_OIDC_ISSUER: 'https://login.example.org', ARION_CENTER_OIDC_CLIENT_ID: 'custom-desktop', ARION_CENTER_OIDC_RESOURCE: 'https://sync.example.org', ARION_CENTER_OIDC_SCOPES: 'openid profile offline_access', ARION_CENTER_OIDC_ENTITLEMENT: 'custom.sync' };

test('center defaults are complete and custom device/OIDC modes remain separate', () => {
  assert.deepEqual(resolveCenter({}), standardCenter);
  assert.deepEqual(resolveCenter({ ARION_CENTER_URL: standardCenter.url }), standardCenter);
  const profile = resolveCenter(device);
  assert.equal(profile.url, 'https://sync.example.org/prefix');
  assert.equal(profile.issuer, '');
  assert.deepEqual(profile.scopes, []);
  const custom = resolveCenter(oidc);
  assert.equal(custom.issuer, 'https://login.example.org');
  assert.equal(custom.entitlement, 'custom.sync');
  assert.notEqual(centerFingerprint(profile), centerFingerprint(custom));
});

test('empty, partial, unknown and secret-bearing settings fail before compilation', () => {
  for (const env of [
    { ARION_CENTER_URL: '' }, { ARION_CENTER_URL: device.ARION_CENTER_URL },
    { ...device, ARION_CENTER_AUTH_MODE: 'unknown' },
    { ...device, ARION_CENTER_OIDC_ISSUER: '' },
    { ...oidc, ARION_CENTER_OIDC_CLIENT_ID: '' },
    { ...oidc, ARION_CENTER_OIDC_SCOPES: 'profile' },
    { ...device, ARION_CENTER_OIDC_CLIENT_SECRET: 'never-embed' },
  ]) assert.throws(() => resolveCenter(env));
  for (const url of ['http://public.example.org', 'https://user:password@example.org', 'https://example.org?secret=x', 'https://example.org#fragment', 'file:///tmp/server']) {
    assert.throws(() => resolveCenter({ ...device, ARION_CENTER_URL: url }));
  }
  for (const url of ['http://localhost:8080', 'http://127.0.0.1:8080/prefix', 'http://[::1]:8080']) assert.doesNotThrow(() => resolveCenter({ ...device, ARION_CENTER_URL: url }));
});

test('custom packages disable official updates and require the build injection path', () => {
  const config = { name: 'Arion', updates: { github: 'example/official' } };
  assert.equal(centerBuildConfig(config, {}), config);
  assert.throws(() => centerBuildConfig(config, device), /must use make/);
  const session = prepareCenterBuild({ ...device, APPLE_APP_SPECIFIC_PASSWORD: 'private-test-value' });
  try {
    assert.deepEqual(centerBuildConfig(config, { ...device, ...session.env }), { name: 'Arion' });
    const overlayPath = session.env.GOFLAGS.match(/-overlay=(.+)"$/)[1];
    const overlay = JSON.parse(readFileSync(overlayPath, 'utf8'));
    const source = readFileSync(Object.values(overlay.Replace)[0], 'utf8');
    assert.ok(!source.includes('private-test-value'));
    assert.ok(source.includes('sync.example.org'));
    session.cleanup();
    assert.ok(!existsSync(overlayPath));
  } finally { session.cleanup(); }
  assert.throws(() => prepareCenterBuild({ ...device, GOFLAGS: '-overlay=other.json' }), /conflicts/);
});

test('official release paths reject custom profiles before packaging', () => {
  assert.doesNotThrow(() => checkOfficialCenter({ ARION_OFFICIAL_RELEASE: '1' }));
  assert.throws(() => checkOfficialCenter({ ...device, ARION_OFFICIAL_RELEASE: '1' }), /exact standard/);
  assert.throws(() => prepareCenterBuild({ ...oidc, GITHUB_REF_TYPE: 'tag' }), /exact standard/);
});
