// Profile authorization for the IMK host's Data Protection Keychain identity.
import { execFileSync } from 'node:child_process';
import { createHash, X509Certificate } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { isDeepStrictEqual } from 'node:util';

const runDefault = (command, args) => execFileSync(command, args, { encoding: 'utf8', maxBuffer: 16 * 1024 * 1024 }).trim();
export const imeEntitlementsPath = root => path.join(root, '.workspace/keyboard-runtime/signing/ime.entitlements.plist');
const teamFromIdentity = identity => identity.match(/\(([A-Z0-9]{10})\)$/)?.[1];
const permitsIdentifier = (allowed, requested) => typeof allowed === 'string' && (allowed === requested || (allowed.endsWith('.*') && requested.startsWith(allowed.slice(0, -1))));

export function validateIMEProfile(profile, { identity, bundleIdentifier, certificateHashes, now = new Date() }) {
  const team = teamFromIdentity(identity);
  if (!team) throw new Error('Invalid Developer ID signing identity.');
  if (!Array.isArray(profile.TeamIdentifier) || profile.TeamIdentifier.length !== 1 || profile.TeamIdentifier[0] !== team) throw new Error('IME provisioning profile TeamIdentifier does not match signing identity.');
  if (!profile.Platform?.includes('OSX') || profile.ProvisionsAllDevices !== true || profile.Entitlements?.['get-task-allow'] === true || profile.ProvisionedDevices) throw new Error('IME profile must authorize Developer ID distribution for macOS.');
  const expiration = new Date(profile.ExpirationDate);
  if (!Number.isFinite(expiration.valueOf()) || expiration <= now) throw new Error('IME provisioning profile is expired or has an invalid expiration.');
  const entitlements = { 'com.apple.application-identifier': `${team}.${bundleIdentifier}`, 'com.apple.developer.team-identifier': team };
  if (!permitsIdentifier(profile.Entitlements?.['com.apple.application-identifier'], entitlements['com.apple.application-identifier']) || profile.Entitlements?.['com.apple.developer.team-identifier'] !== team) throw new Error('IME provisioning profile does not authorize the host application identifier and team entitlement.');
  if (!certificateHashes?.length || !profile.certificateHashes?.some(hash => certificateHashes.includes(hash))) throw new Error('IME provisioning profile does not include the configured signing certificate.');
  return { entitlements, uuid: profile.UUID, expiration: expiration.toISOString(), team, bundleIdentifier };
}

export function inspectIMEProvisioning({ profilePath, identity, bundleIdentifier, run = runDefault, now, certificateHashes: providedCertificateHashes } = {}) {
  if (!profilePath || !path.isAbsolute(profilePath) || !existsSync(profilePath)) throw new Error('MACOS_IME_PROVISIONING_PROFILE must name an existing absolute-path Developer ID provisioning profile.');
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-ime-profile-'));
  try {
    const decoded = path.join(directory, 'profile.plist');
    // Decode Apple's signed CMS profile; final entitlement authorization is
    // still enforced by macOS when the profile-bearing signed host runs.
    writeFileSync(decoded, run('security', ['cms', '-D', '-i', profilePath]), { mode: 0o600 });
    const field = (key, format = 'json', expected) => run('plutil', ['-extract', key, format,
      ...(expected ? ['-expect', expected] : []), '-o', '-', decoded]);
    const profile = Object.fromEntries(['TeamIdentifier', 'Platform', 'Entitlements'].map(key => [key, JSON.parse(field(key))]));
    // plutil cannot emit a standalone boolean as a JSON property list.
    profile.ProvisionsAllDevices = JSON.parse(field('ProvisionsAllDevices', 'raw', 'bool'));
    profile.ExpirationDate = field('ExpirationDate', 'raw');
    profile.UUID = field('UUID', 'raw');
    // A Developer ID profile must not have a device allowlist. Querying a
    // missing optional key is deliberately separate from required extraction.
    const xml = run('plutil', ['-convert', 'xml1', '-o', '-', decoded]);
    if (/<key>ProvisionedDevices<\/key>/.test(xml)) profile.ProvisionedDevices = true;
    const certificates = run('plutil', ['-extract', 'DeveloperCertificates', 'xml1', '-o', '-', decoded]);
    profile.certificateHashes = [...certificates.matchAll(/<data>([\s\S]*?)<\/data>/g)].map(match => createHash('sha256').update(Buffer.from(match[1].replace(/\s/g, ''), 'base64')).digest('hex'));
    let certificateHashes = providedCertificateHashes;
    if (!certificateHashes) {
      const pem = run('security', ['find-certificate', '-a', '-c', identity, '-p']);
      const validIdentities = run('security', ['find-identity', '-v', '-p', 'codesigning']);
      const validHashes = [...validIdentities.matchAll(/\b([A-Fa-f0-9]{40})\s+"([^"]+)"/g)].filter(match => match[2] === identity).map(match => match[1].toLowerCase());
      certificateHashes = [...pem.matchAll(/-----BEGIN CERTIFICATE-----[\s\S]*?-----END CERTIFICATE-----/g)].map(match => new X509Certificate(match[0]))
        .filter(certificate => validHashes.includes(createHash('sha1').update(certificate.raw).digest('hex')))
        .map(certificate => createHash('sha256').update(certificate.raw).digest('hex'));
    }
    return validateIMEProfile(profile, { identity, bundleIdentifier, certificateHashes, now });
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
}

export function stageIMEProvisioning({ profilePath, identity, bundleIdentifier, bundle, entitlementsFile, run, now }) {
  const authorization = inspectIMEProvisioning({ profilePath, identity, bundleIdentifier, run, now });
  mkdirSync(path.dirname(entitlementsFile), { recursive: true });
  const entries = Object.entries(authorization.entitlements).map(([key, value]) => `<key>${key}</key><string>${value}</string>`).join('\n');
  if (!/^[A-Za-z0-9.-]+$/.test(bundleIdentifier)) throw new Error('Invalid IMK bundle identifier.');
  writeFileSync(entitlementsFile, `<?xml version="1.0" encoding="UTF-8"?>\n<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n<plist version="1.0"><dict>\n${entries}\n</dict></plist>\n`, { mode: 0o600 });
  copyFileSync(profilePath, path.join(bundle, 'Contents/embedded.provisionprofile'));
  return authorization;
}

export function verifyIMEProvisioning({ bundle, identity, run = runDefault }) {
  const info = JSON.parse(run('plutil', ['-convert', 'json', '-o', '-', path.join(bundle, 'Contents/Info.plist')]));
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-ime-entitlements-'));
  try {
    const prefix = path.join(directory, 'certificate-');
    run('codesign', ['--display', `--extract-certificates=${prefix}`, bundle]);
    const certificateHashes = [createHash('sha256').update(readFileSync(`${prefix}0`)).digest('hex')];
    const authorization = inspectIMEProvisioning({ profilePath: path.join(bundle, 'Contents/embedded.provisionprofile'), identity, bundleIdentifier: info.CFBundleIdentifier, run, certificateHashes });
    const xml = run('codesign', ['--display', '--entitlements', ':-', bundle]);
    const file = path.join(directory, 'entitlements.plist');
    writeFileSync(file, xml);
    const actual = JSON.parse(run('plutil', ['-convert', 'json', '-o', '-', file]));
    if (!isDeepStrictEqual(actual, authorization.entitlements)) throw new Error('Signed IMK host entitlements do not match its profile-authorized identity.');
    return authorization;
  } finally { rmSync(directory, { recursive: true, force: true }); }
}
