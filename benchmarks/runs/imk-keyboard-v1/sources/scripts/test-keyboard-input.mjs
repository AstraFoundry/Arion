// Actual librime conversion and the production typing state against AppKit's text store.
import { execFileSync, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildIMELearning } from './lib/build-ime-learning.mjs';

if (process.platform !== 'darwin') throw new Error('Keyboard integration tests require macOS.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const options = new Map();
for (let i = 2; i < process.argv.length; i += 2) {
  const name = process.argv[i], value = process.argv[i + 1];
  if (!['--data', '--report'].includes(name) || !value || options.has(name)) {
    throw new Error('Usage: node scripts/test-keyboard-input.mjs [--data PATH] [--report PATH]');
  }
  options.set(name, path.resolve(value));
}
const data = options.get('--data') ?? path.join(root, 'apps/desktop/resources/darwin/keyboard/data');
const report = options.get('--report') ?? path.join(root, '.workspace/tests/keyboard-input.json');
const runtime = path.join(root, 'apps/desktop/resources/darwin/keyboard');
for (const required of [path.join(data, 'build/arion_pinyin.schema.yaml'),
  path.join(runtime, 'include/rime_api.h'), path.join(runtime, 'lib/librime.1.dylib')]) {
  if (!existsSync(required)) throw new Error(`Required pinned keyboard artifact is missing: ${required}`);
}
mkdirSync(path.dirname(report), { recursive: true });
const temporary = mkdtempSync(path.join(os.tmpdir(), 'arion-keyboard-test-'));
let status = 2;
try {
  const source = path.join(root, 'apps/desktop/ime/macos');
  const arch = process.arch === 'arm64' ? 'arm64' : 'x86_64';
  const learning = buildIMELearning({ root, outputDir: temporary, arch, testing: true });
  const objects = ['keyboard.m', 'typing.m', 'tests/typing_test.m'].map((file, i) => {
    const object = path.join(temporary, `typing-${i}.o`);
    execFileSync('xcrun', [file.startsWith('tests/') ? 'clang++' : 'clang',
      ...(file.startsWith('tests/') ? ['-x', 'objective-c++', '-std=c++17'] : []),
      '-target', learning.target, '-fobjc-arc', '-Wall', '-Wextra', '-Werror',
      '-DARION_LEARNING_TESTING', '-DGLOG_USE_GLOG_EXPORT', '-I', path.join(runtime, 'include'),
      '-isystem', path.join(root, '.workspace/keyboard-runtime/sdk/include'),
      '-c', path.join(source, file), '-o', object], { cwd: root, stdio: 'inherit' });
    return object;
  });
  const binary = path.join(temporary, 'typing-test');
  const library = path.join(runtime, 'lib');
  execFileSync('xcrun', [learning.linker, '-target', learning.target, ...objects, ...learning.objects,
    path.join(library, 'librime.1.dylib'), '-Xlinker', '-rpath', '-Xlinker', library,
    '-framework', 'Cocoa', '-framework', 'InputMethodKit', ...learning.linkArgs, '-o', binary],
  { cwd: root, stdio: 'inherit' });
  const result = spawnSync(binary, [data, path.join(temporary, 'user'), report], { cwd: root, stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.signal) throw new Error(`Keyboard test terminated by ${result.signal}`);
  status = result.status ?? 2;
  process.stdout.write(`Keyboard test report: ${report}\n`);
} finally {
  rmSync(temporary, { recursive: true, force: true });
}
process.exitCode = status;
