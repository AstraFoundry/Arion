import { createHash } from 'node:crypto';
import { readFileSync, mkdirSync, mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
export const standardCenter = JSON.parse(readFileSync(path.join(root, 'apps/desktop/internal/center/default.json'), 'utf8'));
const fields = new Map([
  ['ARION_CENTER_URL', 'url'], ['ARION_CENTER_AUTH_MODE', 'authMode'],
  ['ARION_CENTER_OIDC_ISSUER', 'issuer'], ['ARION_CENTER_OIDC_CLIENT_ID', 'clientID'],
  ['ARION_CENTER_OIDC_RESOURCE', 'resource'], ['ARION_CENTER_OIDC_SCOPES', 'scopes'],
  ['ARION_CENTER_OIDC_ENTITLEMENT', 'entitlement'],
]);
// Match encoding/json's canonical string escaping so Go and tooling hash identical bytes.
export const centerFingerprint = profile => createHash('sha256').update(JSON.stringify(profile).replace(/[<>&\u2028\u2029]/g, char => `\\u${char.charCodeAt(0).toString(16).padStart(4, '0')}`)).digest('hex');
export const isStandardCenter = profile => centerFingerprint(profile) === centerFingerprint(standardCenter);

function checkURL(value, name) {
  let url;
  try { url = new URL(value); } catch { throw new Error(`Invalid ${name}`); }
  const host = url.hostname;
  const loopback = host === 'localhost' || host === '[::1]' || /^127(?:\.\d{1,3}){3}$/.test(host);
  if (url.username || url.password || value.includes('?') || value.includes('#') ||
      (url.protocol !== 'https:' && !(url.protocol === 'http:' && loopback))) throw new Error(`Invalid ${name}: use HTTPS or explicit loopback HTTP, without credentials, query or fragment`);
  return url.toString().replace(/\/+$/, '');
}

export function resolveCenter(env) {
  for (const name of Object.keys(env)) {
    if (name.startsWith('ARION_CENTER_') && !fields.has(name)) throw new Error(`Unsupported public center setting: ${name}`);
  }
  const present = [...fields.keys()].filter(name => Object.hasOwn(env, name));
  if (!present.length) return structuredClone(standardCenter);
  for (const name of present) if (!env[name]?.trim()) throw new Error(`Empty center setting: ${name}; omit it to use defaults`);
  const matchesStandard = present.every(name => {
    const field = fields.get(name);
    let value = env[name].trim();
    if (['url', 'issuer', 'resource'].includes(field)) value = checkURL(value, name);
    if (field === 'scopes') value = value.split(/\s+/);
    return JSON.stringify(value) === JSON.stringify(standardCenter[field]);
  });
  if (matchesStandard) return structuredClone(standardCenter);
  if (!env.ARION_CENTER_URL || !env.ARION_CENTER_AUTH_MODE) throw new Error('Center overrides require ARION_CENTER_URL and ARION_CENTER_AUTH_MODE');
  const profile = { url: checkURL(env.ARION_CENTER_URL.trim(), 'center URL'), authMode: env.ARION_CENTER_AUTH_MODE.trim(), issuer: '', clientID: '', resource: '', scopes: [], entitlement: '' };
  const oidc = [...fields.keys()].filter(name => name.startsWith('ARION_CENTER_OIDC_'));
  if (profile.authMode === 'device') {
    if (oidc.some(name => Object.hasOwn(env, name))) throw new Error('Device authentication cannot contain OIDC settings');
  } else if (profile.authMode === 'oidc') {
    const missing = oidc.filter(name => !env[name]?.trim());
    if (missing.length) throw new Error(`Custom OIDC requires ${missing.join(', ')}`);
    for (const name of oidc) profile[fields.get(name)] = env[name].trim();
    profile.issuer = checkURL(profile.issuer, 'OIDC issuer');
    profile.resource = checkURL(profile.resource, 'OIDC resource');
    profile.scopes = profile.scopes.split(/\s+/);
    if (!profile.scopes.includes('openid')) throw new Error('OIDC scopes require openid');
  } else throw new Error('ARION_CENTER_AUTH_MODE must be device or oidc');
  return profile;
}

export function checkOfficialCenter(env, profile = resolveCenter(env)) {
  if ((env.GITHUB_REF_TYPE === 'tag' || env.ARION_OFFICIAL_RELEASE === '1') && !isStandardCenter(profile)) throw new Error('Official releases require the exact standard center profile');
}

export function prepareCenterBuild(env) {
  const profile = resolveCenter(env);
  checkOfficialCenter(env, profile);
  if (/(?:^|\s|["'])-overlay(?:=|\s|["'])/.test(env.GOFLAGS ?? '')) throw new Error('An external Go overlay conflicts with center profile injection');
  const workspace = path.join(root, '.workspace');
  mkdirSync(workspace, { recursive: true });
  const directory = mkdtempSync(path.join(workspace, 'center-build-'));
  try {
    const source = path.join(directory, 'embedded.go');
    writeFileSync(source, `package center\n\nvar embeddedProfile = []byte(${JSON.stringify(JSON.stringify(profile))})\n`);
    const overlay = path.join(directory, 'overlay.json');
    writeFileSync(overlay, JSON.stringify({ Replace: { [path.join(root, 'apps/desktop/internal/center/embedded.go')]: source } }));
    return {
      profile, fingerprint: centerFingerprint(profile),
      env: { GOFLAGS: `${env.GOFLAGS ?? ''} "-overlay=${overlay}"`.trim(), ARION_BUILD_CENTER_FINGERPRINT: centerFingerprint(profile) },
      cleanup: () => rmSync(directory, { recursive: true, force: true }),
    };
  } catch (error) {
    rmSync(directory, { recursive: true, force: true });
    throw error;
  }
}

export function centerBuildConfig(config, env = process.env) {
  const profile = resolveCenter(env);
  checkOfficialCenter(env, profile);
  if (!isStandardCenter(profile) && env.ARION_BUILD_CENTER_FINGERPRINT !== centerFingerprint(profile)) throw new Error('Custom center packages must use make build desktop or make desktop-dev to embed the profile');
  if (isStandardCenter(profile)) return config;
  const { updates, ...custom } = config;
  return custom;
}
