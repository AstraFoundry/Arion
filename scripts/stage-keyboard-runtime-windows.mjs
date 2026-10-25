// Build one Windows C++ ABI for Rime and its encrypted database adapter.
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, cpSync, existsSync, mkdirSync, readFileSync, readdirSync, renameSync, rmSync, statSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { pinnedFile } from './lib/pinned-file.mjs';
import { stageKeyboardData } from './lib/keyboard-data.mjs';
import { verifyWindowsPEDependencies } from './lib/windows-pe.mjs';
import { verifyWindowsKeyboardRuntime } from './lib/windows-keyboard.mjs';

if (process.platform !== 'win32' || process.arch !== 'x64') throw new Error('Windows keyboard runtime requires native Windows x64.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const manifestPath = path.join(root, 'scripts/keyboard-runtime.json');
const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
const work = path.join(root, '.workspace/keyboard-runtime/windows');
const sources = path.join(work, 'sources');
const sdk = path.join(work, 'sdk');
const destination = path.join(root, 'apps/desktop/resources/windows-amd64/keyboard');
const staging = path.join(work, `staging-${process.pid}`);
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const copy = (from, to) => { mkdirSync(path.dirname(to), { recursive: true }); copyFileSync(from, to); };
const run = (command, args, options = {}) => execFileSync(command, args, { cwd: root, windowsHide: true, stdio: 'inherit', ...options });
const compiler = process.env.CXX ?? 'g++';
const compilerVersion = run(compiler, ['--version'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).split('\n')[0].trim();
const triple = run(compiler, ['-dumpmachine'], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
if (triple !== 'x86_64-w64-mingw32') throw new Error(`The encrypted keyboard SDK requires x64 MinGW-w64 GCC; found ${triple}`);
mkdirSync(sources, { recursive: true });
mkdirSync(staging, { recursive: true });

// Unpack only after checking the exact bytes. Preserve mtimes across stage runs.
function source(spec, directory, members = []) {
  const archive = pinnedFile(spec);
  const target = path.join(sources, directory);
  const marker = path.join(target, '.arion-source-sha256');
  if (!existsSync(marker) || readFileSync(marker, 'utf8').trim() !== spec.sha256) {
    rmSync(target, { recursive: true, force: true });
    mkdirSync(target, { recursive: true });
    run('tar', ['-xf', archive, '-C', target, '--strip-components=1', ...members]);
    writeFileSync(marker, `${spec.sha256}\n`);
  }
  return target;
}
const flags = ['-G', 'Ninja', '-DCMAKE_BUILD_TYPE=Release', '-DCMAKE_POLICY_VERSION_MINIMUM=3.5',
  `-DCMAKE_CXX_COMPILER=${compiler}`, `-DCMAKE_C_COMPILER=${process.env.CC ?? 'gcc'}`,
  `-DCMAKE_INSTALL_PREFIX=${sdk}`, '-DCMAKE_INSTALL_LIBDIR=lib', '-DBUILD_SHARED_LIBS=OFF',
  '-DCMAKE_EXE_LINKER_FLAGS=-static', '-DCMAKE_SHARED_LINKER_FLAGS=-static-libgcc -static-libstdc++ -Wl,-Bstatic -lwinpthread',
  '-DBUILD_TESTING=OFF'];
function build(name, sourceDirectory, options, targets = [], install = true) {
  const directory = path.join(work, 'build', name);
  run('cmake', ['-S', sourceDirectory, '-B', directory, ...flags, ...options]);
  run('cmake', ['--build', directory, '--parallel', process.env.ARION_KEYBOARD_BUILD_JOBS ?? '6', ...(targets.length ? ['--target', ...targets] : [])]);
  if (install) run('cmake', ['--install', directory]);
  return directory;
}
function replacePinned(file, before, after) {
  const text = readFileSync(file, 'utf8');
  if (text.includes(after)) return;
  if (text.includes(before)) writeFileSync(file, text.replaceAll(before, after));
  else throw new Error(`Pinned keyboard source patch no longer applies: ${file}`);
}
try {
  const deps = Object.fromEntries(Object.entries(manifest.windows.dependencies).map(([name, spec]) => [name, source(spec, name)]));
  const boost = source(manifest.archives.boost, 'boost', ['boost_1_89_0/boost', 'boost_1_89_0/LICENSE_1_0.txt']);
  mkdirSync(path.join(sdk, 'include'), { recursive: true });
  // Boost is a build-only header dependency, never a runtime search directory.
  build('yaml', deps.yaml, ['-DYAML_CPP_BUILD_TESTS=OFF', '-DYAML_CPP_BUILD_TOOLS=OFF', '-DYAML_BUILD_SHARED_LIBS=OFF']);
  build('leveldb', deps.leveldb, ['-DLEVELDB_BUILD_TESTS=OFF', '-DLEVELDB_BUILD_BENCHMARKS=OFF', '-DHAVE_SNAPPY=OFF', '-DHAVE_CRC32C=OFF']);
  build('marisa', deps.marisa, ['-DENABLE_TOOLS=OFF', '-DENABLE_NATIVE_CODE=OFF']);
  // The exact Marisa dependency selected by librime now needs C++17 headers.
  replacePinned(path.join(deps.opencc, 'CMakeLists.txt'), '-std=c++14', '-std=c++17');
  // GCC 16 no longer supplies integer types through unrelated STL headers.
  replacePinned(path.join(deps.opencc, 'src/SerializedValues.hpp'), '#include "SerializableDict.hpp"', '#include <cstdint>\n#include "SerializableDict.hpp"');
  const openccBuild = build('opencc', deps.opencc, ['-DUSE_SYSTEM_MARISA=ON', '-DENABLE_GTEST=OFF', '-DENABLE_BENCHMARK=OFF',
    `-DCMAKE_PREFIX_PATH=${sdk}`, `-DCMAKE_CXX_FLAGS=-I${path.join(sdk, 'include').replaceAll('\\', '/')}`,
    `-DCMAKE_EXE_LINKER_FLAGS=-static -L${path.join(sdk, 'lib').replaceAll('\\', '/')}`], ['libopencc'], false);
  copy(path.join(openccBuild, 'src/libopencc.a'), path.join(sdk, 'lib/libopencc.a'));
  cpSync(path.join(deps.opencc, 'src'), path.join(sdk, 'include/opencc'), { recursive: true, filter: file => statSync(file).isDirectory() || /\.(h|hpp)$/.test(file) });
  copy(path.join(openccBuild, 'src/opencc_config.h'), path.join(sdk, 'include/opencc/opencc_config.h'));
  copy(path.join(openccBuild, 'src/Opencc_Export.h'), path.join(sdk, 'include/opencc/Opencc_Export.h'));

  const rime = source(manifest.archives.source, 'rime');
  // An extracted release has no Git identity. Do not inherit the enclosing
  // Arion checkout's branch/commit through Git's parent-directory discovery.
  replacePinned(path.join(rime, 'cmake/AddRCInfo.cmake'), 'find_program(git_executable git)', 'set(git_executable "")');
  replacePinned(path.join(rime, 'cmake/AddRCInfo.cmake'), 'set(original_filename "librime.dll")', 'set(original_filename "ArionRime.dll")');
  // A TSF host can already contain another vendor's Rime. Windows resolves
  // imported modules by base name, so this engine needs its own DLL identity.
  replacePinned(path.join(rime, 'src/CMakeLists.txt'), '  add_library(rime ${rime_src})',
    '  add_library(rime ${rime_src})\n  set_target_properties(rime PROPERTIES OUTPUT_NAME ArionRime PREFIX "")');
  // Upstream exports only two Db methods on Windows. The private adapter needs
  // its constructor and virtual metadata implementation as well. This changes
  // visibility only; the engine and SDK receive the identical patched header.
  const dbHeader = path.join(rime, 'src/rime/dict/db.h');
  const unpatched = readFileSync(dbHeader, 'utf8');
  const patched = unpatched.replace('class Db : public Class<Db, const string&> {', 'class RIME_DLL Db : public Class<Db, const string&> {')
    .replace('  Db(const path& file_path, const string& name);', '  RIME_DLL Db(const path& file_path, const string& name);')
    .replace('  virtual ~Db() = default;', '  virtual ~Db();')
    .replace('  virtual bool CreateMetadata();', '  RIME_DLL virtual bool CreateMetadata();');
  if (!patched.includes('class RIME_DLL Db :') || !patched.includes('RIME_DLL Db(const path&') || !patched.includes('virtual ~Db();') || !patched.includes('RIME_DLL virtual bool CreateMetadata();')) throw new Error('Pinned Rime private ABI export patch no longer applies.');
  if (patched !== unpatched) writeFileSync(dbHeader, patched);
  // Give the imported class an out-of-line key function so MinGW owns and
  // exports its vtable/typeinfo in the engine instead of client COMDATs.
  replacePinned(path.join(rime, 'src/rime/dict/db.cc'), '// Db members', '// Db members\n\nDb::~Db() = default;');
  // Upstream adds the dependency include directory before its own source.
  // Remove previous SDK output here so an incremental engine build cannot
  // accidentally compile against stale exported Rime headers in that prefix.
  rmSync(path.join(sdk, 'include/rime'), { recursive: true, force: true });
  for (const name of ['rime_api.h', 'rime_api_stdbool.h', 'rime_api_deprecated.h', 'rime_levers_api.h']) rmSync(path.join(sdk, 'include', name), { force: true });
  const rimeBuild = build('rime', rime, ['-DBUILD_SHARED_LIBS=ON', '-DBUILD_STATIC=ON', '-DBUILD_TEST=OFF', '-DBUILD_DATA=OFF',
    '-DENABLE_LOGGING=OFF', '-DENABLE_EXTERNAL_PLUGINS=OFF', '-DINSTALL_PRIVATE_HEADERS=ON',
    `-DCMAKE_PREFIX_PATH=${sdk}`, `-DBOOST_ROOT=${boost}`, '-DBoost_NO_SYSTEM_PATHS=ON'], ['rime', 'rime_deployer'], false);
  cpSync(path.join(rime, 'src/rime'), path.join(sdk, 'include/rime'), { recursive: true, filter: file => statSync(file).isDirectory() || /\.h$/.test(file) });
  copy(path.join(rimeBuild, 'src/rime/build_config.h'), path.join(sdk, 'include/rime/build_config.h'));
  for (const name of ['rime_api.h', 'rime_api_stdbool.h', 'rime_api_deprecated.h', 'rime_levers_api.h']) copy(path.join(rime, 'src', name), path.join(sdk, 'include', name));
  cpSync(path.join(boost, 'boost'), path.join(sdk, 'include/boost'), { recursive: true });
  cpSync(path.join(rime, 'include'), path.join(sdk, 'include'), { recursive: true });
  const importLibrary = ['lib/libArionRime.dll.a', 'src/libArionRime.dll.a'].map(member => path.join(rimeBuild, member)).find(existsSync);
  if (!importLibrary) throw new Error('Pinned Rime build did not produce its C++ import library.');
  copy(importLibrary, path.join(sdk, 'lib/libArionRime.dll.a'));
  rmSync(path.join(sdk, 'lib/librime.dll.a'), { force: true });
  const library = path.join(staging, 'lib/ArionRime.dll');
  copy(path.join(rimeBuild, 'bin/ArionRime.dll'), library);
  for (const name of ['rime_api.h', 'rime_api_stdbool.h', 'rime_api_deprecated.h', 'rime_levers_api.h']) copy(path.join(sdk, 'include', name), path.join(staging, 'include', name));
  const deployer = path.join(work, 'deployer/rime_deployer.exe');
  copy(path.join(rimeBuild, 'bin/rime_deployer.exe'), deployer);
  copy(library, path.join(path.dirname(deployer), 'ArionRime.dll'));
  rmSync(path.join(path.dirname(deployer), 'librime.dll'), { force: true });
  verifyWindowsPEDependencies([library, deployer]);
  const systemRoot = process.env.SystemRoot;
  if (!systemRoot) throw new Error('Windows schema deployment requires SystemRoot');
  const isolatedEnv = { SystemRoot: systemRoot, WINDIR: systemRoot, PATH: path.join(systemRoot, 'System32'), TEMP: process.env.TEMP, TMP: process.env.TMP };
  stageKeyboardData({ manifest, manifestPath, profile: path.join(root, 'apps/desktop/ime/shared/keyboard-data'), staging, work,
    deploy: args => run(deployer, args, { env: isolatedEnv, stdio: ['ignore', 'pipe', 'pipe'], maxBuffer: 8 * 1024 * 1024, timeout: 120000 }),
  });
  copy(path.join(boost, 'LICENSE_1_0.txt'), path.join(staging, 'licenses/boost-LICENSE'));
  copy(path.join(rime, 'include/COPYING.darts-clone'), path.join(staging, 'licenses/darts-clone-COPYING'));
  copy(path.join(rime, 'include/utf8.h'), path.join(staging, 'licenses/utf8cpp-notice.h'));
  for (const spec of manifest.windows.files) copy(pinnedFile(spec), path.join(staging, spec.to));
  // OpenCC bundles these header-only dependencies inside its pinned source.
  for (const [member, name] of [['deps/rapidjson-1.1.0/rapidjson/rapidjson.h', 'rapidjson-notice.h'], ['deps/darts-clone-0.32/darts.h', 'OpenCC-darts-clone-notice.h']]) copy(path.join(deps.opencc, member), path.join(staging, 'licenses', name));
  const files = {};
  function inventory(directory) {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const file = path.join(directory, entry.name);
      if (entry.isDirectory()) inventory(file);
      else files[path.relative(staging, file).replaceAll('\\', '/')] = { sizeBytes: statSync(file).size, sha256: hash(file) };
    }
  }
  inventory(staging);
  const buildIdentity = { compiler: compilerVersion, compilerTarget: triple, logging: false, dbVisibilityPatch: 3, openccCxx17Patch: 1,
    sources: { rime: manifest.archives.source, boost: manifest.archives.boost, ...manifest.windows.dependencies } };
  writeFileSync(path.join(staging, 'runtime.json'), `${JSON.stringify({ ...manifest.engine, architectures: manifest.windows.architectures, library: 'lib/ArionRime.dll', build: buildIdentity, files }, null, 2)}\n`);
  writeFileSync(path.join(sdk, 'build.json'), `${JSON.stringify(buildIdentity, null, 2)}\n`);
  // The host-module build runs after engine staging. A package must use the
  // verifier's default requireModule:true after recording ArionKeyboard.dll.
  verifyWindowsKeyboardRuntime(staging, { requireModule: false });
  rmSync(destination, { recursive: true, force: true });
  mkdirSync(path.dirname(destination), { recursive: true });
  renameSync(staging, destination);
  console.log(`Staged Windows x64 librime ${manifest.engine.version}, encrypted-userdb schema and prebuilt dictionary.`);
  console.log(`Build-only private SDK: ${sdk}`);
} finally { rmSync(staging, { recursive: true, force: true }); }
