// Checks the packaged app that `mygo build` signed: no build-machine rpath
// ships in the binary (scripts/cc-relocatable.sh drops them at link time),
// and the bundle's signature is valid. It changes nothing, so the app, its
// disk image and its signed update archive stay the same build.
import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') process.exit(0);
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const arch = process.arch === 'arm64' ? 'arm64' : 'amd64';
const app = path.join(root, 'apps', 'desktop', 'build', `darwin-${arch}`, 'Arion.app');
const binary = path.join(app, 'Contents', 'MacOS', 'Arion');
const run = (command, args) => execFileSync(command, args, { encoding: 'utf8' });

const rpaths = [...run('otool', ['-l', binary]).matchAll(/cmd LC_RPATH[\s\S]*?path (.+?) \(offset/g)].map(m => m[1]);
const absolute = rpaths.filter(rpath => !rpath.startsWith('@'));
if (absolute.length) throw new Error(`build-machine rpath ships in ${path.relative(root, binary)}: ${absolute.join(', ')}`);
run('codesign', ['--verify', '--deep', '--strict', app]);
console.log(`Verified ${path.relative(root, app)} (rpaths: ${rpaths.join(', ')})`);
