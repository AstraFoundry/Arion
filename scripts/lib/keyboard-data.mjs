// Both native hosts consume the same pinned dictionary and compiled schema.
import { copyFileSync, existsSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, utimesSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { pinnedFile } from './pinned-file.mjs';

export function stageKeyboardData({ manifest, manifestPath, profile, staging, work, deploy }) {
  const copy = (from, to) => { mkdirSync(path.dirname(to), { recursive: true }); copyFileSync(from, to); };
  for (const spec of manifest.files) copy(pinnedFile(spec), path.join(staging, spec.to));
  for (const name of ['arion_pinyin.schema.yaml', 'default.custom.yaml']) copy(path.join(profile, name), path.join(staging, 'data', name));
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
  for (const name of ['default.yaml', 'arion_pinyin.schema.yaml', 'pinyin_simp.table.bin', 'pinyin_simp.reverse.bin', 'pinyin_simp.prism.bin']) {
    if (!existsSync(path.join(build, name))) throw new Error(`Keyboard prebuild is incomplete: ${name}`);
  }
  const schema = readFileSync(path.join(build, 'arion_pinyin.schema.yaml'), 'utf8');
  if (!schema.includes('db_class: arion_encrypted_userdb') || !schema.includes('schema_id: arion_pinyin')) throw new Error('Compiled keyboard schema lost encrypted database or schema identity.');
  rmSync(user, { recursive: true, force: true });
}
