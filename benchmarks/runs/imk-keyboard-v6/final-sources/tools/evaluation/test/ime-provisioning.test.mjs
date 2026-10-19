import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { inspectIMEProvisioning, stageIMEProvisioning, validateIMEProfile } from '../../../scripts/lib/ime-provisioning.mjs';

const options = { identity: 'Developer ID Application: Test (ABCDE12345)', bundleIdentifier: 'si.astrafoundry.inputmethod.arion', certificateHashes: ['a'.repeat(64)], now: new Date('2025-01-01T00:00:00Z') };
const valid = () => ({ UUID: 'synthetic-profile-fixture', TeamIdentifier: ['ABCDE12345'], Platform: ['OSX'], ProvisionsAllDevices: true,
  ExpirationDate: '2030-01-01T00:00:00Z', certificateHashes: ['a'.repeat(64)],
  Entitlements: { 'com.apple.application-identifier': 'ABCDE12345.si.astrafoundry.inputmethod.arion', 'com.apple.developer.team-identifier': 'ABCDE12345' } });

test('single host requests only profile-authorized application and team identity', () => {
  assert.deepEqual(validateIMEProfile(valid(), options).entitlements, valid().Entitlements);
  const wildcard = valid(); wildcard.Entitlements['com.apple.application-identifier'] = 'ABCDE12345.si.astrafoundry.*';
  assert.equal(validateIMEProfile(wildcard, options).entitlements['com.apple.application-identifier'], valid().Entitlements['com.apple.application-identifier']);
});

test('missing profile fails before CMS or certificate commands', () => {
  let calls = 0;
  assert.throws(() => inspectIMEProvisioning({ profilePath: '/does-not-exist/ArionInput.provisionprofile', run: () => { calls++; } }), /existing absolute-path/);
  assert.equal(calls, 0);
});

test('expired, wrong-platform, development, team, app and certificate profiles are rejected', () => {
  const mutations = [
    p => { p.ExpirationDate = '2024-01-01T00:00:00Z'; },
    p => { p.ExpirationDate = 'invalid'; },
    p => { p.Platform = ['iOS']; },
    p => { p.ProvisionsAllDevices = false; },
    p => { p.ProvisionedDevices = ['synthetic-device']; },
    p => { p.Entitlements['get-task-allow'] = true; },
    p => { p.TeamIdentifier = ['OTHER12345']; },
    p => { p.Entitlements['com.apple.developer.team-identifier'] = 'OTHER12345'; },
    p => { p.Entitlements['com.apple.application-identifier'] = 'ABCDE12345.si.other.app'; },
    p => { p.Entitlements['com.apple.application-identifier'] = 'ABCDE12345.si.astrafoundry.inputmethod.ario*'; },
    p => { p.certificateHashes = ['b'.repeat(64)]; },
  ];
  for (const mutate of mutations) { const profile = valid(); mutate(profile); assert.throws(() => validateIMEProfile(profile, options)); }
});


test('failed profile preparation never emits restricted entitlements or embeds a profile', () => {
  const directory = mkdtempSync(path.join(os.tmpdir(), 'ime-profile-gate-'));
  const entitlementsFile = path.join(directory, 'host.entitlements.plist');
  try {
    assert.throws(() => stageIMEProvisioning({ profilePath: path.join(directory, 'missing.provisionprofile'), identity: options.identity,
      bundleIdentifier: options.bundleIdentifier, bundle: directory, entitlementsFile }), /existing absolute-path/);
    assert.equal(existsSync(entitlementsFile), false);
    assert.equal(existsSync(path.join(directory, 'Contents/embedded.provisionprofile')), false);
  } finally { rmSync(directory, { recursive: true, force: true }); }
});
