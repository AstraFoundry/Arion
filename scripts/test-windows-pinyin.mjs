// Actual conversion through the same shared core and encrypted Rime backend
// used by the Windows input method. This is not installed-editor acceptance.
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'win32') throw new Error('Windows Pinyin tests require native Windows.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'apps/desktop/ime/shared');
const sdk = path.join(root, '.workspace/keyboard-runtime/windows/sdk');
const runtime = path.join(root, 'apps/desktop/resources/windows-amd64/keyboard');
const library = path.join(runtime, 'lib/ArionRime.dll');
if (!existsSync(library)) throw new Error('Stage the pinned Windows keyboard runtime first.');
const output = path.join(root, '.workspace/tests/windows-pinyin');
mkdirSync(output, { recursive: true });
const runDirectory = mkdtempSync(path.join(output, 'run-'));
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const sourceNames = ['pinyin.c', 'pinyin.h', 'menu.c', 'menu.h', 'input.c', 'input.h', 'learning.cpp', 'learning.h', 'learning_windows.cpp',
  'learning_rime.cpp', 'learning_rime.h', 'ranking.cpp', 'ranking.h', 'ranking_unicode.cpp', 'ranking_unicode.h',
  'ranking_unicode_data.h', 'pinyin_native_test.cpp'];
const sourceIdentities = Object.fromEntries(sourceNames.map(name => [name, hash(path.join(source, name))]));
const runtimeSha256 = hash(library);
const run = (command, args) => execFileSync(command, args, { cwd: root, stdio: 'inherit', windowsHide: true });
const c = process.env.CC ?? 'gcc', cxx = process.env.CXX ?? 'g++';
const pinyin = path.join(runDirectory, 'pinyin.o');
const menu = path.join(runDirectory, 'menu.o');
const input = path.join(runDirectory, 'input.o');
run(c, ['-std=c11', '-O2', '-Wall', '-Wextra', '-Werror', '-I', path.join(sdk, 'include'),
  '-c', path.join(source, 'pinyin.c'), '-o', pinyin]);
run(c, ['-std=c11', '-O2', '-Wall', '-Wextra', '-Werror', '-I', path.join(sdk, 'include'),
  '-c', path.join(source, 'menu.c'), '-o', menu]);
run(c, ['-std=c11', '-O2', '-Wall', '-Wextra', '-Werror',
  '-c', path.join(source, 'input.c'), '-o', input]);
const binary = path.join(runDirectory, 'pinyin-native-test.exe');
run(cxx, ['-std=c++17', '-DRIME_IMPORTS', '-O2', '-Wall', '-Wextra', '-Werror', '-Wno-missing-field-initializers', '-static',
  '-isystem', path.join(sdk, 'include'), ...['learning.cpp', 'learning_windows.cpp', 'learning_rime.cpp', 'ranking.cpp', 'ranking_unicode.cpp', 'pinyin_native_test.cpp'].map(name => path.join(source, name)),
  pinyin, menu, input, path.join(sdk, 'lib/libArionRime.dll.a'), '-lcrypt32', '-ladvapi32', '-lole32', '-o', binary]);
copyFileSync(library, path.join(runDirectory, 'ArionRime.dll'));
const profile = path.join(runDirectory, 'profile');
mkdirSync(profile);
const systemRoot = process.env.SystemRoot;
if (!systemRoot) throw new Error('SystemRoot is required for an isolated Windows test environment.');
const execution = spawnSync(binary, [path.join(runtime, 'data'), profile], { cwd: runDirectory, windowsHide: true,
  encoding: 'utf8', timeout: 120000, env: { SystemRoot: systemRoot, WINDIR: systemRoot,
    PATH: path.join(systemRoot, 'System32'), TEMP: process.env.TEMP, TMP: process.env.TMP, APPDATA: process.env.APPDATA } });
const changedSources = sourceNames.filter(name => hash(path.join(source, name)) !== sourceIdentities[name]);
const runtimeChanged = hash(library) !== runtimeSha256;
const report = { schema: 1, boundary: 'actual Rime and encrypted provider; no TSF or installed editor dispatch',
  binarySha256: hash(binary), runtimeSha256, profile, sourceIdentities, changedSources, runtimeChanged,
  status: execution.status, error: execution.error?.message, stdout: execution.stdout, stderr: execution.stderr };
const reportPath = path.join(runDirectory, 'report.json');
writeFileSync(reportPath, JSON.stringify(report, null, 2) + '\n');
process.stdout.write(execution.stdout ?? '');
process.stderr.write(execution.stderr ?? '');
console.log(`Native Pinyin evidence: ${reportPath}`);
if (execution.error) throw execution.error;
if (execution.status !== 0) throw new Error(`Real Pinyin test failed (${execution.status}).`);
if (changedSources.length || runtimeChanged) throw new Error('Pinyin sources or runtime changed during the test; inspect the report before retrying.');
