// Removes build-machine rpaths (the sherpa-onnx package links against the Go
// module cache) from the packaged app, re-signs it with its entitlements and
// creates the disk image, so no local user path ships in the binary.
import { execFileSync } from 'node:child_process';
import { readFileSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') process.exit(0);
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
const out = path.join(desktop, 'build', `darwin-${arch}`);
const app = path.join(out, 'Arion.app');
const binary = path.join(app, 'Contents', 'MacOS', 'Arion');
const run = (command, args) => execFileSync(command, args, { encoding: 'utf8' });

const loadCommands = run('otool', ['-l', binary]);
const rpaths = [...loadCommands.matchAll(/cmd LC_RPATH[\s\S]*?path (.+?) \(offset/g)].map(m => m[1]);
for (const rpath of rpaths) {
  if (!rpath.startsWith('@')) run('install_name_tool', ['-delete_rpath', rpath, binary]);
}
const remaining = [...run('otool', ['-l', binary]).matchAll(/path (.+?) \(offset/g)].map(m => m[1]);
if (remaining.some(rpath => !rpath.startsWith('@'))) throw new Error(`absolute rpath remains: ${remaining}`);

const config = JSON.parse(readFileSync(path.join(desktop, 'mygo.json'), 'utf8'));
run('codesign', ['--force', '--sign', config.macos?.signingIdentity ?? '-', '--entitlements', path.join(desktop, config.macos.entitlements), app]);
run('codesign', ['--verify', '--deep', '--strict', app]);
const dmg = path.join(out, `Arion ${config.version} ${arch}.dmg`);
rmSync(dmg, { force: true });
run('hdiutil', ['create', '-quiet', '-volname', 'Arion', '-srcfolder', app, '-ov', '-format', 'UDZO', dmg]);
console.log(`Finalized ${path.relative(root, app)} (rpaths: ${remaining.join(', ')}) and ${path.relative(root, dmg)}`);
