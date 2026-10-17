import { spawnSync } from 'node:child_process';
import { randomBytes } from 'node:crypto';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';

const certificateVariables = ['MACOS_CERTIFICATE_P12_BASE64', 'MACOS_CERTIFICATE_PASSWORD'];
const notaryVariables = ['APPLE_ID', 'APPLE_TEAM_ID', 'APPLE_APP_SPECIFIC_PASSWORD'];
const signingVariables = ['MACOS_SIGNING_IDENTITY', 'MACOS_NOTARY_PROFILE', 'MACOS_NOTARY_KEYCHAIN', ...certificateVariables, ...notaryVariables];

function requireGroup(env, names) {
  const present = names.filter(name => env[name]?.trim());
  if (present.length && present.length !== names.length) {
    throw new Error(`Incomplete macOS signing configuration; missing ${names.filter(name => !present.includes(name)).join(', ')}`);
  }
  return present.length > 0;
}

export function macosSigningSettings(env) {
  if (!signingVariables.some(name => env[name]?.trim())) return null;
  const identity = env.MACOS_SIGNING_IDENTITY?.trim();
  if (!/^Developer ID Application: .+ \([A-Z0-9]{10}\)$/.test(identity ?? '')) {
    throw new Error('MACOS_SIGNING_IDENTITY must be a full Developer ID Application identity, including its Team ID');
  }
  const importCertificate = requireGroup(env, certificateVariables);
  if (importCertificate) certificateBytes(env.MACOS_CERTIFICATE_P12_BASE64);
  const storeCredentials = requireGroup(env, notaryVariables);
  const profile = env.MACOS_NOTARY_PROFILE?.trim() || (storeCredentials ? 'astrafoundry' : '');
  if (!profile) throw new Error('Set MACOS_NOTARY_PROFILE or all APPLE_ID, APPLE_TEAM_ID and APPLE_APP_SPECIFIC_PASSWORD values');
  if (storeCredentials && !identity.endsWith(`(${env.APPLE_TEAM_ID.trim()})`)) throw new Error('Apple notarization Team ID does not match the signing identity');
  if (storeCredentials && env.MACOS_NOTARY_KEYCHAIN?.trim()) throw new Error('APPLE_* credentials create a temporary keychain; do not also set MACOS_NOTARY_KEYCHAIN');
  return {
    identity, profile, keychain: env.MACOS_NOTARY_KEYCHAIN?.trim(),
    certificate: importCertificate ? env.MACOS_CERTIFICATE_P12_BASE64 : undefined,
    certificatePassword: env.MACOS_CERTIFICATE_PASSWORD,
    appleId: storeCredentials ? env.APPLE_ID.trim() : undefined,
    teamId: env.APPLE_TEAM_ID?.trim(), appPassword: env.APPLE_APP_SPECIFIC_PASSWORD,
  };
}

export function macosBuildConfig(config, { command, platform = process.platform, env = process.env }) {
  if (command !== 'build' || platform !== 'darwin') return config;
  const settings = macosSigningSettings(env);
  if (!settings) return config;
  const notarize = { keychainProfile: settings.profile };
  if (settings.keychain) notarize.keychain = settings.keychain;
  return { ...config, macos: { ...config.macos, signingIdentity: settings.identity, notarize } };
}

// Never include child-process argv in errors: security/notarytool receive passwords.
export function signingRunner(secrets = []) {
  return (command, args) => {
    const result = spawnSync(command, args, { encoding: 'utf8' });
    const output = `${result.stdout ?? ''}${result.stderr ?? ''}`;
    if (!result.error && result.status === 0) return output;
    let diagnostic = output || result.error?.code || `exit ${result.status}, signal ${result.signal}`;
    for (const secret of secrets.filter(Boolean)) diagnostic = diagnostic.replaceAll(secret, '[redacted]');
    throw new Error(`${command} failed: ${diagnostic.trim()}`);
  };
}

function certificateBytes(encoded) {
  const value = encoded.replace(/\s/g, '');
  const bytes = Buffer.from(value, 'base64');
  if (!bytes.length || bytes.toString('base64') !== value) throw new Error('MACOS_CERTIFICATE_P12_BASE64 is not valid Base64');
  return bytes;
}

function importCertificate(settings, session, password, run) {
  const file = path.join(session.directory, 'certificate.p12');
  writeFileSync(file, certificateBytes(settings.certificate), { mode: 0o600 });
  try {
    run('security', ['import', file, '-P', settings.certificatePassword, '-t', 'cert', '-f', 'pkcs12', '-k', session.keychain, '-T', '/usr/bin/codesign', '-T', '/usr/bin/security']);
    run('security', ['set-key-partition-list', '-S', 'apple-tool:,apple:,codesign:', '-s', '-k', password, session.keychain]);
  } finally {
    rmSync(file, { force: true });
  }
}

export function prepareMacOSSigning(settings, { run, temporaryRoot = os.tmpdir() } = {}) {
  const password = randomBytes(32).toString('hex');
  run ??= signingRunner([password, settings.certificate, settings.certificatePassword, settings.appPassword]);
  const session = { directory: undefined, keychain: undefined, created: false, searchList: undefined, changedSearchList: false, cleaned: false };
  const cleanup = () => {
    if (session.cleaned) return;
    session.cleaned = true;
    const failures = [];
    const attempt = operation => { try { operation(); } catch (error) { failures.push(error); } };
    if (session.changedSearchList) attempt(() => run('security', ['list-keychains', '-d', 'user', '-s', ...session.searchList]));
    if (session.created) attempt(() => run('security', ['delete-keychain', session.keychain]));
    if (session.directory) attempt(() => rmSync(session.directory, { recursive: true, force: true }));
    if (failures.length) throw new AggregateError(failures, 'Could not completely clean up the macOS signing session');
  };
  try {
    if (settings.certificate || settings.appleId) {
      session.directory = mkdtempSync(path.join(temporaryRoot, 'arion-signing-'));
      session.keychain = path.join(session.directory, 'signing.keychain-db');
      session.searchList = [...run('security', ['list-keychains', '-d', 'user']).matchAll(/"([^"]+)"/g)].map(match => match[1]);
      run('security', ['create-keychain', '-p', password, session.keychain]);
      session.created = true;
      run('security', ['set-keychain-settings', '-lut', '21600', session.keychain]);
      run('security', ['unlock-keychain', '-p', password, session.keychain]);
      if (settings.certificate) importCertificate(settings, session, password, run);
      session.changedSearchList = true;
      run('security', ['list-keychains', '-d', 'user', '-s', session.keychain, ...session.searchList]);
    }
    const identities = run('security', ['find-identity', '-v', '-p', 'codesigning', ...(settings.certificate ? [session.keychain] : [])]);
    if (!identities.includes(`"${settings.identity}"`)) throw new Error('The configured Developer ID identity and private key are not available in the keychain');
    const keychain = settings.appleId ? session.keychain : settings.keychain;
    const keychainArgs = keychain ? ['--keychain', keychain] : [];
    if (settings.appleId) {
      run('xcrun', ['notarytool', 'store-credentials', settings.profile, '--apple-id', settings.appleId, '--team-id', settings.teamId, '--password', settings.appPassword, ...keychainArgs]);
    } else {
      run('xcrun', ['notarytool', 'history', '--keychain-profile', settings.profile, ...keychainArgs, '--output-format', 'json']);
    }
    const env = { MACOS_SIGNING_IDENTITY: settings.identity, MACOS_NOTARY_PROFILE: settings.profile };
    if (keychain) env.MACOS_NOTARY_KEYCHAIN = keychain;
    return { env, cleanup };
  } catch (error) {
    try { cleanup(); } catch (cleanupError) { throw new AggregateError([error, cleanupError], 'macOS signing setup and cleanup failed'); }
    throw error;
  }
}

export function stripAppleCredentials(env) {
  const result = { ...env };
  for (const name of [...certificateVariables, ...notaryVariables]) delete result[name];
  return result;
}
