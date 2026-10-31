// Native algorithm checks use an isolated synthetic prism, never user data.
import { cachedSpawn } from './lib/windows-compile-cache.mjs';
import { createHash } from 'node:crypto';
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (!['darwin', 'win32'].includes(process.platform)) throw new Error('Native macOS or Windows required.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const windows = process.platform === 'win32';
const sdk = path.join(root, '.workspace/keyboard-runtime', windows ? 'windows/sdk' : 'sdk');
const runtime = path.join(root, 'apps/desktop/resources', windows ? 'windows-amd64' : 'darwin', 'keyboard');
const output = path.join(root, '.workspace/tests/keyboard-spelling');
mkdirSync(output, { recursive: true });
const directory = mkdtempSync(path.join(output, 'run-'));
const source = path.join(root, 'apps/desktop/ime/shared');
const binary = path.join(directory, windows ? 'spelling-test.exe' : 'spelling-test');
const library = windows ? path.join(sdk, 'lib/libArionRime.dll.a') : path.join(runtime, 'lib/librime.1.dylib');
const prism = path.join(runtime, 'data/build/pinyin_simp.prism.bin');
const files = ['spelling.h', 'spelling_options.h', 'spelling.cpp', 'personal_vocabulary.h', 'personal_vocabulary_bridge.h', 'personal_vocabulary.cpp', 'ranking_unicode.h', 'ranking_unicode.cpp', 'ranking_unicode_data.h', 'spelling_test.cpp'].map(name => path.join(source, name));
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const report = { platform: process.platform, arch: process.arch,
  scope: 'Direct spelling expansion and personal pronunciation projection; real candidate commits/host tests remain separate',
  sourceHashes: Object.fromEntries(files.map(file => [file, hash(file)])),
  library: { path: library, sha256: hash(library) },
  prism: { path: prism, sha256: hash(prism) }, commands: [], status: 'running' };
function run(command, args) {
  const result = cachedSpawn(command, args, { cwd: directory, encoding: 'utf8', timeout: 120000,
    env: { ...process.env, PATH: `${path.join(runtime, 'lib')}${path.delimiter}${process.env.PATH}` } });
  report.commands.push({ command, args, status: result.status, signal: result.signal,
    error: result.error?.message, stdout: result.stdout, stderr: result.stderr });
  process.stdout.write(result.stdout ?? '');
  if (result.error || result.status !== 0) throw new Error(`Native spelling check failed: ${result.stderr}`);
}
try {
  run(process.env.CXX ?? (windows ? 'g++' : 'clang++'), ['-std=c++17', '-O2', '-Wall', '-Wextra', '-Werror',
    '-DGLOG_USE_GLOG_EXPORT', ...(windows ? ['-DRIME_IMPORTS'] : []), '-isystem', path.join(sdk, 'include'),
    ...files.filter(file => file.endsWith('.cpp')), library,
    ...(!windows ? [`-Wl,-rpath,${path.dirname(library)}`] : []), '-o', binary]);
  run(binary, [directory, prism]);
  report.status = 'passed';
} catch (error) {
  report.status = 'failed';
  throw error;
} finally {
  const file = path.join(directory, 'report.json');
  writeFileSync(file, `${JSON.stringify(report, null, 2)}\n`);
  console.log(file);
}
