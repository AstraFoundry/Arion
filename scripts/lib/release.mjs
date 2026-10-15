import { createHash, createPrivateKey, createPublicKey, verify } from 'node:crypto';
import { readFileSync, statSync } from 'node:fs';
import path from 'node:path';

export const releasePlatforms = ['darwin-arm64', 'darwin-amd64', 'windows-amd64'];
export const digest = bytes => createHash('sha256').update(bytes).digest();
export const releaseStatus = (config, markdown) => markdown.replace(/\]\(([^)]+)\)/g, (link, target) => {
  if (/^(https?:|#)/.test(target)) return link;
  return `](https://github.com/${config.updates.github}/blob/v${config.version}/${path.posix.normalize(path.posix.join('docs/operations', target))})`;
});
const decodeKey = (encoded, size, label) => {
  const value = encoded.trim();
  const bytes = Buffer.from(value, 'base64');
  if (bytes.length !== size || bytes.toString('base64') !== value) throw new Error(`Invalid update ${label}`);
  return bytes;
};
const publicKey = encoded => {
  const bytes = decodeKey(encoded, 32, 'public key');
  return createPublicKey({ key: Buffer.concat([Buffer.from('302a300506032b6570032100', 'hex'), bytes]), type: 'spki', format: 'der' });
};

export function checkRelease(config, tag, secret) {
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(config.version) || tag !== `v${config.version}`) throw new Error('Release tag must match the app version');
  if (!secret?.trim()) throw new Error('MYGO_UPDATER_PRIVATE_KEY is required for signed updates; it is independent of Apple Developer credentials');
  const bytes = decodeKey(secret, 64, 'private key');
  const key = createPrivateKey({ key: Buffer.concat([Buffer.from('302e020100300506032b657004220420', 'hex'), bytes.subarray(0, 32)]), type: 'pkcs8', format: 'der' });
  const derived = createPublicKey(key).export({ type: 'spki', format: 'der' }).subarray(-32);
  if (!derived.equals(bytes.subarray(32)) || !derived.equals(decodeKey(config.updates.publicKey, 32, 'public key'))) throw new Error('Update private key does not match the built-in public key');
}

export function installerName(config, platform) {
  if (!releasePlatforms.includes(platform)) throw new Error(`Unsupported release platform: ${platform}`);
  const arch = platform.split('-')[1];
  return platform.startsWith('darwin-') ? `${config.name} ${config.version} ${arch}.dmg` : `${config.name} Setup ${config.version} ${arch}.exe`;
}

function requireFile(directory, name) {
  const file = path.join(directory, name);
  const info = statSync(file);
  if (!info.isFile() || !info.size) throw new Error(`Missing or empty release asset: ${name}`);
  return file;
}

export function platformAssets(directory, config, platform, signed = true) {
  const files = [installerName(config, platform)];
  requireFile(directory, files[0]);
  if (!signed) return files;
  const name = `update-${platform}.json`;
  const manifestFile = requireFile(directory, name);
  if (statSync(manifestFile).size > 1024 * 1024) throw new Error('Update manifest exceeds the updater limit');
  const manifest = JSON.parse(readFileSync(manifestFile, 'utf8'));
  if (manifest.version !== config.version) throw new Error(`Wrong update version for ${platform}`);
  const key = publicKey(config.updates.publicKey);
  const prefix = `https://github.com/${config.updates.github}/releases/download/v${config.version}/`;
  const archiveName = `${config.name.toLowerCase()}-${config.version}-${platform}.tar.gz`;
  if (manifest.url !== `${prefix}${archiveName}`) throw new Error(`Wrong update archive URL for ${platform}`);
  for (const asset of [manifest, ...(manifest.deltas ?? [])]) {
    if (!asset.url.startsWith(prefix)) throw new Error(`Wrong update origin for ${platform}`);
    const fileName = asset.url.slice(prefix.length);
    if (!fileName || fileName.includes('/') || fileName.includes('\\') || decodeURIComponent(fileName) !== fileName || fileName === '..') throw new Error('Unsafe update asset name');
    if (asset !== manifest && !fileName.endsWith(`-${platform}.delta`)) throw new Error('Wrong delta platform');
    const file = requireFile(directory, fileName);
    const signature = decodeKey(asset.signature ?? '', 64, 'signature');
    if (!Number.isSafeInteger(asset.size) || asset.size > 1024 ** 3 || statSync(file).size !== asset.size || !verify(null, digest(readFileSync(file)), key, signature)) throw new Error(`Update size or signature verification failed: ${fileName}`);
    files.push(fileName);
  }
  return [...files, name];
}
