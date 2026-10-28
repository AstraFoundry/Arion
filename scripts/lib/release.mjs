import { createHash, createPrivateKey, createPublicKey, verify } from 'node:crypto';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';

export const releasePlatforms = ['darwin-arm64', 'windows-amd64'];
// Every release, including alpha previews, requires both supported targets.
export const platformsForRelease = () => [...releasePlatforms];
export function releaseMatrix(config) {
  const runners = { 'darwin-arm64': 'macos-26', 'windows-amd64': 'windows-2022' };
  return { include: platformsForRelease(config).map(platform => ({ os: runners[platform], platform })) };
}
export const digest = bytes => createHash('sha256').update(bytes).digest();
export function checkReleaseChangelog(version, markdown) {
  if (!markdown.split(/\r?\n/).includes(`## ${version}`)) throw new Error(`CHANGELOG.md requires a "## ${version}" section before release packaging`);
}
export const githubAssetName = name => name.replaceAll(' ', '.');
export const releaseStatus = (config, markdown) => markdown.replace(/\]\(([^)]+)\)/g, (link, target) => {
  if (/^(https?:|#)/.test(target)) return link;
  return `](https://github.com/${config.release.repository}/blob/v${config.version}/${path.posix.normalize(path.posix.join('docs/operations', target))})`;
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

export function archiveName(config, platform) {
  if (!releasePlatforms.includes(platform)) throw new Error(`Unsupported release platform: ${platform}`);
  return `${config.name.toLowerCase()}-${config.version}-${platform}.tar.gz`;
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
  const prefix = `${config.updates.url}/`;
  if (manifest.url !== `${prefix}${archiveName(config, platform)}`) throw new Error(`Wrong update archive URL for ${platform}`);
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

// Release bucket layout: packages/contracts/releases/README.md.
export const stagedSchema = 'astrafoundry.release.v1';
export const releaseDocuments = ['SHA256SUMS', 'RELEASE-STATUS.md'];
export const stagedRecordKey = version => `releases/${version}/release.json`;
export const fileKey = name => `files/${name}`;
export const documentKey = (version, name) => `releases/${version}/${name}`;
const contentTypes = [[/\.dmg$/, 'application/x-apple-diskimage'], [/\.exe$/, 'application/vnd.microsoft.portable-executable'], [/\.tar\.gz$/, 'application/gzip'], [/\.json$/, 'application/json'], [/\.md$/, 'text/markdown; charset=utf-8'], [/^SHA256SUMS$/, 'text/plain; charset=utf-8']];
export const contentType = name => contentTypes.find(([pattern]) => pattern.test(name))?.[1] ?? 'application/octet-stream';

const fileEntry = (directory, name) => {
  const bytes = readFileSync(requireFile(directory, name));
  return { name: githubAssetName(name), size: bytes.length, sha256: digest(bytes).toString('hex') };
};

/**
 * The staged record of a verified release directory. Every file it names is
 * uploaded flat under files/ by its download name; nothing is published.
 */
export function stagedRecord(directory, config, platforms, { commit, createdAt }) {
  const targets = {};
  let notes = '';
  for (const platform of platforms) {
    const manifest = JSON.parse(readFileSync(path.join(directory, `update-${platform}.json`), 'utf8'));
    const assetName = url => url.slice(`${config.updates.url}/`.length);
    notes ||= manifest.notes ?? '';
    targets[platform] = {
      installer: fileEntry(directory, installerName(config, platform)),
      update: fileEntry(directory, assetName(manifest.url)),
      deltas: (manifest.deltas ?? []).map(delta => fileEntry(directory, assetName(delta.url))),
      manifest,
    };
  }
  return {
    schema: stagedSchema,
    product: config.release.product,
    version: config.version,
    prerelease: config.version.includes('-'),
    createdAt,
    notes,
    source: { kind: 'ci', repository: config.release.repository, tag: `v${config.version}`, commit },
    targets,
    documents: releaseDocuments,
  };
}

/** Uploads in contract order: files, documents, then the record that stages them. */
export function stagingUploads(directory, record) {
  const files = Object.values(record.targets).flatMap(target => [target.installer, target.update, ...target.deltas]);
  const local = name => readdirSync(directory).find(file => githubAssetName(file) === name);
  return [
    ...files.map(file => ({ key: fileKey(file.name), path: path.join(directory, local(file.name)), contentType: contentType(file.name), cacheControl: 'public, max-age=31536000, immutable' })),
    ...record.documents.map(name => ({ key: documentKey(record.version, name), path: path.join(directory, name), contentType: contentType(name), cacheControl: 'public, max-age=31536000, immutable' })),
    { key: stagedRecordKey(record.version), body: `${JSON.stringify(record, null, 2)}\n`, contentType: 'application/json', cacheControl: 'no-cache' },
  ];
}

/** S3 staging settings from the environment; partial configuration fails. */
export function stagingSettings(env) {
  const names = ['ARION_RELEASES_S3_ENDPOINT', 'ARION_RELEASES_S3_REGION', 'ARION_RELEASES_S3_BUCKET', 'ARION_RELEASES_S3_PREFIX', 'AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY'];
  const missing = names.filter(name => !env[name]?.trim());
  if (missing.length) throw new Error(`Release staging requires ${missing.join(', ')}`);
  const endpoint = new URL(env.ARION_RELEASES_S3_ENDPOINT);
  if (endpoint.protocol !== 'https:' || endpoint.pathname.replace(/\/+$/, '') || endpoint.search) throw new Error('ARION_RELEASES_S3_ENDPOINT must be an https URL without a path');
  return { endpoint: endpoint.origin, region: env.ARION_RELEASES_S3_REGION, bucket: env.ARION_RELEASES_S3_BUCKET, prefix: env.ARION_RELEASES_S3_PREFIX.replace(/^\/+|\/+$/g, '') };
}
