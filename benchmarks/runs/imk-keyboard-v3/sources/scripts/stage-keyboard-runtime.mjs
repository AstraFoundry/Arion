// Stage a pinned universal offline keyboard engine; SDK headers remain build-only.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { chmodSync, copyFileSync, cpSync, existsSync, mkdirSync, readFileSync, readdirSync, renameSync, rmSync, statSync, utimesSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './lib/pinned-file.mjs';

if (process.platform !== 'darwin') {
  console.log('The offline keyboard runtime is macOS-only; skipped.');
  process.exit(0);
}
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const manifestPath = path.join(root, 'scripts/keyboard-runtime.json');
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
const work = path.join(root, '.workspace/keyboard-runtime');
const destination = path.join(root, 'apps/desktop/resources/darwin/keyboard');
const staging = path.join(work, `staging-${process.pid}`);
const sdk = path.join(work, 'sdk');
const profile = path.join(root, 'apps/desktop/ime/macos/keyboard-data');
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const run = (command, args, options = {}) => execFileSync(command, args, { cwd: root, ...options });
const copy = (from, to) => { mkdirSync(path.dirname(to), { recursive: true }); copyFileSync(from, to); };
const member = (archive, name, to) => {
  mkdirSync(path.dirname(to), { recursive: true });
  writeFileSync(to, run('tar', ['-xOf', archive, name], { maxBuffer: 32 * 1024 * 1024 }));
};
const extract = (archive, to, names = []) => {
  mkdirSync(to, { recursive: true });
  run('tar', ['-xf', archive, '-C', to, ...names]);
};

mkdirSync(work, { recursive: true });
rmSync(staging, { recursive: true, force: true });
mkdirSync(staging, { recursive: true });
try {
  const archives = Object.fromEntries(Object.entries(manifest.archives).map(([key, spec]) => [key, pinnedFile(spec)]));
  const runtime = path.join(work, 'release');
  rmSync(runtime, { recursive: true, force: true });
  extract(archives.runtime, runtime);
  const library = path.join(runtime, 'dist/lib/librime.1.17.0.dylib');
  const architectures = run('lipo', ['-archs', library], { encoding: 'utf8' }).trim().split(/\s+/).sort();
  if (JSON.stringify(architectures) !== JSON.stringify([...manifest.engine.architectures].sort())) throw new Error('Keyboard engine must include arm64 and x86_64.');
  const linkage = run('otool', ['-L', library], { encoding: 'utf8' });
  const dependencies = [...linkage.matchAll(/^\s+(\S+) \(compatibility/gm)].map(match => match[1]);
  const allowed = new Set(['@rpath/librime.1.dylib', '/usr/lib/libSystem.B.dylib', '/usr/lib/libc++.1.dylib']);
  if (dependencies.length !== 6 || dependencies.some(name => !allowed.has(name))) throw new Error(`Unexpected nonportable keyboard engine linkage:\n${linkage}`);
  copy(library, path.join(staging, 'lib/librime.1.dylib'));
  chmodSync(path.join(staging, 'lib/librime.1.dylib'), 0o644);
  // Public headers are small; private C++/Boost headers are never shipped.
  cpSync(path.join(runtime, 'dist/include'), path.join(staging, 'include'), { recursive: true });

  const sdkFingerprint = createHash('sha256').update(JSON.stringify(manifest.archives)).digest('hex');
  const sdkMarker = path.join(sdk, 'manifest.sha256');
  if (!existsSync(sdkMarker) || readFileSync(sdkMarker, 'utf8').trim() !== sdkFingerprint) {
    rmSync(sdk, { recursive: true, force: true });
    mkdirSync(path.join(sdk, 'include'), { recursive: true });
    const source = path.join(work, 'source');
    rmSync(source, { recursive: true, force: true });
    extract(archives.source, source);
    const sourceRoot = path.join(source, `librime-${manifest.engine.commit}`);
    cpSync(path.join(sourceRoot, 'src/rime'), path.join(sdk, 'include/rime'), { recursive: true, filter: file => statSync(file).isDirectory() || /\.h$/.test(file) });
    extract(archives.dependencies, sdk, ['include']);
    const boost = path.join(work, 'boost');
    rmSync(boost, { recursive: true, force: true });
    extract(archives.boost, boost, ['boost_1_89_0/boost', 'boost_1_89_0/LICENSE_1_0.txt']);
    cpSync(path.join(boost, 'boost_1_89_0/boost'), path.join(sdk, 'include/boost'), { recursive: true });
    copy(path.join(boost, 'boost_1_89_0/LICENSE_1_0.txt'), path.join(sdk, 'boost-LICENSE'));
    cpSync(path.join(runtime, 'dist/include'), path.join(sdk, 'include'), { recursive: true });
    // Official macOS release: ENABLE_LOGGING=ON, ALSO_LOG_TO_STDERR=OFF.
    writeFileSync(path.join(sdk, 'include/rime/build_config.h'), '#ifndef RIME_BUILD_CONFIG_H_\n#define RIME_BUILD_CONFIG_H_\n#define RIME_ENABLE_LOGGING 1\n#define RIME_DATA_DIR "rime-data"\n#define RIME_PLUGINS_DIR "rime-plugins"\n#endif\n');
    writeFileSync(sdkMarker, `${sdkFingerprint}\n`);
  }
  copy(path.join(sdk, 'boost-LICENSE'), path.join(staging, 'licenses/boost-LICENSE'));
  member(archives.dependencies, 'include/COPYING.darts-clone', path.join(staging, 'licenses/darts-clone-COPYING'));
  member(archives.dependencies, 'include/utf8.h', path.join(staging, 'licenses/utf8cpp-notice.h'));
  for (const spec of manifest.files) copy(pinnedFile(spec), path.join(staging, spec.to));
  for (const name of ['arion_pinyin.schema.yaml', 'default.custom.yaml']) copy(path.join(profile, name), path.join(staging, 'data', name));
  copy(path.join(profile, 'README.md'), path.join(staging, 'licenses/Arion-profile-README.md'));
  copy(manifestPath, path.join(staging, 'licenses/keyboard-runtime.json'));

  const data = path.join(staging, 'data');
  const build = path.join(data, 'build');
  const user = path.join(staging, 'compile-profile');
  mkdirSync(user, { recursive: true });
  mkdirSync(build, { recursive: true });
  // Compiler schema metadata uses source modification times; keep them stable.
  for (const name of readdirSync(data)) if (statSync(path.join(data, name)).isFile()) utimesSync(path.join(data, name), 1, 1);
  try {
    run(path.join(runtime, 'dist/bin/rime_deployer'), ['--build', user, data, build], {
      env: { ...process.env, DYLD_LIBRARY_PATH: path.join(runtime, 'dist/lib'), GLOG_logtostderr: '1' },
      stdio: ['ignore', 'ignore', 'pipe'], maxBuffer: 8 * 1024 * 1024,
    });
  } catch (error) {
    const log = path.join(work, 'deployment-failure.log');
    writeFileSync(log, error.stderr ?? error.message);
    throw new Error(`Keyboard schema deployment failed; inspect ${log}`, { cause: error.message });
  }
  for (const name of ['default.yaml', 'arion_pinyin.schema.yaml', 'pinyin_simp.table.bin', 'pinyin_simp.reverse.bin', 'pinyin_simp.prism.bin']) {
    if (!existsSync(path.join(build, name))) throw new Error(`Keyboard prebuild is incomplete: ${name}`);
  }
  const schema = readFileSync(path.join(build, 'arion_pinyin.schema.yaml'), 'utf8');
  if (!schema.includes('db_class: arion_encrypted_userdb') || !schema.includes('schema_id: arion_pinyin')) throw new Error('Compiled keyboard schema lost encrypted database or schema identity.');
  rmSync(user, { recursive: true, force: true });
  const files = {};
  const inventory = directory => {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const file = path.join(directory, entry.name);
      if (entry.isDirectory()) inventory(file);
      else files[path.relative(staging, file)] = { sizeBytes: statSync(file).size, sha256: hash(file) };
    }
  };
  inventory(staging);
  writeFileSync(path.join(staging, 'runtime.json'), `${JSON.stringify({ ...manifest.engine, files }, null, 2)}\n`);
  rmSync(destination, { recursive: true, force: true });
  mkdirSync(path.dirname(destination), { recursive: true });
  renameSync(staging, destination);
  console.log(`Staged librime ${manifest.engine.version} (arm64 + x86_64), arion_pinyin prebuilt data and license notices.`);
  console.log(`Build-only SDK: ${path.relative(root, sdk)}/include`);
} finally {
  rmSync(staging, { recursive: true, force: true });
}
