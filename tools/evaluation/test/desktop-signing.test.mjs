import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
import { macosSigningSettings, prepareMacOSSigning, macosBuildConfig, signingRunner, stripAppleCredentials } from '../../../scripts/lib/desktop-signing.mjs';

const identity = 'Developer ID Application: Arion Test (ABCDE12345)';
const provisioning = '/private/test/ArionInput.provisionprofile';
const local = { MACOS_IME_PROVISIONING_PROFILE: provisioning, MACOS_SIGNING_IDENTITY: identity, MACOS_NOTARY_PROFILE: 'astrafoundry' };
const imported = {
  MACOS_IME_PROVISIONING_PROFILE: provisioning,
  MACOS_SIGNING_IDENTITY: identity,
  MACOS_CERTIFICATE_P12_BASE64: Buffer.from('certificate test fixture').toString('base64'),
  MACOS_CERTIFICATE_PASSWORD: 'p12-test-password',
  APPLE_ID: 'build@example.invalid', APPLE_TEAM_ID: 'ABCDE12345', APPLE_APP_SPECIFIC_PASSWORD: 'notary-test-password',
};

test('no Apple configuration keeps preview builds unsigned; update signing is independent', () => {
  assert.equal(macosSigningSettings({}), null);
  assert.equal(macosSigningSettings({ MYGO_UPDATER_PRIVATE_KEY: 'update-key' }), null);
  const config = { macos: { minimumSystemVersion: '14.0' } };
  assert.deepEqual(macosBuildConfig(config, { command: 'build', platform: 'darwin', env: {} }), config);
  for (const options of [{ command: 'dev', platform: 'darwin' }, { command: 'build', platform: 'win32' }]) {
    assert.deepEqual(macosBuildConfig(config, { ...options, env: { APPLE_ID: 'partial' } }), config);
  }
});

test('partial Apple configuration cannot silently produce an unsigned build', () => {
  for (const name of Object.keys(imported).concat('MACOS_NOTARY_PROFILE', 'MACOS_NOTARY_KEYCHAIN')) {
    assert.throws(() => macosSigningSettings({ [name]: 'partial' }));
  }
  assert.throws(() => macosSigningSettings({ ...local, MACOS_CERTIFICATE_PASSWORD: 'partial' }), /MACOS_CERTIFICATE_P12_BASE64/);
  assert.throws(() => macosSigningSettings({ ...local, APPLE_ID: 'partial' }), /APPLE_TEAM_ID/);
  assert.throws(() => macosSigningSettings({ ...local, MACOS_SIGNING_IDENTITY: '-' }), /Developer ID Application/);
  assert.throws(() => macosSigningSettings({ ...imported, APPLE_TEAM_ID: 'OTHER12345' }), /team/i);
  assert.throws(() => macosSigningSettings({ ...imported, MACOS_CERTIFICATE_P12_BASE64: '%invalid' }), /Base64/);
});

test('macOS prerelease bundles use numeric metadata while archives retain the full version', () => {
  const config = { version: '0.1.0-alpha.4', macos: { infoPlist: { LSUIElement: true } } };
  for (const env of [{}, local]) {
    const built = macosBuildConfig(config, { command: 'build', platform: 'darwin', env });
    assert.equal(built.version, config.version);
    assert.equal(built.macos.infoPlist.CFBundleShortVersionString, '0.1.0');
    assert.equal(built.macos.infoPlist.CFBundleVersion, '0.1.0');
    assert.equal(built.macos.infoPlist.LSUIElement, true);
  }
  assert.equal(config.macos.infoPlist.CFBundleShortVersionString, undefined);
});

test('local signing uses an existing identity and profile without creating a keychain', () => {
  const calls = [];
  const run = (command, args) => {
    calls.push([command, args]);
    return command === 'security' ? `1) 1234 "${identity}"` : '';
  };
  const session = prepareMacOSSigning(macosSigningSettings(local), { run });
  assert.equal(session.env.MACOS_NOTARY_PROFILE, 'astrafoundry');
  const config = macosBuildConfig({ macos: { entitlements: 'entitlements.plist' } }, { command: 'build', platform: 'darwin', env: { ...local, ...session.env } });
  assert.equal(config.macos.entitlements, 'entitlements.plist');
  assert.equal(config.macos.signingIdentity, identity);
  assert.deepEqual(config.macos.notarize, { keychainProfile: 'astrafoundry' });
  assert.match(config.macos.helperEntitlements['ArionInput.app'], /ime.entitlements.plist$/);
  assert.equal(session.env.MACOS_IME_PROVISIONING_PROFILE, provisioning);
  session.cleanup();
  assert.ok(!calls.some(([, args]) => args.includes('create-keychain')));
});

test('CI credentials are isolated and cleanup restores the keychain search list', () => {
  const calls = [];
  const run = (command, args) => {
    calls.push([command, args]);
    if (args[0] === 'list-keychains' && !args.includes('-s')) return '    "/Users/test/Library/Keychains/login.keychain-db"\n';
    if (args[0] === 'find-identity') return `1) 1234 "${identity}"`;
    return '';
  };
  const session = prepareMacOSSigning(macosSigningSettings(imported), { run });
  const keychain = session.env.MACOS_NOTARY_KEYCHAIN;
  assert.ok(existsSync(keychain.substring(0, keychain.lastIndexOf('/'))));
  assert.ok(calls.some(([command, args]) => command === 'security' && args[0] === 'import' && args.includes('-T') && !args.includes('-A')));
  assert.ok(calls.some(([, args]) => args[0] === 'find-identity' && args.at(-1) === keychain));
  assert.ok(calls.some(([command, args]) => command === 'xcrun' && args[0] === 'notarytool' && args.includes('--keychain') && args.includes(keychain)));
  session.cleanup();
  assert.ok(!existsSync(keychain.substring(0, keychain.lastIndexOf('/'))));
  assert.ok(calls.some(([, args]) => JSON.stringify(args) === JSON.stringify(['list-keychains', '-d', 'user', '-s', '/Users/test/Library/Keychains/login.keychain-db'])));
  assert.ok(calls.some(([, args]) => args[0] === 'delete-keychain' && args[1] === keychain));
});

test('import failure still deletes the temporary keychain and files', () => {
  let keychain;
  let deleted = false;
  const run = (command, args) => {
    if (args[0] === 'list-keychains') return '"/Users/test/login.keychain-db"';
    if (args[0] === 'create-keychain') keychain = args.at(-1);
    if (args[0] === 'import') throw new Error('certificate import rejected');
    if (args[0] === 'delete-keychain') deleted = true;
    return '';
  };
  assert.throws(() => prepareMacOSSigning(macosSigningSettings(imported), { run }), /import rejected/);
  assert.ok(deleted);
  assert.ok(!existsSync(keychain.substring(0, keychain.lastIndexOf('/'))));
});

test('build subprocesses receive keychain references and keep the independent update key', () => {
  const env = stripAppleCredentials({ ...imported, ...local, MYGO_UPDATER_PRIVATE_KEY: 'update-test-key' });
  for (const name of ['MACOS_CERTIFICATE_P12_BASE64', 'MACOS_CERTIFICATE_PASSWORD', 'APPLE_ID', 'APPLE_TEAM_ID', 'APPLE_APP_SPECIFIC_PASSWORD']) {
    assert.equal(env[name], undefined);
  }
  assert.equal(env.MYGO_UPDATER_PRIVATE_KEY, 'update-test-key');
  assert.equal(env.MACOS_SIGNING_IDENTITY, identity);
});

test('notary authentication failure restores the search list and removes imported credentials', () => {
  const calls = [];
  let directory;
  const run = (command, args) => {
    calls.push([command, args]);
    if (args[0] === 'list-keychains' && !args.includes('-s')) return '"/Users/test/login.keychain-db"';
    if (args[0] === 'create-keychain') directory = args.at(-1).substring(0, args.at(-1).lastIndexOf('/'));
    if (args[0] === 'find-identity') return `1) 1234 "${identity}"`;
    if (args[0] === 'notarytool') throw new Error('notary authentication rejected');
    return '';
  };
  assert.throws(() => prepareMacOSSigning(macosSigningSettings(imported), { run }), /authentication rejected/);
  assert.ok(!existsSync(directory));
  assert.ok(calls.some(([, args]) => JSON.stringify(args) === JSON.stringify(['list-keychains', '-d', 'user', '-s', '/Users/test/login.keychain-db'])));
  assert.ok(calls.some(([, args]) => args[0] === 'delete-keychain'));
});

test('signing failures retain diagnostics without exposing secret argv or output', () => {
  const secret = 'private-test-password';
  const run = signingRunner([secret]);
  assert.throws(() => run(process.execPath, ['-e', 'process.stderr.write("invalid credentials: " + process.argv[1]); process.exit(1)', secret]), error => {
    assert.ok(error.message.includes('invalid credentials: [redacted]'));
    assert.ok(!error.message.includes(secret));
    return true;
  });
});


test('generic signing stays available but canonical macOS packages require an IME profile', () => {
  const env = { MACOS_SIGNING_IDENTITY: identity, MACOS_NOTARY_PROFILE: 'astrafoundry' };
  assert.equal(macosSigningSettings(env).identity, identity);
  assert.throws(() => macosBuildConfig({ macos: {} }, { command: 'build', platform: 'darwin', env }), /MACOS_IME_PROVISIONING_PROFILE/);
});
