import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { digest, platformAssets, stagedRecord, stagingSettings, stagingUploads } from './release.mjs';

const fixture = JSON.parse(readFileSync(new URL('../../packages/contracts/releases/staged-release-v1.json', import.meta.url), 'utf8'));

function signedRelease() {
  const { publicKey, privateKey } = generateKeyPairSync('ed25519');
  const config = {
    name: 'Arion',
    version: '0.1.0-alpha.6',
    updates: { publicKey: publicKey.export({ type: 'spki', format: 'der' }).subarray(-32).toString('base64'), url: 'https://api.arion.example/v1/updates' },
    release: { product: 'arion', repository: 'AstraFoundry/Arion' },
  };
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-release-'));
  const archive = Buffer.from('archive bytes');
  writeFileSync(path.join(directory, 'Arion 0.1.0-alpha.6 arm64.dmg'), 'installer bytes');
  writeFileSync(path.join(directory, 'arion-0.1.0-alpha.6-darwin-arm64.tar.gz'), archive);
  const manifest = { version: config.version, notes: '- Faster', url: `${config.updates.url}/arion-0.1.0-alpha.6-darwin-arm64.tar.gz`, size: archive.length, signature: sign(null, digest(archive), privateKey).toString('base64') };
  writeFileSync(path.join(directory, 'update-darwin-arm64.json'), JSON.stringify(manifest));
  for (const name of ['SHA256SUMS', 'RELEASE-STATUS.md']) writeFileSync(path.join(directory, name), name);
  return { config, directory, manifest };
}

test('verified assets are staged under the contract layout', () => {
  const { config, directory, manifest } = signedRelease();
  try {
    assert.deepEqual(platformAssets(directory, config, 'darwin-arm64'), ['Arion 0.1.0-alpha.6 arm64.dmg', 'arion-0.1.0-alpha.6-darwin-arm64.tar.gz', 'update-darwin-arm64.json']);
    const record = stagedRecord(directory, config, ['darwin-arm64'], { commit: 'abc', createdAt: '2026-10-20T07:00:00.000Z' });
    assert.deepEqual(Object.keys(record).sort(), Object.keys(fixture).sort());
    assert.equal(record.schema, fixture.schema);
    assert.equal(record.prerelease, true);
    assert.equal(record.notes, '- Faster');
    assert.equal(record.targets['darwin-arm64'].installer.name, 'Arion.0.1.0-alpha.6.arm64.dmg');
    assert.equal(record.targets['darwin-arm64'].update.sha256, digest(Buffer.from('archive bytes')).toString('hex'));
    assert.deepEqual(record.targets['darwin-arm64'].manifest, manifest);
    const uploads = stagingUploads(directory, record);
    assert.deepEqual(uploads.map(upload => upload.key), [
      'files/Arion.0.1.0-alpha.6.arm64.dmg',
      'files/arion-0.1.0-alpha.6-darwin-arm64.tar.gz',
      'releases/0.1.0-alpha.6/SHA256SUMS',
      'releases/0.1.0-alpha.6/RELEASE-STATUS.md',
      'releases/0.1.0-alpha.6/release.json',
    ]);
    assert.equal(uploads[0].path, path.join(directory, 'Arion 0.1.0-alpha.6 arm64.dmg'));
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test('a manifest pointing outside the center is rejected', () => {
  const { config, directory, manifest } = signedRelease();
  try {
    writeFileSync(path.join(directory, 'update-darwin-arm64.json'), JSON.stringify({ ...manifest, url: 'https://github.com/AstraFoundry/Arion/releases/download/v0.1.0-alpha.6/arion-0.1.0-alpha.6-darwin-arm64.tar.gz' }));
    assert.throws(() => platformAssets(directory, config, 'darwin-arm64'), /Wrong update archive URL/);
  } finally {
    rmSync(directory, { recursive: true, force: true });
  }
});

test('staging settings are complete or rejected', () => {
  const env = { ARION_RELEASES_S3_ENDPOINT: 'https://account.r2.cloudflarestorage.com', ARION_RELEASES_S3_REGION: 'auto', ARION_RELEASES_S3_BUCKET: 'releases', ARION_RELEASES_S3_PREFIX: '/arion/', AWS_ACCESS_KEY_ID: 'id', AWS_SECRET_ACCESS_KEY: 'secret' };
  assert.deepEqual(stagingSettings(env), { endpoint: 'https://account.r2.cloudflarestorage.com', region: 'auto', bucket: 'releases', prefix: 'arion' });
  assert.throws(() => stagingSettings({ ...env, AWS_SECRET_ACCESS_KEY: '' }), /AWS_SECRET_ACCESS_KEY/);
  assert.throws(() => stagingSettings({ ...env, ARION_RELEASES_S3_ENDPOINT: 'https://host/bucket' }), /without a path/);
});
