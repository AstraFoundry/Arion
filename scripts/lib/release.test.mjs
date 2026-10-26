import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { checkReleaseChangelog, digest, installerName, platformAssets, platformsForRelease, releaseMatrix, stagedRecord, stagingSettings, stagingUploads } from './release.mjs';

const fixture = JSON.parse(readFileSync(new URL('../../packages/contracts/releases/staged-release-v1.json', import.meta.url), 'utf8'));

test('release preflight rejects a changelog left at the previous version', () => {
  assert.throws(() => checkReleaseChangelog('0.1.0-alpha.6', '## 0.1.0-alpha.5\n\n- Previous release\n'), /CHANGELOG.md requires/);
  const config = JSON.parse(readFileSync(new URL('../../apps/desktop/desktop.json', import.meta.url), 'utf8'));
  assert.doesNotThrow(() => checkReleaseChangelog(config.version, readFileSync(new URL('../../apps/desktop/CHANGELOG.md', import.meta.url), 'utf8')));
});

test('future releases require supported targets and reject Intel macOS installers', () => {
  for (const version of ['0.1.0', '0.1.0-beta.1', '0.1.0-alpha', '0.1.0-alpha.12']) {
    assert.deepEqual(platformsForRelease({ version }), ['darwin-arm64', 'windows-amd64']);
    assert.deepEqual(releaseMatrix({ version }).include, [
      { os: 'macos-26', platform: 'darwin-arm64' },
      { os: 'windows-2022', platform: 'windows-amd64' },
    ]);
  }
  assert.throws(() => installerName({ name: 'Arion', version: '0.1.0' }, 'darwin-amd64'), /Unsupported release platform/);
});

function signedRelease(platform = 'darwin-arm64') {
  const { publicKey, privateKey } = generateKeyPairSync('ed25519');
  const config = {
    name: 'Arion',
    version: '0.1.0-alpha.6',
    updates: { publicKey: publicKey.export({ type: 'spki', format: 'der' }).subarray(-32).toString('base64'), url: 'https://api.arion.example/v1/updates' },
    release: { product: 'arion', repository: 'AstraFoundry/Arion' },
  };
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-release-'));
  const archive = Buffer.from('archive bytes');
  writeFileSync(path.join(directory, installerName(config, platform)), 'installer bytes');
  writeFileSync(path.join(directory, `arion-0.1.0-alpha.6-${platform}.tar.gz`), archive);
  const manifest = { version: config.version, notes: '- Faster', url: `${config.updates.url}/arion-0.1.0-alpha.6-${platform}.tar.gz`, size: archive.length, signature: sign(null, digest(archive), privateKey).toString('base64') };
  writeFileSync(path.join(directory, `update-${platform}.json`), JSON.stringify(manifest));
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

test('Windows release assets require a matching signed archive and manifest', () => {
  const platform = 'windows-amd64';
  const { config, directory, manifest } = signedRelease(platform);
  const manifestPath = path.join(directory, `update-${platform}.json`);
  try {
    assert.deepEqual(platformAssets(directory, config, platform), ['Arion Setup 0.1.0-alpha.6 amd64.exe', 'arion-0.1.0-alpha.6-windows-amd64.tar.gz', 'update-windows-amd64.json']);
    for (const changed of [{ size: manifest.size + 1 }, { signature: Buffer.alloc(64).toString('base64') }]) {
      writeFileSync(manifestPath, JSON.stringify({ ...manifest, ...changed }));
      assert.throws(() => platformAssets(directory, config, platform), /size or signature verification failed/);
    }
    writeFileSync(manifestPath, JSON.stringify({ ...manifest, version: '0.1.0-alpha.5' }));
    assert.throws(() => platformAssets(directory, config, platform), /Wrong update version/);
    writeFileSync(manifestPath, JSON.stringify({ ...manifest, url: manifest.url.replace('windows-amd64', 'darwin-arm64') }));
    assert.throws(() => platformAssets(directory, config, platform), /Wrong update archive URL/);
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
