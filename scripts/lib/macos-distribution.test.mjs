import test from 'node:test';
import assert from 'node:assert/strict';
import { generateKeyPairSync, sign } from 'node:crypto';
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { digest } from './release.mjs';
import { verifyMacOSDistribution } from './verify-macos-distribution.mjs';

const identity = 'Developer ID Application: Arion Test (ABCDE12345)';
const signature = `Authority=${identity}\nTeamIdentifier=ABCDE12345\nCodeDirectory v=20500 flags=0x10000(runtime)\nTimestamp=Oct 16 12:00:00\n`;
// MyGo's DMG name for Arion 0.1.0 on Apple silicon.
const dmgName = 'arion-0.1.0-arm64.dmg';

function packageDirectory(t, { dmg = dmgName, parent = os.tmpdir() } = {}) {
  const directory = mkdtempSync(path.join(parent, 'arion-distribution-test-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const helper = path.join(directory, 'Arion.app/Contents/Resources/llm/arion-llm');
  mkdirSync(path.dirname(helper), { recursive: true });
  writeFileSync(helper, 'helper test fixture');
  if (dmg) writeFileSync(path.join(directory, dmg), 'installer test fixture');
  return { directory, config: { name: 'Arion', version: '0.1.0' }, platform: 'darwin-arm64', binaries: [helper], identity, updates: false };
}

const signed = (command, args) => args.includes('--display') ? signature : '';

test('formal packages require a Developer ID, runtime, timestamp and valid tickets', t => {
  const distribution = packageDirectory(t);
  for (const invalid of [signature.replace('TeamIdentifier=ABCDE12345', 'TeamIdentifier=OTHER12345'), signature.replace(identity, 'Apple Development: Test'), signature.replace('(runtime)', ''), signature.replace('Timestamp=Oct 16 12:00:00', '')]) {
    assert.throws(() => verifyMacOSDistribution(distribution, (command, args) => args.includes('--display') ? invalid : ''), /signature|runtime|timestamp/);
  }
  assert.throws(() => verifyMacOSDistribution(distribution, (command, args) => {
    if (command === 'xcrun') throw new Error('invalid notarization ticket');
    return signed(command, args);
  }), /notarization ticket/);
});

test('a complete package checks the helper, App and the DMG MyGo builds without changing signatures', t => {
  const distribution = packageDirectory(t);
  const calls = [];
  verifyMacOSDistribution(distribution, (command, args) => {
    calls.push([command, args]);
    return signed(command, args);
  });
  assert.ok(calls.some(([command, args]) => command === 'codesign' && args.includes(distribution.binaries[0])));
  assert.ok(calls.some(([command, args]) => command === 'codesign' && args.at(-1) === path.join(distribution.directory, dmgName)));
  assert.equal(calls.filter(([command]) => command === 'xcrun').length, 2);
  assert.ok(calls.some(([command]) => command === 'spctl'));
  assert.ok(!calls.some(([, args]) => args.includes('--sign') || args.includes('staple')));
});

test('a missing DMG fails before codesign is asked to check it', t => {
  const distribution = packageDirectory(t, { dmg: 'Arion 0.1.0 arm64.dmg' });
  const checked = [];
  assert.throws(() => verifyMacOSDistribution(distribution, (command, args) => {
    if (command === 'codesign') checked.push(args.at(-1));
    return signed(command, args);
  }), new RegExp(`Missing signed package file: .*${dmgName}`));
  assert.ok(checked.includes(path.join(distribution.directory, 'Arion.app')));
  assert.ok(!checked.some(file => file.endsWith('.dmg')));
});

test('paths codesign deprecates are rejected instead of passed to codesign', t => {
  const parent = mkdtempSync(path.join(os.tmpdir(), 'arion:distribution-'));
  t.after(() => rmSync(parent, { recursive: true, force: true }));
  const distribution = packageDirectory(t, { parent });
  assert.throws(() => verifyMacOSDistribution(distribution, (command, args) => {
    if (command === 'codesign') assert.fail(`codesign received ${args.at(-1)}`);
    return signed(command, args);
  }), /path containing ':'/);
});

test('updates must retain their notarization ticket after archiving, and extraction is cleaned on failure', t => {
  const distribution = packageDirectory(t);
  const { directory } = distribution;
  const { privateKey, publicKey } = generateKeyPairSync('ed25519');
  const config = { name: 'Arion', version: '0.1.0', updates: { url: 'https://updates.example/v1/updates', publicKey: publicKey.export({ type: 'spki', format: 'der' }).subarray(-32).toString('base64') } };
  const archive = 'arion-0.1.0-darwin-arm64.tar.gz';
  // Archive extraction and Apple tools are mocked; signature verification is real.
  const bytes = Buffer.from('archive test fixture');
  writeFileSync(path.join(directory, archive), bytes);
  writeFileSync(path.join(directory, 'update-darwin-arm64.json'), JSON.stringify({
    version: '0.1.0', url: `https://updates.example/v1/updates/${archive}`,
    size: bytes.length, signature: sign(null, digest(bytes), privateKey).toString('base64'),
  }));
  let extracted;
  assert.throws(() => verifyMacOSDistribution({ ...distribution, config, updates: true }, (command, args) => {
    if (command === 'tar') {
      extracted = args.at(-1);
      mkdirSync(path.join(extracted, 'Arion.app'));
    }
    if (command === 'xcrun' && extracted && args.at(-1).startsWith(extracted)) throw new Error('update ticket missing');
    return signed(command, args);
  }), /update ticket missing/);
  assert.ok(extracted);
  assert.ok(!existsSync(extracted));
});
