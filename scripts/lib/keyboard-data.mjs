// Both native hosts consume the same pinned dictionary and compiled schema.
import { copyFileSync, existsSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, utimesSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { pinnedFile } from './pinned-file.mjs';
import { writeEnglishDictionary } from './keyboard-english.mjs';

export const keyboardProfileFiles = ['arion_pinyin.schema.yaml', 'arion_english.schema.yaml', 'default.custom.yaml'];
export const englishBuildFiles = ['arion_english.schema.yaml', 'arion_english.table.bin', 'arion_english.prism.bin', 'arion_english.reverse.bin'];

export function verifyEnglishProfile(data) {
  const schema = readFileSync(path.join(data, 'build/arion_pinyin.schema.yaml'), 'utf8');
  const english = schema.match(/^english:\n((?:  [^\n]*(?:\n|$))+)/m)?.[1];
  if (!schema.includes('table_translator@english') || !schema.includes('- arion_english') ||
      !english || !/^  dictionary: arion_english$/m.test(english) ||
      !/^  enable_user_dict: false$/m.test(english) || !/^  enable_sentence: false$/m.test(english) ||
      !/^  enable_completion: false$/m.test(english) || !/^  initial_quality: 10$/m.test(english)) {
    throw new Error('Compiled English candidates lost their exact-match, ordering or read-only policy.');
  }
}

export function stageKeyboardData({ manifest, manifestPath, profile, staging, work, deploy }) {
  const copy = (from, to) => { mkdirSync(path.dirname(to), { recursive: true }); copyFileSync(from, to); };
  for (const spec of manifest.files) copy(pinnedFile(spec), path.join(staging, spec.to));
  for (const name of keyboardProfileFiles) copy(path.join(profile, name), path.join(staging, 'data', name));
  writeEnglishDictionary(staging, manifest.english);
  copy(path.join(profile, 'README.md'), path.join(staging, 'licenses/Arion-profile-README.md'));
  copy(manifestPath, path.join(staging, 'licenses/keyboard-runtime.json'));
  const data = path.join(staging, 'data');
  const build = path.join(data, 'build');
  const user = path.join(staging, 'compile-profile');
  mkdirSync(user, { recursive: true });
  mkdirSync(build, { recursive: true });
  // Rime embeds source modification times in its compiler metadata.
  for (const name of readdirSync(data)) if (statSync(path.join(data, name)).isFile()) utimesSync(path.join(data, name), 1, 1);
  try { deploy(['--build', user, data, build]); }
  catch (error) {
    const log = path.join(work, 'deployment-failure.log');
    writeFileSync(log, error.stderr ?? error.message);
    throw new Error(`Keyboard schema deployment failed; inspect ${log}`, { cause: error });
  }
  for (const name of ['default.yaml', 'arion_pinyin.schema.yaml', 'pinyin_simp.table.bin', 'pinyin_simp.reverse.bin', 'pinyin_simp.prism.bin', ...englishBuildFiles]) {
    if (!existsSync(path.join(build, name))) throw new Error(`Keyboard prebuild is incomplete: ${name}`);
  }
  const schema = readFileSync(path.join(build, 'arion_pinyin.schema.yaml'), 'utf8');
  if (!schema.includes('db_class: arion_encrypted_userdb') || !schema.includes('schema_id: arion_pinyin')) throw new Error('Compiled keyboard schema lost encrypted database or schema identity.');
  verifyEnglishProfile(data);
  rmSync(user, { recursive: true, force: true });
}
