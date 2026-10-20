import test from 'node:test';
import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { checkRelease, digest, githubAssetName, installerName, platformAssets, platformsForRelease, releaseMatrix, releasePlatforms } from '../../../scripts/lib/release.mjs';

function fixture(t) {
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-release-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const { privateKey, publicKey } = generateKeyPairSync('ed25519');
  const rawPublic = publicKey.export({ type: 'spki', format: 'der' }).subarray(-32);
  const rawPrivate = privateKey.export({ type: 'pkcs8', format: 'der' }).subarray(-32);
  const config = { name: 'Arion', version: '0.1.0', updates: { github: 'example/arion', publicKey: rawPublic.toString('base64') } };
  const manifests = {};
  for (const platform of releasePlatforms) {
    writeFileSync(path.join(directory, installerName(config, platform)), 'installer fixture');
    const name = `arion-${config.version}-${platform}.tar.gz`;
    const bytes = Buffer.from(`archive fixture for ${platform}`);
    writeFileSync(path.join(directory, name), bytes);
    const manifest = { version: config.version, url: `https://github.com/example/arion/releases/download/v${config.version}/${name}`, size: bytes.length, signature: sign(null, digest(bytes), privateKey).toString('base64') };
    manifests[platform] = manifest;
    writeFileSync(path.join(directory, `update-${platform}.json`), JSON.stringify(manifest));
  }
  return { directory, config, manifests, secret: Buffer.concat([rawPrivate, rawPublic]).toString('base64') };
}

test('checksum names match GitHub uploaded installer filenames', () => {
  assert.equal(githubAssetName('Arion 0.1.0-alpha.4 arm64.dmg'), 'Arion.0.1.0-alpha.4.arm64.dmg');
  assert.equal(githubAssetName('update-darwin-arm64.json'), 'update-darwin-arm64.json');
});

test('release preflight rejects missing, mismatched and malformed signing credentials', t => {
  const { config, secret } = fixture(t);
  assert.doesNotThrow(() => checkRelease(config, 'v0.1.0', secret));
  assert.throws(() => checkRelease(config, 'v0.1.1', secret), /tag/);
  assert.throws(() => checkRelease(config, 'v0.1.0', ''), /independent of Apple/);
  assert.throws(() => checkRelease(config, 'v0.1.0', 'invalid'), /private key/);
  assert.throws(() => checkRelease(config, 'v0.1.0', `%${secret}`), /private key/);
  const other = fixture(t);
  assert.throws(() => checkRelease(config, 'v0.1.0', other.secret), /does not match/);
});

test('all platform installers and archives must exist and carry valid update signatures', t => {
  const { directory, config } = fixture(t);
  for (const platform of releasePlatforms) assert.equal(platformAssets(directory, config, platform).length, 3);
  rmSync(path.join(directory, installerName(config, 'windows-amd64')));
  assert.throws(() => releasePlatforms.flatMap(platform => platformAssets(directory, config, platform)), /ENOENT/);
});

test('alpha releases require ARM64 macOS while stable and other prereleases require all platforms', t => {
  const { directory, config } = fixture(t);
  assert.deepEqual(platformsForRelease({ version: '0.1.0-alpha.1' }), ['darwin-arm64']);
  assert.deepEqual(releaseMatrix({ version: '0.1.0-alpha.1' }), { include: [{ os: 'macos-15', platform: 'darwin-arm64' }] });
  for (const version of ['0.1.0', '0.1.0-beta.1', '0.1.0-alphabet']) assert.deepEqual(platformsForRelease({ version }), releasePlatforms);
  for (const platform of ['darwin-amd64', 'windows-amd64']) rmSync(path.join(directory, installerName(config, platform)));
  const platforms = platformsForRelease({ version: '0.1.0-alpha.1' });
  assert.equal(platforms.flatMap(platform => platformAssets(directory, config, platform)).length, 3);
  rmSync(path.join(directory, installerName(config, 'darwin-arm64')));
  assert.throws(() => platforms.flatMap(platform => platformAssets(directory, config, platform)), /ENOENT/);
});

test('a same-size archive modification is rejected even when its manifest is unchanged', t => {
  const { directory, config } = fixture(t);
  const file = path.join(directory, 'arion-0.1.0-darwin-arm64.tar.gz');
  const bytes = readFileSync(file);
  bytes[0] ^= 1;
  writeFileSync(file, bytes);
  assert.throws(() => platformAssets(directory, config, 'darwin-arm64'), /signature/);
});

test('update versions, origins and platform archive names cannot be substituted', t => {
  const { directory, config, manifests } = fixture(t);
  const file = path.join(directory, 'update-darwin-arm64.json');
  for (const changed of [
    { version: '0.2.0' },
    { url: manifests['windows-amd64'].url },
    { url: 'https://example.com/arion.tar.gz' },
    { size: 1 },
  ]) {
    writeFileSync(file, JSON.stringify({ ...manifests['darwin-arm64'], ...changed }));
    assert.throws(() => platformAssets(directory, config, 'darwin-arm64'));
  }
});

test('ordinary CI can retain installers without an update key, signed releases cannot', t => {
  const { directory, config } = fixture(t);
  rmSync(path.join(directory, 'update-darwin-arm64.json'));
  assert.equal(platformAssets(directory, config, 'darwin-arm64', false).length, 1);
  assert.throws(() => platformAssets(directory, config, 'darwin-arm64'), /ENOENT/);
});
