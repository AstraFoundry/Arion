// Verify the nested IMK bundle without launching it or mutating signed files.
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, lstatSync, readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { isDeepStrictEqual } from 'node:util';
import { keyboardProfileFiles, englishLexiconFile, verifyEnglishProfile } from './lib/keyboard-data.mjs';

const repository = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const digest = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const run = (command, args) => execFileSync(command, args, { encoding: 'utf8', maxBuffer: 16 * 1024 * 1024 }).trim();
const plist = file => JSON.parse(run('plutil', ['-convert', 'json', '-o', '-', file]));
const requireFile = file => {
  const stat = lstatSync(file);
  if (!stat.isFile() || stat.size === 0) throw new Error(`Missing, linked or empty keyboard package file: ${file}`);
  return stat;
};
const numericVersion = value => value.split('.').reduce((result, component, index) => result + Number(component) * [1000000, 1000, 1][index], 0);
const requireHash = (file, expected) => {
  const stat = requireFile(file);
  if (stat.size !== expected.sizeBytes || digest(file) !== expected.sha256) throw new Error(`Keyboard package hash mismatch: ${file}`);
};
const safeMember = name => {
  if (path.isAbsolute(name) || name.includes('\\') || name.split('/').some(part => !part || part === '..' || part === '.')) throw new Error(`Unsafe keyboard manifest member: ${name}`);
  return name;
};

export function verifyInputMethodIdentity({ bundleId, sourceId, adapter, identity }) {
  const sharedBundle = identity.match(/^\s*BundleID\s*=\s*"([^"]+)"\s*$/m)?.[1];
  const suffix = identity.match(/^\s*SourceID\s*=\s*BundleID\s*\+\s*"([^"]+)"\s*$/m)?.[1];
  if (sharedBundle !== bundleId || suffix !== '.pinyin' || sourceId !== sharedBundle + suffix ||
      !adapter.includes('"arion.local/desktop/infrastructure/imebridge"') ||
      !/^const IMESourceID = imebridge\.SourceID$/m.test(adapter) ||
      !/^const IMEBundleID = imebridge\.BundleID$/m.test(adapter)) {
    throw new Error('Input mode identity must match the shared bridge identity, registration adapter and parent bundle.');
  }
}

export function verifyKeyboardBundle({ root = repository, bundle = path.join(root, 'apps/desktop/resources/darwin/ArionInput.app') } = {}) {
  if (process.platform !== 'darwin') throw new Error('Keyboard bundle verification requires macOS.');
  bundle = path.resolve(bundle);
  const config = JSON.parse(readFileSync(path.join(root, 'apps/desktop/desktop.json'), 'utf8'));
  const pins = JSON.parse(readFileSync(path.join(root, 'scripts/keyboard-runtime.json'), 'utf8'));
  const source = path.join(root, 'apps/desktop/ime/macos');
  const expectedPlist = plist(path.join(source, 'Info.plist'));
  const contents = path.join(bundle, 'Contents');
  const actualPlist = plist(path.join(contents, 'Info.plist'));
  const version = config.version.split('-')[0];
  const expected = {
    CFBundleIdentifier: expectedPlist.CFBundleIdentifier, CFBundleName: config.name,
    CFBundleDisplayName: config.name, CFBundleShortVersionString: version, CFBundleVersion: version,
    LSMinimumSystemVersion: config.macos.minimumSystemVersion,
  };
  for (const key of ['CFBundleExecutable', 'CFBundlePackageType', 'CFBundleDevelopmentRegion', 'CFBundleLocalizations',
    'LSBackgroundOnly', 'NSPrincipalClass', 'InputMethodConnectionName', 'InputMethodServerControllerClass',
    'InputMethodServerDelegateClass', 'tsInputMethodCharacterRepertoireKey', 'ComponentInputModeDict', 'tsInputMethodIconFileKey']) expected[key] = expectedPlist[key];
  for (const [key, value] of Object.entries(expected)) if (!isDeepStrictEqual(actualPlist[key], value)) throw new Error(`Keyboard bundle identity/configuration mismatch: ${key}`);
  if (!actualPlist.CFBundleIdentifier.includes('.inputmethod.')) throw new Error('Input-method bundle identifier must contain .inputmethod.');
  const sourceIds = expectedPlist.ComponentInputModeDict.tsVisibleInputModeOrderedArrayKey;
  if (sourceIds.length !== 1) throw new Error('Expected one full-Pinyin input mode.');
  const sourceId = sourceIds[0];
  const adapter = readFileSync(path.join(root, 'apps/desktop/infrastructure/macos/ime.go'), 'utf8');
  const identity = readFileSync(path.join(root, 'apps/desktop/infrastructure/imebridge/identity.go'), 'utf8');
  verifyInputMethodIdentity({ bundleId: actualPlist.CFBundleIdentifier, sourceId, adapter, identity });
  const mode = actualPlist.ComponentInputModeDict?.tsInputModeListKey?.[sourceId];
  if (mode?.TISInputSourceID !== sourceId || mode.TISIntendedLanguage !== 'zh-Hans' || !mode.tsInputModeIsVisibleKey ||
      !isDeepStrictEqual(actualPlist.ComponentInputModeDict.tsVisibleInputModeOrderedArrayKey, [sourceId])) throw new Error('Keyboard input-source identity is invalid.');
  const resources = path.join(contents, 'Resources');
  requireFile(path.join(resources, actualPlist.tsInputMethodIconFileKey));
  // The native menu selects these centralized catalog keys dynamically.
  const nativeCopyKeys = new Set(readdirSync(source).filter(name => /\.(m|mm)$/.test(name)).flatMap(name => [...readFileSync(path.join(source, name), 'utf8').matchAll(/@"(Keyboard[A-Za-z0-9]+)"/g)].map(match => match[1])));
  if (!nativeCopyKeys.size) throw new Error('Native keyboard copy keys were not found.');
  for (const [locale, language] of [['en', 'en'], ['zh-CN', 'zh-Hans']]) {
    const file = path.join(resources, `${language}.lproj/Localizable.strings`);
    requireFile(file);
    const strings = plist(file);
    const catalog = JSON.parse(readFileSync(path.join(root, 'apps/desktop/ui/shared/copy/locales', `${locale}.json`), 'utf8'));
    for (const key of nativeCopyKeys) if (typeof strings[key] !== 'string' || !strings[key] || strings[key] !== catalog[key]) throw new Error(`Keyboard localization mismatch: ${language}/${key}`);
    const identityFile = path.join(resources, `${language}.lproj/InfoPlist.strings`);
    requireFile(identityFile);
    const identity = plist(identityFile);
    for (const key of ['CFBundleName', 'CFBundleDisplayName', sourceId]) {
      if (identity[key] !== config.name) throw new Error(`Keyboard input-source localization mismatch: ${language}/${key}`);
    }
  }

  const keyboard = path.join(resources, 'keyboard');
  const runtime = JSON.parse(readFileSync(path.join(keyboard, 'runtime.json'), 'utf8'));
  for (const [key, value] of Object.entries(pins.engine)) if (!isDeepStrictEqual(runtime[key], value)) throw new Error(`Keyboard runtime identity mismatch: ${key}`);
  if (!isDeepStrictEqual(runtime.build?.correctionIdentity,pins.patches.correctionIdentity)) throw new Error('Keyboard correction identity patch provenance is missing.');
  const expectedSources=Object.fromEntries(['source','dependencies','boost'].map(key=>[key,pins.archives[key]]));
  if (!isDeepStrictEqual(runtime.build?.sources,expectedSources)) throw new Error('Keyboard source build pins differ.');
  if (runtime.build?.builderSha256!==digest(path.join(root,'scripts/lib/keyboard-macos-runtime.mjs')) || runtime.build?.patcherSha256!==digest(path.join(root,'scripts/lib/keyboard-correction-patch.mjs'))) throw new Error('Keyboard source build recipe differs.');
  const {fingerprint,outputs,...buildIdentity}=runtime.build;
  if (fingerprint!==createHash('sha256').update(JSON.stringify(buildIdentity)).digest('hex') ||
      !isDeepStrictEqual(buildIdentity.architectures,[...pins.engine.architectures].sort()) ||
      buildIdentity.minimumSystemVersion!==config.macos.minimumSystemVersion ||
      buildIdentity.logging!==true || buildIdentity.externalPlugins!==false)
    throw new Error('Keyboard source build configuration or fingerprint differs.');
  const libraryOutput=`build/lib/librime.${pins.engine.version}.dylib`;
  if (!outputs || !isDeepStrictEqual(Object.keys(outputs).sort(),[libraryOutput,'build/bin/rime_deployer'].sort()) ||
      !isDeepStrictEqual(outputs[libraryOutput],runtime.files?.['lib/librime.1.dylib']))
    throw new Error('Keyboard source build output inventory differs.');
  if (!runtime.files || typeof runtime.files !== 'object' || Array.isArray(runtime.files)) throw new Error('Keyboard runtime hash manifest is missing.');
  const required = ['data/default.yaml', 'data/default.custom.yaml', 'data/key_bindings.yaml', 'data/punctuation.yaml', 'data/symbols.yaml',
    'data/pinyin_simp.schema.yaml', 'data/pinyin_simp.dict.yaml', 'data/arion_pinyin.schema.yaml',
    'data/build/default.yaml', 'data/build/arion_pinyin.schema.yaml', 'data/build/pinyin_simp.table.bin',
    'data/build/pinyin_simp.prism.bin', 'data/build/pinyin_simp.reverse.bin', 'licenses/keyboard-runtime.json',
    'licenses/boost-LICENSE', 'licenses/darts-clone-COPYING', 'licenses/utf8cpp-notice.h', 'licenses/Arion-profile-README.md',
    `data/${englishLexiconFile}`, ...keyboardProfileFiles.map(name => `data/${name}`)];
  for (const name of required) if (!Object.hasOwn(runtime.files, name)) throw new Error(`Keyboard runtime manifest lacks required member: ${name}`);
  for (const [name, expectedFile] of Object.entries(runtime.files)) {
    safeMember(name);
    if (name.startsWith('data/') || name.startsWith('licenses/')) requireHash(path.join(keyboard, name), expectedFile);
  }
  // Check upstream assets against the reviewed pins as well as the shipped ledger.
  for (const spec of pins.files) requireHash(path.join(keyboard, safeMember(spec.to)), spec);
  const shippedPins = JSON.parse(readFileSync(path.join(keyboard, 'licenses/keyboard-runtime.json'), 'utf8'));
  if (!isDeepStrictEqual(shippedPins, pins)) throw new Error('Keyboard source/download pins differ from the reviewed manifest.');
  for (const name of keyboardProfileFiles) if (digest(path.join(keyboard, 'data', name)) !== digest(path.join(source, '../shared/keyboard-data', name))) throw new Error(`Keyboard profile differs from reviewed source: ${name}`);
  const schema = readFileSync(path.join(keyboard, 'data/build/arion_pinyin.schema.yaml'), 'utf8');
  if (!schema.includes('schema_id: arion_pinyin') || !schema.includes('db_class: arion_encrypted_userdb') || !schema.includes('page_size: 9')) throw new Error('Keyboard compiled schema identity, encrypted learning or page size is invalid.');
  verifyEnglishProfile(path.join(keyboard, 'data'));
  const walk = directory => {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const file = path.join(directory, entry.name);
      if (entry.isSymbolicLink()) throw new Error(`Unexpected keyboard package symlink: ${file}`);
      if (entry.isDirectory()) {
        if (['sdk', 'boost', 'include', 'compile-profile'].includes(entry.name)) throw new Error(`Build-only keyboard SDK was packaged: ${file}`);
        walk(file);
      } else if (entry.isFile()) {
        if (/\.userdb|^(user|installation)\.yaml$/.test(entry.name)) throw new Error(`Mutable keyboard profile was packaged: ${file}`);
        const member = path.relative(keyboard, file).split(path.sep).join('/');
        if (member !== 'runtime.json' && !Object.hasOwn(runtime.files, member)) throw new Error(`Unrecorded keyboard package file: ${member}`);
      }
    }
  };
  walk(keyboard);

  const executable = path.join(contents, 'MacOS', actualPlist.CFBundleExecutable);
  const library = path.join(contents, 'Frameworks/librime.1.dylib');
  for (const [file, host] of [[executable, true], [library, false]]) {
    requireFile(file);
    const architectures = run('lipo', ['-archs', file]).split(/\s+/).sort();
    if (!isDeepStrictEqual(architectures, [...pins.engine.architectures].sort())) throw new Error(`Keyboard binary is not universal arm64 + x86_64: ${file}`);
    for (const arch of architectures) {
      const commands = run('otool', ['-arch', arch, '-l', file]);
      const minimums = [...commands.matchAll(/\bminos ([\d.]+)|cmd LC_VERSION_MIN_MACOSX[\s\S]*?\bversion ([\d.]+)/g)].map(match => match[1] ?? match[2]);
      if (!minimums.length || minimums.some(value => numericVersion(value) > numericVersion(config.macos.minimumSystemVersion))) throw new Error(`Keyboard binary minimum OS exceeds ${config.macos.minimumSystemVersion}: ${file}/${arch}`);
      const linkage = run('otool', ['-arch', arch, '-L', file]);
      const dependencies = [...linkage.matchAll(/^\s+(\S+) \(compatibility version [^,]+, current version ([^)]+)\)/gm)];
      if (!dependencies.length || dependencies.some(([, dependency]) => dependency !== '@rpath/librime.1.dylib' && !dependency.startsWith('/usr/lib/') && !dependency.startsWith('/System/Library/Frameworks/'))) throw new Error(`Nonportable keyboard library load path: ${file}/${arch}`);
      const engine = dependencies.find(([, dependency]) => dependency === '@rpath/librime.1.dylib');
      if (!engine || engine[2] !== pins.engine.version) throw new Error(`Keyboard engine link version mismatch: ${file}/${arch}`);
      const rpaths = [...commands.matchAll(/cmd LC_RPATH[\s\S]*?\bpath (\S+) \(offset/g)].map(match => match[1]);
      if (rpaths.some(value => !['@executable_path/../Frameworks', '/usr/lib/swift'].includes(value)) || (host && !rpaths.includes('@executable_path/../Frameworks'))) throw new Error(`Keyboard binary has an invalid runtime search path: ${file}/${arch}`);
      if (!host && !/^_rime_get_api$/m.test(run('nm', ['-arch', arch, '-gUj', file]))) throw new Error(`Keyboard engine C API is missing: ${arch}`);
    }
  }
  // codesign changes Mach-O bytes. Immutable data/licenses retain exact hashes;
  // the dylib is identified by its pinned source build and inspected ABI here.
  const signature = spawnSync('codesign', ['--display', '--verbose=2', bundle], { encoding: 'utf8' });
  const signatureText = `${signature.stdout ?? ''}${signature.stderr ?? ''}`;
  const sealed = existsSync(path.join(contents, '_CodeSignature/CodeResources')) || (signature.status === 0 && !signatureText.includes('Info.plist=not bound'));
  if (sealed) {
    run('codesign', ['--verify', '--strict', library]);
    run('codesign', ['--verify', '--strict', '--deep', bundle]);
  }
  return { bundle, schema: pins.engine.schema, engine: pins.engine.version, architectures: pins.engine.architectures, signature: sealed ? 'sealed-verified' : 'unsealed-development' };
}

if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  const args = process.argv.slice(2);
  if (args.length && (args.length !== 2 || args[0] !== '--bundle')) throw new Error('usage: verify-keyboard-bundle.mjs [--bundle /absolute/path/ArionInput.app]');
  const result = verifyKeyboardBundle(args.length ? { bundle: args[1] } : {});
  console.log(`Verified ${result.schema}, librime ${result.engine}, universal IMK identity/data/licenses/localizations (${result.signature}).`);
}
