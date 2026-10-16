import test from 'node:test';
import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { existsSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { digest } from '../../../scripts/lib/release.mjs';
import { verifyMacOSDistribution } from '../../../scripts/lib/verify-macos-distribution.mjs';

const identity = 'Developer ID Application: Arion Test (ABCDE12345)';
const signature = `Authority=${identity}\nTeamIdentifier=ABCDE12345\nCodeDirectory v=20500 flags=0x10000(runtime)\nTimestamp=Oct 16 12:00:00\n`;
const distribution = {
  directory: '/package/darwin-arm64', config: { name: 'Arion', version: '0.1.0' }, platform: 'darwin-arm64',
  binaries: ['/package/darwin-arm64/Arion.app/Contents/Resources/llm/arion-llm'], identity, updates: false,
};

test('formal packages require a Developer ID, runtime, timestamp and valid tickets', () => {
  for (const invalid of [signature.replace('TeamIdentifier=ABCDE12345', 'TeamIdentifier=OTHER12345'), signature.replace(identity, 'Apple Development: Test'), signature.replace('(runtime)', ''), signature.replace('Timestamp=Oct 16 12:00:00', '')]) {
    assert.throws(() => verifyMacOSDistribution(distribution, (command, args) => args.includes('--display') ? invalid : ''), /signature|runtime|timestamp/);
  }
  assert.throws(() => verifyMacOSDistribution(distribution, (command, args) => {
    if (command === 'xcrun') throw new Error('invalid notarization ticket');
    return args.includes('--display') ? signature : '';
  }), /notarization ticket/);
});

test('a complete package checks the helper, App and DMG without changing signatures', () => {
  const calls = [];
  verifyMacOSDistribution(distribution, (command, args) => {
    calls.push([command, args]);
    return args.includes('--display') ? signature : '';
  });
  assert.ok(calls.some(([command, args]) => command === 'codesign' && args.includes(distribution.binaries[0])));
  assert.equal(calls.filter(([command]) => command === 'xcrun').length, 2);
  assert.ok(calls.some(([command]) => command === 'spctl'));
  assert.ok(!calls.some(([, args]) => args.includes('--sign') || args.includes('staple')));
});

test('updates must retain their notarization ticket after archiving, and extraction is cleaned on failure', t => {
  const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-distribution-test-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const { privateKey, publicKey } = generateKeyPairSync('ed25519');
  const config = { name: 'Arion', version: '0.1.0', updates: { github: 'example/arion', publicKey: publicKey.export({ type: 'spki', format: 'der' }).subarray(-32).toString('base64') } };
  const archive = 'arion-0.1.0-darwin-arm64.tar.gz';
  // Archive extraction and Apple tools are mocked; signature verification is real.
  const bytes = Buffer.from('archive test fixture');
  writeFileSync(path.join(directory, archive), bytes);
  writeFileSync(path.join(directory, 'Arion 0.1.0 arm64.dmg'), 'installer test fixture');
  writeFileSync(path.join(directory, 'update-darwin-arm64.json'), JSON.stringify({
    version: '0.1.0', url: `https://github.com/example/arion/releases/download/v0.1.0/${archive}`,
    size: bytes.length, signature: sign(null, digest(bytes), privateKey).toString('base64'),
  }));
  let extracted;
  assert.throws(() => verifyMacOSDistribution({ ...distribution, directory, config, updates: true }, (command, args) => {
    if (command === 'tar') extracted = args.at(-1);
    if (command === 'xcrun' && extracted && args.at(-1).startsWith(extracted)) throw new Error('update ticket missing');
    return args.includes('--display') ? signature : '';
  }), /update ticket missing/);
  assert.ok(extracted);
  assert.ok(!existsSync(extracted));
});
