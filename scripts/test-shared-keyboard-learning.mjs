// Native shared storage/ranking checks use only newly owned synthetic profiles.
// Windows also exercises actual DPAPI and the pinned private Rime C++ ABI.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { verifyWindowsPEDependencies } from './lib/windows-pe.mjs';

if (!['win32', 'darwin'].includes(process.platform)) throw new Error('Shared keyboard learning checks require native Windows or macOS.');
if (process.argv.length !== 2) throw new Error('Usage: node scripts/test-shared-keyboard-learning.mjs');
const windows = process.platform === 'win32';
if (windows && process.arch !== 'x64') throw new Error('The pinned Windows Rime ABI requires x64.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'apps/desktop/ime/shared');
const corpus = path.join(root, 'apps/desktop/ime/shared/testdata/ime-learning-v1');
const output = path.join(root, '.workspace/tests/shared-keyboard-learning');
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(path.join(output, 'run-'));
const profile = path.join(directory, 'profiles');
mkdirSync(profile);
const reportPath = path.join(directory, 'report.json');
const digest = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const identity = file => { const bytes = readFileSync(file); return { sizeBytes: bytes.length, sha256: createHash('sha256').update(bytes).digest('hex') }; };
const files = ['learning.h', 'learning.cpp', 'learning_windows.cpp', 'learning_rime.h', 'learning_rime.cpp', 'learning_test.cpp', 'learning_rime_test.cpp', 'learning_multiprocess_test.cpp', 'pinyin.c', 'pinyin.h',
  'english.h', 'english.cpp', 'english_lexicon.h', 'english_lexicon.cpp', 'emoji.h', 'emoji.cpp', 'emoji_data.h',
  'ranking.h', 'ranking.cpp', 'ranking_bridge.h', 'ranking_unicode.h', 'ranking_unicode.cpp', 'ranking_unicode_data.h', 'ranking_test.cpp',
  'ranking_unicode_sources.json', 'ranking_unicode_LICENSE.txt', 'ranking_unicode_generate.py'];
const engineSources = ['emoji.cpp', 'english_lexicon.cpp', 'spelling.cpp', 'personal_vocabulary.cpp', 'english.cpp'];
files.push(...engineSources, 'emoji.h', 'emoji_data.h', 'spelling.h', 'spelling_options.h',
  'personal_vocabulary.h', 'personal_vocabulary_bridge.h', 'english.h');
const report = { schema: 1, platform: process.platform, architecture: process.arch,
  boundary: windows ? 'Shared codec/ranking, actual current-user DPAPI, pinned Rime encrypted Db ABI and owned concurrent editor processes; no TSF/installed editor dispatch' : 'Shared codec and ranking only; macOS native Keychain, CryptoKit and AppKit checks remain separate',
  directory, profiles: profile, sourceIdentities: {}, fixtureIdentities: {}, runtimeIdentities: {}, commands: [], status: 'running' };
const save = () => writeFileSync(reportPath, `${JSON.stringify(report, null, 2)}\n`);
function run(phase, command, args, { env = process.env, timeout = 120000 } = {}) {
  const result = spawnSync(command, args, { cwd: directory, encoding: 'utf8', windowsHide: true, timeout, maxBuffer: 16 * 1024 * 1024, env });
  const record = { phase, command, args, cwd: directory, status: result.status, signal: result.signal, error: result.error?.message, stdout: result.stdout, stderr: result.stderr };
  report.commands.push(record);
  save();
  process.stdout.write(result.stdout ?? '');
  process.stderr.write(result.stderr ?? '');
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${phase} failed (${result.status}).`);
  return result.stdout ?? '';
}
try {
  for (const name of files) report.sourceIdentities[`apps/desktop/ime/shared/${name}`] = identity(path.join(source, name));
  if (windows) for (const name of ['keyboard_options.h', 'keyboard_options_test.cpp', 'json.h', 'wire.h', 'wire_text.h'])
    report.sourceIdentities[`apps/desktop/ime/windows/${name}`] = identity(path.join(source, '../windows', name));
  report.sourceIdentities['scripts/test-shared-keyboard-learning.mjs'] = identity(fileURLToPath(import.meta.url));
  const provenance = JSON.parse(readFileSync(path.join(corpus, 'provenance.json'), 'utf8'));
  for (const fixture of provenance.fixtures) {
    if (!/^[a-zA-Z0-9_.-]+$/.test(fixture.file)) throw new Error('Unsafe keyboard fixture name.');
    const file = path.join(corpus, fixture.file);
    const checked = identity(file);
    if (checked.sha256 !== fixture.sha256) throw new Error(`Pinned keyboard fixture changed: ${fixture.file}`);
    report.fixtureIdentities[`apps/desktop/ime/shared/testdata/ime-learning-v1/${fixture.file}`] = checked;
  }
  report.fixtureIdentities['apps/desktop/ime/shared/testdata/ime-learning-v1/provenance.json'] = identity(path.join(corpus, 'provenance.json'));
  const unicode = JSON.parse(readFileSync(path.join(source, 'ranking_unicode_sources.json'), 'utf8'));
  if (unicode.unicode_version !== '17.0.0' || digest(path.join(corpus, 'GraphemeBreakTest-17.0.0.txt')) !== unicode.sources['GraphemeBreakTest.txt'].sha256 || digest(path.join(corpus, 'Scripts-17.0.0.txt')) !== unicode.sources['Scripts.txt'].sha256 || digest(path.join(source, 'ranking_unicode_LICENSE.txt')) !== unicode.sources['license.txt'].sha256) throw new Error('Unicode corpus/license differs from the reviewed source ledger.');
  const compiler = process.env.CXX ?? (windows ? 'g++' : 'clang++');
  const python = process.env.PYTHON ?? (windows ? 'python' : 'python3');
  report.compiler = run('compiler-version', compiler, ['--version']).split('\n')[0].trim();
  if (windows) report.compilerTarget = run('compiler-target', compiler, ['-dumpmachine']).trim();
  report.python = run('independent-plist-reader-version', python, ['--version']).trim();
  const flags = ['-std=c++17', '-O2', '-Wall', '-Wextra', '-Werror', ...(windows ? ['-static'] : [])];
  const suffix = windows ? '.exe' : '';
  const binaries = report.binaries = {};
  const compile = (name, members, extra = []) => {
    const binary = path.join(directory, `${name}${suffix}`);
    run(`compile-${name}`, compiler, [...flags, ...members.map(member => path.join(source, member)), ...extra, '-o', binary]);
    binaries[name] = { path: binary, ...identity(binary) };
    return binary;
  };
  const ranking = compile('ranking', ['learning.cpp', 'ranking.cpp', 'ranking_unicode.cpp', 'ranking_test.cpp']);
  const learning = compile('learning', ['learning.cpp', ...(windows ? ['learning_windows.cpp'] : []), 'learning_test.cpp'], windows ? ['-lcrypt32', '-ladvapi32', '-lole32'] : []);
  let isolatedEnv = { ...process.env, TMPDIR: `${directory}${path.sep}`, TMP: directory, TEMP: directory };
  if (windows) {
    const systemRoot = process.env.SystemRoot;
    if (!systemRoot) throw new Error('Windows DPAPI checks require SystemRoot.');
    isolatedEnv = { SystemRoot: systemRoot, WINDIR: systemRoot, PATH: path.join(systemRoot, 'System32'), TEMP: directory, TMP: directory };
    const tokenCommand = path.join(systemRoot, 'System32/whoami.exe');
    const tokenArgs = ['/groups', '/fo', 'csv', '/nh'];
    const token = spawnSync(tokenCommand, tokenArgs, { cwd: directory, encoding: 'utf8', windowsHide: true, timeout: 15000, env: isolatedEnv });
    if (token.error) throw token.error;
    if (token.status !== 0) throw new Error('Windows test token integrity could not be queried.');
    report.integritySid = token.stdout.match(/S-1-16-\d+\b/)?.[0];
    report.commands.push({ phase: 'windows-test-token-integrity', command: tokenCommand, args: tokenArgs, status: token.status, integritySid: report.integritySid });
    // A restricted low-integrity test token cannot establish the provider's
    // ordinary desktop DPAPI behavior. Do not misreport it as a crypto error.
    if (!['S-1-16-8192', 'S-1-16-8448'].includes(report.integritySid)) throw new Error('Actual DPAPI acceptance requires an ordinary medium-integrity desktop token.');
    verifyWindowsPEDependencies([ranking, learning]);
  }
  run('ranking-and-unicode-conformance', ranking, [path.join(corpus, 'GraphemeBreakTest-17.0.0.txt'), path.join(corpus, 'Scripts-17.0.0.txt')], { env: isolatedEnv });
  const storageProfile = path.join(profile, 'storage');
  mkdirSync(storageProfile);
  const emitted = path.join(directory, 'emitted-legacy.bplist');
  run('shared-codec-and-native-storage', learning, [storageProfile, path.join(corpus, 'legacy-nsdata-pairs.bplist'), emitted], { env: isolatedEnv });
  run('independent-plistlib-compatibility', python, ['-c', 'import pathlib,plistlib,sys; original=plistlib.loads(pathlib.Path(sys.argv[1]).read_bytes()); emitted=plistlib.loads(pathlib.Path(sys.argv[2]).read_bytes()); assert dict(original)==dict(emitted); print("Independent plistlib mapping compatibility passed.")', path.join(corpus, 'legacy-nsdata-pairs.bplist'), emitted]);
  report.emittedLegacyIdentity = identity(emitted);
  if (windows) {
    const sdk = path.join(root, '.workspace/keyboard-runtime/windows/sdk');
    const runtime = path.join(root, 'apps/desktop/resources/windows-amd64/keyboard');
    const library = path.join(runtime, 'lib/ArionRime.dll');
    const importLibrary = path.join(sdk, 'lib/libArionRime.dll.a');
    for (const file of [library, importLibrary, path.join(sdk, 'include/rime_api.h'), path.join(sdk, 'include/rime/dict/db.h'), path.join(sdk, 'include/rime/build_config.h'), path.join(sdk, 'build.json'), path.join(runtime, 'runtime.json')]) {
      if (!existsSync(file)) throw new Error(`Stage the pinned Windows keyboard runtime first: ${file}`);
      report.runtimeIdentities[path.relative(root, file).replaceAll('\\', '/')] = identity(file);
    }
    const sdkBuild = JSON.parse(readFileSync(path.join(sdk, 'build.json'), 'utf8'));
    // GCC prints argv[0]: an explicit Windows executable path adds .exe to
    // the same compiler's first line. Compare version identity and target.
    const compilerIdentity = value => value.replace(/^g\+\+\.exe /, 'g++ ');
    if (compilerIdentity(sdkBuild.compiler) !== compilerIdentity(report.compiler) ||
        sdkBuild.compilerTarget !== report.compilerTarget || report.compilerTarget !== 'x86_64-w64-mingw32') throw new Error('The private Rime SDK and learning adapter must use the same GCC ABI.');
    const abi = compile('rime-learning-abi', ['learning.cpp', 'learning_windows.cpp', 'learning_rime.cpp', 'ranking.cpp', 'ranking_unicode.cpp', 'learning_rime_test.cpp'], ['-DRIME_IMPORTS', '-isystem', path.join(sdk, 'include'), importLibrary, '-lcrypt32', '-ladvapi32', '-lole32']);
    copyFileSync(library, path.join(directory, 'ArionRime.dll'));
    verifyWindowsPEDependencies([abi]);
    const abiProfile = path.join(profile, 'rime-abi');
    mkdirSync(abiProfile);
    run('actual-rime-encrypted-db-cursor-transaction-context-abi', abi, [abiProfile], { env: isolatedEnv });
    const pinyin = path.join(directory, 'multiprocess-pinyin.o');
    run('compile-multiprocess-pinyin-c-core', process.env.CC ?? 'gcc', ['-std=c11', '-O2', '-Wall', '-Wextra', '-Werror',
      '-I', path.join(sdk, 'include'), '-c', path.join(source, 'pinyin.c'), '-o', pinyin]);
    const multi = compile('multiprocess-learning', ['learning.cpp', 'learning_windows.cpp', 'learning_rime.cpp', 'ranking.cpp', 'ranking_unicode.cpp', ...engineSources, 'learning_multiprocess_test.cpp'],
      ['-DRIME_IMPORTS', '-isystem', path.join(sdk, 'include'), pinyin, importLibrary, '-lcrypt32', '-ladvapi32', '-lole32']);
    verifyWindowsPEDependencies([multi]);
    const multiProfile = path.join(profile, 'multiprocess');
    mkdirSync(multiProfile);
    run('actual-multiprocess-rime-and-dpapi-learning', multi, [multiProfile, path.join(runtime, 'data')], { env: isolatedEnv, timeout: 120000 });
    const options = compile('native-keyboard-options', ['../windows/keyboard_options_test.cpp'], ['-municode', '-ladvapi32', '-lole32']);
    verifyWindowsPEDependencies([options]);
    run('actual-atomic-mode-persistence-and-crossprocess-options', options, [path.join(profile, 'native-options')], { env: isolatedEnv, timeout: 60000 });
  } else {
    report.remainingNativeGates = ['node scripts/test-keyboard-learning.mjs', 'node scripts/test-keyboard-input.mjs'];
  }
  report.binaries = binaries;
  for (const [name, expected] of Object.entries(report.sourceIdentities)) if (digest(path.join(root, name)) !== expected.sha256) throw new Error(`Source changed during shared keyboard checks: ${name}`);
  for (const [name, expected] of Object.entries(report.runtimeIdentities)) if (digest(path.join(root, name)) !== expected.sha256) throw new Error(`Runtime changed during shared keyboard checks: ${name}`);
  report.status = 'passed';
} catch (error) { report.status = 'failed'; report.error = error.message; process.exitCode = 1; }
finally { save(); console.log(`Shared keyboard learning evidence: ${reportPath}`); }
if (report.status !== 'passed') throw new Error(report.error);
