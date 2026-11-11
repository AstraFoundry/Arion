import { existsSync, mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { signingRunner } from './desktop-signing.mjs';
import { installerName, platformAssets } from './release.mjs';

// codesign deprecates ':' in paths and reports a missing file only as a failed check.
function codesign(run, args) {
  const file = args.at(-1);
  if (file.includes(':')) throw new Error(`codesign cannot verify a path containing ':': ${file}`);
  if (!existsSync(file)) throw new Error(`Missing signed package file: ${file}`);
  return run('codesign', args);
}

function verifySignedCode(file, identity, run) {
  codesign(run, ['--verify', '--strict', file]);
  const signature = codesign(run, ['--display', '--verbose=4', file]);
  const team = identity.match(/\(([A-Z0-9]{10})\)$/)[1];
  if (!signature.split('\n').includes(`Authority=${identity}`) || !signature.split('\n').includes(`TeamIdentifier=${team}`)) {
    throw new Error(`Unexpected Developer ID signature: ${file}`);
  }
  if (!/flags=.*\bruntime\b/.test(signature) || !/^Timestamp=.+/m.test(signature)) {
    throw new Error(`Missing hardened runtime or secure timestamp: ${file}`);
  }
}

export function verifyMacOSDistribution({ directory, config, platform, binaries, identity, updates }, run = signingRunner()) {
  const app = path.join(directory, `${config.name}.app`);
  const dmg = path.join(directory, installerName(config, platform));
  for (const file of [app, ...binaries]) verifySignedCode(file, identity, run);
  codesign(run, ['--verify', '--deep', '--strict', app]);
  codesign(run, ['--verify', '--strict', dmg]);
  for (const file of [app, dmg]) run('xcrun', ['stapler', 'validate', file]);
  run('spctl', ['--assess', '--type', 'execute', '--verbose=4', app]);
  if (!updates) return;
  const files = platformAssets(directory, config, platform);
  const archive = files.find(file => file.endsWith('.tar.gz'));
  const unpacked = mkdtempSync(path.join(os.tmpdir(), 'arion-update-verification-'));
  try {
    run('tar', ['-xzf', path.join(directory, archive), '-C', unpacked]);
    const updated = path.join(unpacked, `${config.name}.app`);
    verifySignedCode(updated, identity, run);
    codesign(run, ['--verify', '--deep', '--strict', updated]);
    run('xcrun', ['stapler', 'validate', updated]);
  } finally {
    rmSync(unpacked, { recursive: true, force: true });
  }
}
