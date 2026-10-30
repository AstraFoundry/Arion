import test from 'node:test';
import assert from 'node:assert/strict';
import { developmentSigningIdentity, desktopProcesses, requireDesktopStopped } from './desktop-development.mjs';

const hash = 'A'.repeat(40);
const name = 'Apple Development: Developer (ABCDE12345)';
const inventory = `  1) ${hash} "${name}"\n     1 valid identities found\n`;
test('development requires a stable installed identity before invoking any signing tool', () => {
  for (const value of [undefined, '', '-', '   ']) {
    assert.throws(() => developmentSigningIdentity({ MACOS_DEV_SIGNING_IDENTITY: value }, () => assert.fail('invoked security')), /persistent/);
  }
  for (const value of [name, hash.toLowerCase()]) {
    assert.equal(developmentSigningIdentity({ MACOS_DEV_SIGNING_IDENTITY: value }, () => inventory), hash);
  }
  assert.throws(() => developmentSigningIdentity({ MACOS_DEV_SIGNING_IDENTITY: name }, () => ''), /exactly one/);
  assert.throws(() => developmentSigningIdentity({ MACOS_DEV_SIGNING_IDENTITY: name }, () => inventory + inventory), /exactly one/);
});
test('exclusive development identifies release and dev bundles across checkouts without matching helpers', () => {
  const processes = desktopProcesses(`
12 /Applications/Arion.app/Contents/MacOS/Arion
34 /Users/test/Library/Caches/Arion/desktop-dev/abc/darwin-arm64/Arion Dev.app/Contents/MacOS/Arion Dev
56 /Library/Input Methods/ArionInput.app/Contents/MacOS/ArionInput
78 /tmp/not-Arion.app/Contents/MacOS/Arion
90 /tmp/Arion Clipboard Check.app/Contents/MacOS/ArionClipboardCheck
`);
  assert.deepEqual(processes.map(p => [p.pid, p.development]), [[12, false], [34, true]]);
  assert.throws(() => requireDesktopStopped(processes), /12: \/Applications\/Arion.app/);
  assert.doesNotThrow(() => requireDesktopStopped([]));
});
