// Static package validation is separate from actual-engine/native-host tests.
import { createHash } from 'node:crypto';
import { lstatSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { isDeepStrictEqual } from 'node:util';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyWindowsPEDependencies, windowsPEImports } from './windows-pe.mjs';
import { keyboardProfileFiles, englishBuildFiles, verifyEnglishProfile } from './keyboard-data.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const hash = bytes => createHash('sha256').update(bytes).digest('hex');
function safeMember(name) {
  if (typeof name !== 'string' || /^[\\/]/.test(name) || /[\\:\x00-\x1f"<>|?*]/.test(name) || name.split('/').some(part => !part || part === '.' || part === '..' || /[. ]$/.test(part))) throw new Error(`Unsafe Windows keyboard member: ${name}`);
  return name;
}
function fileIdentity(file) {
  const stat = lstatSync(file);
  if (!stat.isFile() || stat.size === 0) throw new Error(`Missing regular Windows keyboard file: ${file}`);
  const bytes = readFileSync(file);
  return { sizeBytes: bytes.length, sha256: hash(bytes) };
}
function requireHash(file, expected) {
  if (!expected || !Number.isSafeInteger(expected.sizeBytes) || expected.sizeBytes <= 0 || !/^[a-f0-9]{64}$/.test(expected.sha256 ?? '') || !isDeepStrictEqual(fileIdentity(file), { sizeBytes: expected.sizeBytes, sha256: expected.sha256 })) throw new Error(`Windows keyboard size or SHA-256 mismatch: ${file}`);
}

// The native-module build owns this addition; stage records only the engine.
export function recordWindowsKeyboardModule(directory) {
  const manifestPath = path.join(directory, 'runtime.json');
  const runtime = JSON.parse(readFileSync(manifestPath, 'utf8'));
  runtime.files['lib/ArionKeyboard.dll'] = fileIdentity(path.join(directory, 'lib/ArionKeyboard.dll'));
  writeFileSync(manifestPath, `${JSON.stringify(runtime, null, 2)}\n`);
}

export function verifyWindowsKeyboardRuntime(directory, { requireModule = true } = {}) {
  directory = path.resolve(directory);
  if (!lstatSync(directory).isDirectory() || lstatSync(directory).isSymbolicLink()) throw new Error('Windows keyboard root must be a regular directory.');
  const pins = JSON.parse(readFileSync(path.join(root, 'scripts/keyboard-runtime.json'), 'utf8'));
  const runtime = JSON.parse(readFileSync(path.join(directory, 'runtime.json'), 'utf8'));
  for (const [key, value] of Object.entries({ ...pins.engine, architectures: pins.windows.architectures, library: 'lib/ArionRime.dll' })) {
    if (!isDeepStrictEqual(runtime[key], value)) throw new Error(`Windows keyboard identity mismatch: ${key}`);
  }
  if (runtime.build?.compilerTarget !== 'x86_64-w64-mingw32' || runtime.build?.logging !== false || runtime.build?.dbVisibilityPatch !== 3 || runtime.build?.emojiVisibilityPatch !== 1 || runtime.build?.openccCxx17Patch !== 1) throw new Error('Windows keyboard private SDK ABI provenance is missing.');
  const sources = { rime: pins.archives.source, boost: pins.archives.boost, ...pins.windows.dependencies };
  if (!isDeepStrictEqual(runtime.build.sources, sources)) throw new Error('Windows keyboard source pins differ from reviewed dependencies.');
  if (!runtime.files || typeof runtime.files !== 'object' || Array.isArray(runtime.files)) throw new Error('Windows keyboard file manifest is missing.');
  const required = ['lib/ArionRime.dll', 'data/default.yaml', 'data/default.custom.yaml', 'data/key_bindings.yaml', 'data/punctuation.yaml', 'data/symbols.yaml',
    'data/pinyin_simp.schema.yaml', 'data/pinyin_simp.dict.yaml', 'data/arion_pinyin.schema.yaml', 'data/build/default.yaml',
    'data/build/arion_pinyin.schema.yaml', 'data/build/pinyin_simp.table.bin', 'data/build/pinyin_simp.prism.bin', 'data/build/pinyin_simp.reverse.bin',
    'licenses/keyboard-runtime.json', 'licenses/boost-LICENSE', 'licenses/darts-clone-COPYING', 'licenses/utf8cpp-notice.h', 'licenses/Arion-profile-README.md',
    ...pins.files.map(spec => spec.to), ...pins.windows.files.map(spec => spec.to),
    'data/arion_english.dict.yaml', ...keyboardProfileFiles.map(name => `data/${name}`),
    ...englishBuildFiles.map(name => `data/build/${name}`)];
  if (requireModule) required.push('lib/ArionKeyboard.dll');
  for (const member of required) if (!Object.hasOwn(runtime.files, member)) throw new Error(`Windows keyboard manifest lacks required member: ${member}`);
  const seen = new Set();
  const directories = new Set();
  for (const [member, identity] of Object.entries(runtime.files)) {
    safeMember(member);
    const folded = member.toLowerCase();
    if (seen.has(folded) || folded === 'runtime.json') throw new Error(`Duplicate Windows keyboard manifest member: ${member}`);
    seen.add(folded);
    const parts = member.split('/');
    for (let length=1;length<parts.length;length++) directories.add(parts.slice(0,length).join('/'));
    requireHash(path.join(directory, member), identity);
  }
  function walk(current) {
    for (const entry of readdirSync(current, { withFileTypes: true })) {
      const file = path.join(current, entry.name);
      const member = path.relative(directory, file).replaceAll('\\', '/');
      if (entry.isSymbolicLink()) throw new Error(`Windows keyboard symlink is forbidden: ${member}`);
      if (entry.isDirectory()) {
        if (!directories.has(member)) throw new Error(`Unrecorded Windows keyboard directory: ${member}`);
        if (['sdk', 'boost', 'compile-profile'].includes(entry.name)) throw new Error(`Build-only Windows keyboard SDK was packaged: ${member}`);
        walk(file);
      } else if (entry.isFile()) {
        if (/\.userdb|^(user|installation)\.yaml$/.test(entry.name)) throw new Error(`Mutable Windows keyboard profile was packaged: ${member}`);
        if (member !== 'runtime.json' && !Object.hasOwn(runtime.files, member)) throw new Error(`Unrecorded Windows keyboard member: ${member}`);
        if (member.startsWith('lib/') && !['lib/ArionRime.dll', 'lib/ArionKeyboard.dll'].includes(member)) throw new Error(`Unexpected Windows keyboard runtime library: ${member}`);
        if (member.startsWith('include/') && !/^include\/rime_(api|api_stdbool|api_deprecated|levers_api)\.h$/.test(member)) throw new Error(`Private Windows keyboard header was packaged: ${member}`);
      } else throw new Error(`Non-regular Windows keyboard member: ${member}`);
    }
  }
  walk(directory);
  for (const spec of [...pins.files, ...pins.windows.files]) requireHash(path.join(directory, safeMember(spec.to)), spec);
  const shippedPins = JSON.parse(readFileSync(path.join(directory, 'licenses/keyboard-runtime.json'), 'utf8'));
  if (!isDeepStrictEqual(shippedPins, pins)) throw new Error('Windows keyboard shipped pins differ from the reviewed manifest.');
  for (const name of keyboardProfileFiles) {
    if (hash(readFileSync(path.join(directory, 'data', name))) !== hash(readFileSync(path.join(root, 'apps/desktop/ime/shared/keyboard-data', name)))) throw new Error(`Windows keyboard profile differs from shared source: ${name}`);
  }
  const schema = readFileSync(path.join(directory, 'data/build/arion_pinyin.schema.yaml'), 'utf8');
  if (!schema.includes('schema_id: arion_pinyin') || !schema.includes('db_class: arion_encrypted_userdb') || !schema.includes('page_size: 9')) throw new Error('Windows keyboard compiled schema lost its identity, encrypted learning or page size.');
  verifyEnglishProfile(path.join(directory, 'data'));
  const binaries = [path.join(directory, 'lib/ArionRime.dll')];
  if (Object.hasOwn(runtime.files, 'lib/ArionKeyboard.dll')) {
    const module = path.join(directory, 'lib/ArionKeyboard.dll');
    if (!windowsPEImports(readFileSync(module)).includes('arionrime.dll')) throw new Error('Windows keyboard module must import the unique ArionRime.dll.');
    binaries.push(module);
  }
  const verified = verifyWindowsPEDependencies(binaries);
  return { engine: runtime.version, schema: runtime.schema, module: binaries.length === 2, files: Object.keys(runtime.files).length, verified };
}
