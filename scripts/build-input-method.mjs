// Builds Arion's InputMethodKit bundle (ArionInput.app) into the desktop
// macOS resources, where `mygo build` ships and signs it inside Arion.app.
import { execFileSync } from 'node:child_process';
import { copyFileSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') {
  console.log('Input method bundle is macOS-only; skipped.');
  process.exit(0);
}
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const source = path.join(desktop, 'ime', 'macos');
const bundle = path.join(desktop, 'resources', 'darwin', 'ArionInput.app', 'Contents');
rmSync(path.dirname(bundle), { recursive: true, force: true });
mkdirSync(path.join(bundle, 'MacOS'), { recursive: true });
mkdirSync(path.join(bundle, 'Resources'), { recursive: true });
copyFileSync(path.join(source, 'Info.plist'), path.join(bundle, 'Info.plist'));
const run = (command, args, cwd = desktop) => execFileSync(command, args, { cwd, stdio: 'inherit' });
// Universal so one bundle serves Apple silicon and Intel Macs.
run('clang', ['-fobjc-arc', '-O2', '-mmacosx-version-min=13.0', '-arch', 'arm64', '-arch', 'x86_64',
  '-framework', 'Cocoa', '-framework', 'InputMethodKit', path.join(source, 'main.m'), '-o', path.join(bundle, 'MacOS', 'ArionInput')]);
const png = path.join(bundle, 'Resources', 'menu_icon.png');
run('go', ['run', './cmd/arion-icon', '-menu', png]);
run('sips', ['-s', 'format', 'tiff', png, '--out', path.join(bundle, 'Resources', 'menu_icon.tiff')]);
rmSync(png);
console.log(`Built ${path.relative(root, path.dirname(bundle))}`);
