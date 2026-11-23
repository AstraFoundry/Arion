// Stage the phone keyboards' Rime engine and data (ADR 053): the pinned
// librime with its dependencies built from pinned sources for each device and
// simulator slice, the Arion bridge, and schemas prebuilt on the host.
// Usage: node scripts/stage-mobile-rime.mjs [ios]
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, readFileSync, readdirSync, renameSync, rmSync, statSync, utimesSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildParallelism } from './build-parallelism.mjs';
import { pinnedFile } from './lib/pinned-file.mjs';
import { verifiedCorrectionIdentity } from './lib/keyboard-correction-patch.mjs';
import { buildMacKeyboardRuntime } from './lib/keyboard-macos-runtime.mjs';
import { englishLexiconFile } from './lib/keyboard-data.mjs';
import { writeEnglishLexicon } from './lib/keyboard-english.mjs';

if (process.platform !== 'darwin') throw new Error('Staging the mobile Rime engine requires macOS with Xcode.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const desktop = JSON.parse(readFileSync(path.join(root, 'scripts/keyboard-runtime.json'), 'utf8'));
const mobile = JSON.parse(readFileSync(path.join(root, 'packages/mobile-rime/runtime.json'), 'utf8'));
const work = path.join(root, '.workspace/mobile-rime');
const shared = path.join(root, 'apps/desktop/ime/shared');
const bridge = path.join(root, 'packages/mobile-rime');
const iosOutput = path.join(root, 'apps/ios/ArionKit/Frameworks/ArionRime.xcframework');
const dataOutput = path.join(root, 'apps/ios/ArionKeyboard/Rime');
const jobs = String(buildParallelism());
const digest = value => createHash('sha256').update(value).digest('hex');
const hash = file => digest(readFileSync(file));
// The desktop's shared Rime components the phones run: encrypted learning,
// ranking, the correcting Pinyin translator and the bilingual English one.
const glue = ['learning.cpp', 'learning_posix.cpp', 'learning_rime.cpp', 'ranking.cpp', 'ranking_unicode.cpp', 'spelling.cpp',
  'personal_vocabulary.cpp', 'personal_vocabulary_store.cpp', 'english.cpp', 'english_lexicon.cpp', 'english_history.cpp'];
mkdirSync(work, { recursive: true });

function run(command, args, label, options = {}) {
  const result = spawnSync(command, args, { encoding: 'utf8', maxBuffer: 64 * 1024 * 1024, timeout: 1800000, ...options });
  const log = path.join(work, 'logs', `${label}.log`);
  mkdirSync(path.dirname(log), { recursive: true });
  writeFileSync(log, `${command} ${args.join(' ')}\n${result.stdout ?? ''}\n${result.stderr ?? ''}`);
  if (result.error || result.status !== 0) throw new Error(`${label} failed; see ${log}`);
  return result.stdout;
}
const extract = (archive, to) => { mkdirSync(to, { recursive: true }); run('tar', ['-xf', archive, '-C', to], `extract-${path.basename(to)}`); };

// Sources are extracted once per pin; the engine gets the desktop's patch.
function sources() {
  const directory = path.join(work, 'sources', digest(JSON.stringify([desktop.archives.source, desktop.patches, mobile.dependencies])).slice(0, 16));
  const engine = path.join(directory, `librime-${desktop.engine.commit}`);
  if (!existsSync(path.join(directory, 'ready'))) {
    rmSync(directory, { recursive: true, force: true });
    extract(pinnedFile(desktop.archives.source), directory);
    const spelling = path.join(engine, 'src/rime/algo/syllabifier.cc');
    writeFileSync(spelling, verifiedCorrectionIdentity(readFileSync(spelling, 'utf8'), desktop.patches.correctionIdentity));
    for (const spec of Object.values(mobile.dependencies)) extract(pinnedFile(spec), directory);
    const boost = path.join(directory, 'boost');
    mkdirSync(boost);
    run('tar', ['-xf', pinnedFile(desktop.archives.boost), '-C', boost, 'boost_1_89_0/boost', 'boost_1_89_0/LICENSE_1_0.txt'], 'extract-boost');
    writeFileSync(path.join(directory, 'ready'), '');
  }
  return { directory, engine, boost: path.join(directory, 'boost/boost_1_89_0') };
}

// The macOS slice only runs the bridge test on the build machine.
const hostSlice = { sdk: 'macosx', architectures: [process.arch === 'arm64' ? 'arm64' : 'x86_64'] };
const deploymentTarget = slice => slice.sdk === 'macosx' ? '14.0' : mobile.ios.deploymentTarget;
function cmakeTarget(slice) {
  return ['-G', 'Unix Makefiles', `-DCMAKE_SYSTEM_NAME=${slice.sdk === 'macosx' ? 'Darwin' : 'iOS'}`, `-DCMAKE_OSX_SYSROOT=${slice.sdk}`,
    `-DCMAKE_OSX_ARCHITECTURES=${slice.architectures.join(';')}`, `-DCMAKE_OSX_DEPLOYMENT_TARGET=${deploymentTarget(slice)}`];
}
// Bump when the build commands change; editing anything else here keeps caches.
const recipe = 1;

// One merged static library per slice: dependencies, librime, the bridge.
function buildSlice(slice, src) {
  const name = `${slice.sdk}-${slice.architectures.join('_')}`;
  const directory = path.join(work, 'build', name);
  const prefix = path.join(directory, 'prefix');
  const merged = path.join(directory, 'libArionRime.a');
  const identity = digest(JSON.stringify({ slice, mobile, source: desktop.archives.source, patches: desktop.patches,
    recipe, bridge: ['include/arion_rime.h', 'src/arion_rime.cpp'].map(file => hash(path.join(bridge, file))),
    glue: glue.map(file => hash(path.join(shared, file))),
    sdk: run('xcrun', ['--sdk', slice.sdk, '--show-sdk-version'], `sdk-${name}`).trim() }));
  const marker = path.join(directory, 'identity');
  if (existsSync(marker) && readFileSync(marker, 'utf8') === identity && existsSync(merged)) return merged;
  rmSync(directory, { recursive: true, force: true });
  mkdirSync(prefix, { recursive: true });
  const common = [...cmakeTarget(slice), '-DCMAKE_BUILD_TYPE=Release', `-DCMAKE_INSTALL_PREFIX=${prefix}`, `-DCMAKE_PREFIX_PATH=${prefix}`,
    `-DCMAKE_FIND_ROOT_PATH=${prefix}`, '-DCMAKE_FIND_ROOT_PATH_MODE_PACKAGE=BOTH', '-DCMAKE_FIND_ROOT_PATH_MODE_LIBRARY=BOTH',
    '-DCMAKE_FIND_ROOT_PATH_MODE_INCLUDE=BOTH', '-DCMAKE_IGNORE_PREFIX_PATH=/opt/homebrew;/usr/local',
    '-DBUILD_SHARED_LIBS=OFF', '-DCMAKE_POSITION_INDEPENDENT_CODE=ON', `-DCMAKE_CXX_FLAGS=-I${prefix}/include`];
  for (const [key, spec] of Object.entries(mobile.dependencies)) {
    const build = path.join(directory, key);
    run('cmake', ['-S', path.join(src.directory, spec.root), '-B', build, ...common, ...spec.options], `${name}-${key}-configure`);
    run('cmake', ['--build', build, '--target', spec.target ?? 'install', '--parallel', jobs], `${name}-${key}-build`);
    if (spec.target) run('cmake', ['-DCMAKE_INSTALL_LOCAL_ONLY=ON', '-P', path.join(build, spec.installDirectory, 'cmake_install.cmake')], `${name}-${key}-install`);
  }
  const engineBuild = path.join(directory, 'librime');
  run('cmake', ['-S', src.engine, '-B', engineBuild, ...common, `-DCMAKE_INCLUDE_PATH=${prefix}/include`, `-DCMAKE_LIBRARY_PATH=${prefix}/lib`,
    '-DBUILD_STATIC=ON', '-DBUILD_TEST=OFF', '-DBUILD_DATA=OFF', '-DBUILD_MERGED_PLUGINS=OFF', '-DENABLE_EXTERNAL_PLUGINS=OFF',
    '-DENABLE_LOGGING=OFF', '-DCMAKE_MACOSX_BUNDLE=OFF', '-DCMAKE_DISABLE_FIND_PACKAGE_Gflags=ON',
    '-DBoost_NO_BOOST_CMAKE=ON', '-DBoost_NO_SYSTEM_PATHS=ON', `-DBOOST_ROOT=${src.boost}`, `-DBoost_INCLUDE_DIR=${src.boost}`], `${name}-librime-configure`);
  run('cmake', ['--build', engineBuild, '--target', 'rime-static', '--parallel', jobs], `${name}-librime-build`);
  // The glue sees librime's private headers the same way the desktop does.
  const configHeader = path.join(directory, 'rime-config/rime/build_config.h');
  mkdirSync(path.dirname(configHeader), { recursive: true });
  writeFileSync(configHeader, '#ifndef RIME_BUILD_CONFIG_H_\n#define RIME_BUILD_CONFIG_H_\n#define RIME_DATA_DIR "rime-data"\n#define RIME_PLUGINS_DIR "rime-plugins"\n#endif\n');
  const objects = [];
  const target = slice.sdk === 'macosx' ? `${slice.architectures[0]}-apple-macos${deploymentTarget(slice)}`
    : `${slice.architectures[0]}-apple-ios${deploymentTarget(slice)}${slice.sdk === 'iphonesimulator' ? '-simulator' : ''}`;
  for (const file of [...glue.map(f => path.join(shared, f)), path.join(bridge, 'src/arion_rime.cpp')]) {
    const object = path.join(directory, 'glue', `${path.basename(file, '.cpp')}.o`);
    mkdirSync(path.dirname(object), { recursive: true });
    run('xcrun', ['--sdk', slice.sdk, 'clang++', '-std=c++17', '-O2', '-target', target, '-fvisibility=hidden', '-DRIME_STATIC',
      '-I', path.join(bridge, 'include'), '-I', path.join(directory, 'rime-config'), '-I', path.join(src.engine, 'src'), '-I', path.join(src.engine, 'include'),
      '-isystem', src.boost, '-isystem', path.join(prefix, 'include'), '-c', file, '-o', object], `${name}-${path.basename(file)}`);
    objects.push(object);
  }
  const libraries = ['leveldb', 'marisa', 'yaml-cpp', 'opencc'].map(lib => path.join(prefix, 'lib', `lib${lib}.a`));
  run('xcrun', ['libtool', '-static', '-o', merged, path.join(engineBuild, 'lib/librime.a'), ...libraries, ...objects], `${name}-merge`);
  writeFileSync(marker, identity);
  return merged;
}

// Schemas compile on the host with the desktop's pinned deployer.
function stageData(staging) {
  const runtime = buildMacKeyboardRuntime({ manifest: desktop, work: path.join(root, '.workspace/keyboard-runtime'),
    minimumSystemVersion: JSON.parse(readFileSync(path.join(root, 'apps/desktop/desktop.json'), 'utf8')).macos.minimumSystemVersion });
  const source = path.join(staging, 'source');
  const wanted = ['data/pinyin_simp.schema.yaml', 'data/pinyin_simp.dict.yaml', 'data/default.yaml', 'data/key_bindings.yaml', 'data/punctuation.yaml', 'data/symbols.yaml'];
  for (const spec of desktop.files) {
    if (wanted.includes(spec.to)) copyFileSync(pinnedFile(spec), path.join(mkdirFor(source), path.basename(spec.to)));
    if (spec.to.startsWith('licenses/') && /(rime-pinyin-simp|rime-prelude|librime-LICENSE|leveldb|marisa|opencc|yaml-cpp)/.test(spec.to))
      copyFileSync(pinnedFile(spec), path.join(mkdirFor(path.join(staging, 'licenses')), path.basename(spec.to)));
  }
  for (const name of readdirSync(path.join(bridge, 'data'))) copyFileSync(path.join(bridge, 'data', name), path.join(source, name));
  for (const name of readdirSync(source)) utimesSync(path.join(source, name), 1, 1);
  const build = path.join(staging, 'data/build');
  const user = path.join(staging, 'deploy-user');
  mkdirSync(build, { recursive: true });
  mkdirSync(user, { recursive: true });
  run(runtime.deployer, ['--build', user, source, build], 'deploy', {
    env: { ...process.env, DYLD_LIBRARY_PATH: path.dirname(runtime.library), GLOG_logtostderr: '1' } });
  const required = ['default.yaml', 'arion_pinyin.schema.yaml', 'arion_pinyin_9key.schema.yaml', 'pinyin_simp.table.bin',
    'pinyin_simp.reverse.bin', 'pinyin_simp.prism.bin', 'arion_pinyin_9key.prism.bin'];
  for (const name of required) if (!existsSync(path.join(build, name))) throw new Error(`Mobile Rime prebuild is incomplete: ${name}`);
  for (const schema of ['arion_pinyin', 'arion_pinyin_9key']) {
    const text = readFileSync(path.join(build, `${schema}.schema.yaml`), 'utf8');
    if (!text.includes('db_class: arion_encrypted_userdb') || !text.includes(`schema_id: ${schema}`)) throw new Error(`Compiled ${schema} lost its encrypted database or identity.`);
  }
  if (!readFileSync(path.join(build, 'arion_pinyin.schema.yaml'), 'utf8').includes('- arion_bilingual'))
    throw new Error('Compiled arion_pinyin lost its bilingual translator.');
  // The 9-key syllable list comes from the shipped dictionary itself.
  const syllables = new Set();
  let body = false;
  for (const line of readFileSync(path.join(source, 'pinyin_simp.dict.yaml'), 'utf8').split('\n')) {
    if (line === '...') { body = true; continue; }
    if (!body || !line || line.startsWith('#')) continue;
    const [, code] = line.split('\t');
    if (code && /^[a-z]+$/.test(code)) syllables.add(code);
  }
  if (syllables.size < 300) throw new Error('Too few Pinyin syllables in the pinned dictionary.');
  writeFileSync(path.join(staging, 'data/arion_syllables.txt'), `${[...syllables].sort().join('\n')}\n`);
  // The desktop's complete English lexicon from the same pinned sources, which
  // ship beside it with their notices as on the desktop.
  const english = path.join(staging, 'english');
  for (const spec of desktop.files.filter(file => /^licenses\/(esdb|cspell|frequencywords)\//.test(file.to))) {
    copyFileSync(pinnedFile(spec), path.join(mkdirFor(path.dirname(path.join(english, spec.to))), path.basename(spec.to)));
    copyFileSync(pinnedFile(spec), path.join(mkdirFor(path.dirname(path.join(staging, spec.to))), path.basename(spec.to)));
  }
  mkdirSync(path.join(english, 'data'), { recursive: true });
  writeEnglishLexicon(english, desktop);
  renameSync(path.join(english, 'data', englishLexiconFile), path.join(staging, 'data', englishLexiconFile));
  rmSync(english, { recursive: true });
  rmSync(source, { recursive: true });
  rmSync(user, { recursive: true });
  copyFileSync(path.join(src().boost, 'LICENSE_1_0.txt'), path.join(staging, 'licenses/boost-LICENSE'));
}
function mkdirFor(directory) { mkdirSync(directory, { recursive: true }); return directory; }
let cachedSources;
function src() { return cachedSources ??= sources(); }

function stageIOS() {
  const libraries = mobile.ios.slices.map(slice => buildSlice(slice, src()));
  const headers = path.join(work, 'headers');
  rmSync(headers, { recursive: true, force: true });
  mkdirSync(headers, { recursive: true });
  copyFileSync(path.join(bridge, 'include/arion_rime.h'), path.join(headers, 'arion_rime.h'));
  writeFileSync(path.join(headers, 'module.modulemap'), 'module ArionRime {\n  header "arion_rime.h"\n  link "c++"\n  export *\n}\n');
  const framework = path.join(work, 'staging', 'ArionRime.xcframework');
  rmSync(framework, { recursive: true, force: true });
  run('xcodebuild', ['-create-xcframework', ...libraries.flatMap(library => ['-library', library, '-headers', headers]), '-output', framework], 'xcframework');
  const data = `${dataOutput}.staging`;
  rmSync(data, { recursive: true, force: true });
  mkdirSync(data, { recursive: true });
  stageData(data);
  const files = {};
  const inventory = directory => {
    for (const entry of readdirSync(directory, { withFileTypes: true })) {
      const file = path.join(directory, entry.name);
      if (entry.isDirectory()) inventory(file);
      else files[path.relative(data, file)] = { sizeBytes: statSync(file).size, sha256: hash(file) };
    }
  };
  inventory(data);
  writeFileSync(path.join(data, 'runtime.json'), `${JSON.stringify({ engine: desktop.engine.version, commit: desktop.engine.commit, files }, null, 2)}\n`);
  for (const [from, to] of [[framework, iosOutput], [data, dataOutput]]) {
    rmSync(to, { recursive: true, force: true });
    mkdirSync(path.dirname(to), { recursive: true });
    renameSync(from, to);
  }
  console.log(`Staged librime ${desktop.engine.version} for ${mobile.ios.slices.map(s => s.sdk).join(' and ')}, with prebuilt arion_pinyin and arion_pinyin_9key.`);
}

// The bridge runs against the staged data on this Mac before anything ships.
function testBridge() {
  const library = buildSlice(hostSlice, src());
  const directory = path.join(work, 'test');
  rmSync(directory, { recursive: true, force: true });
  mkdirSync(directory, { recursive: true });
  const binary = path.join(directory, 'arion_rime_test');
  run('xcrun', ['clang++', '-std=c++17', '-O2', '-I', path.join(bridge, 'include'), path.join(bridge, 'src/arion_rime_test.cpp'), library, '-o', binary], 'test-compile');
  process.stdout.write(run(binary, [path.join(dataOutput, 'data'), directory], 'test-run'));
}

const platform = process.argv[2] ?? 'ios';
if (platform !== 'ios') throw new Error(`Unknown mobile Rime platform: ${platform}`);
stageIOS();
testBridge();
