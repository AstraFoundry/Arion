// Builds Arion's InputMethodKit bundle (ArionInput.app) into the desktop
// macOS resources, where `mygo build` ships and signs it inside Arion.app.
import { execFileSync } from 'node:child_process';
import { copyFileSync, cpSync, mkdirSync, mkdtempSync, readFileSync, renameSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { macosSigningSettings } from './lib/desktop-signing.mjs';
import { imeEntitlementsPath, stageIMEProvisioning } from './lib/ime-provisioning.mjs';
import { buildIMELearning } from './lib/build-ime-learning.mjs';

if (process.platform !== 'darwin') {
  console.log('Input method bundle is macOS-only; skipped.');
  process.exit(0);
}
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = path.join(root, 'apps', 'desktop');
const config = JSON.parse(readFileSync(path.join(desktop, 'desktop.json'), 'utf8'));
const source = path.join(desktop, 'ime', 'macos');
const inputMetadata = JSON.parse(execFileSync('plutil', ['-convert', 'json', '-o', '-', path.join(source, 'Info.plist')], { encoding: 'utf8' }));
const inputModeIds = inputMetadata.ComponentInputModeDict.tsVisibleInputModeOrderedArrayKey;
if (inputModeIds.length !== 1) throw new Error('Expected exactly one keyboard input mode.');
const inputModeId = inputModeIds[0];
execFileSync(process.execPath, [path.join(root, 'scripts/stage-keyboard-runtime.mjs')], { cwd: root, stdio: 'inherit' });
const keyboard = path.join(desktop, 'resources/darwin/keyboard');
const output = path.join(desktop, 'resources', 'darwin');
mkdirSync(output, { recursive: true });
const destination = path.join(output, 'ArionInput.app');
const temporary = mkdtempSync(path.join(output, '.arion-input-build-'));
const stagedBundle = path.join(temporary, 'ArionInput.app');
const bundle = path.join(stagedBundle, 'Contents');
try {
  mkdirSync(path.join(bundle, 'MacOS'), { recursive: true });
  mkdirSync(path.join(bundle, 'Resources'), { recursive: true });
  copyFileSync(path.join(source, 'Info.plist'), path.join(bundle, 'Info.plist'));
  const run = (command, args, cwd = desktop) => execFileSync(command, args, { cwd, stdio: 'inherit' });
  run('/usr/libexec/PlistBuddy', ['-c', `Set :CFBundleShortVersionString ${config.version.split('-')[0]}`, path.join(bundle, 'Info.plist')]);
  run('/usr/libexec/PlistBuddy', ['-c', `Set :CFBundleVersion ${config.version.split('-')[0]}`, path.join(bundle, 'Info.plist')]);
  run('/usr/libexec/PlistBuddy', ['-c', `Set :LSMinimumSystemVersion ${config.macos.minimumSystemVersion}`, path.join(bundle, 'Info.plist')]);
  // Compile every native source and the CryptoKit boundary for each architecture.
  const objects = path.join(root, '.workspace', 'keyboard-runtime', 'host-build');
  const slices = [];
  for (const arch of ['arm64', 'x86_64']) {
    const target = `${arch}-apple-macos${config.macos.minimumSystemVersion}`;
    const output = path.join(objects, arch);
    mkdirSync(output, { recursive: true });
    const native = [];
    for (const name of ['main', 'candidate_panel', 'composition', 'keyboard', 'ranking', 'typing']) {
      const object = path.join(output, `${name}.o`);
      run('xcrun', ['clang', '-fobjc-arc', '-O2', '-target', target, '-I', path.join(keyboard, 'include'),
        '-c', path.join(source, `${name}.m`), '-o', object]);
      native.push(object);
    }
    const learning = buildIMELearning({ root, outputDir: output, arch, minimumSystemVersion: config.macos.minimumSystemVersion });
    const executable = path.join(output, 'ArionInput');
    run('xcrun', [learning.linker, '-target', target, ...native, ...learning.objects,
      path.join(keyboard, 'lib/librime.1.dylib'), ...learning.linkArgs, '-framework', 'Cocoa', '-framework', 'QuartzCore', '-framework', 'InputMethodKit',
      '-Xlinker', '-rpath', '-Xlinker', '@executable_path/../Frameworks', '-o', executable]);
    slices.push(executable);
  }
  run('lipo', ['-create', ...slices, '-output', path.join(bundle, 'MacOS', 'ArionInput')]);
  mkdirSync(path.join(bundle, 'Frameworks'), { recursive: true });
  copyFileSync(path.join(keyboard, 'lib/librime.1.dylib'), path.join(bundle, 'Frameworks/librime.1.dylib'));
  for (const directory of ['data', 'licenses']) cpSync(path.join(keyboard, directory), path.join(bundle, 'Resources/keyboard', directory), { recursive: true });
  copyFileSync(path.join(keyboard, 'runtime.json'), path.join(bundle, 'Resources/keyboard/runtime.json'));
  for (const [locale, language] of [['en', 'en'], ['zh-CN', 'zh-Hans']]) {
    const catalog = JSON.parse(readFileSync(path.join(desktop, 'ui/shared/copy/locales', `${locale}.json`), 'utf8'));
    const strings = ['KeyboardModeChinese', 'KeyboardModeEnglish', 'KeyboardSwitchChinese', 'KeyboardSwitchEnglish', 'KeyboardUnavailable', 'KeyboardResetLearning', 'KeyboardCopyPending'].map(key => {
      if (!catalog[key]) throw new Error(`Missing keyboard copy: ${locale}/${key}`);
      return `${JSON.stringify(key)} = ${JSON.stringify(catalog[key])};`;
    });
    const directory = path.join(bundle, 'Resources', `${language}.lproj`);
    mkdirSync(directory, { recursive: true });
    writeFileSync(path.join(directory, 'Localizable.strings'), `${strings.join('\n')}\n`);
    // TIS resolves the input mode's display name through InfoPlist.strings,
    // keyed by its complete mode ID; Localizable.strings serves the runtime menu.
    const identity = ['CFBundleName', 'CFBundleDisplayName', inputModeId]
      .map(key => `${JSON.stringify(key)} = ${JSON.stringify(config.name)};`);
    writeFileSync(path.join(directory, 'InfoPlist.strings'), `${identity.join('\n')}\n`);
  }
  // The input menu sizes the icon from its point size, so it must be a 16 pt
  // image: a 16 px 1x slice plus a 32 px 2x slice, as the system sources ship.
  const icons = mkdtempSync(path.join(temporary, 'menu-icon-'));
  const iconSlices = [[16, 'menu_icon.png'], [32, 'menu_icon@2x.png']].map(([pixels, name]) => {
    const png = path.join(icons, name);
    run('go', ['run', './cmd/arion-icon', '-menu', String(pixels), png]);
    const tiff = png.replace(/\.png$/, '.tiff');
    run('sips', ['-s', 'format', 'tiff', png, '--out', tiff]);
    return tiff;
  });
  run('tiffutil', ['-cathidpicheck', ...iconSlices, '-out', path.join(bundle, 'Resources', 'menu_icon.tiff')]);
  const signing = process.env.ARION_IME_SIGNED_BUILD === '1' ? macosSigningSettings(process.env) : null;
  if (signing) stageIMEProvisioning({ profilePath: signing.imeProvisioningProfile, identity: signing.identity,
    bundleIdentifier: inputMetadata.CFBundleIdentifier, bundle: stagedBundle, entitlementsFile: imeEntitlementsPath(root) });
  // The visible app remains complete throughout compilation. Publish only the
  // finished bundle with a same-filesystem rename, so LaunchServices cannot index
  // an Info.plist whose executable has not yet been linked.
  rmSync(destination, { recursive: true, force: true });
  renameSync(stagedBundle, destination);
  console.log(`Built ${path.relative(root, destination)}`);
} finally {
  rmSync(temporary, { recursive: true, force: true });
}
