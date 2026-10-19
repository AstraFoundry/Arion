// Exercises the production encrypted userdb with synthetic keys and isolated
// files. This command never provisions artifacts or accesses the user's Keychain.
import { execFileSync } from 'node:child_process';
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildIMELearning } from './lib/build-ime-learning.mjs';

if (process.platform !== 'darwin') throw new Error('Encrypted keyboard learning tests require macOS.');
if (process.argv.length !== 2) throw new Error('Usage: node scripts/test-keyboard-learning.mjs');
const architectures = { arm64: 'arm64', x64: 'x86_64' };
const arch = architectures[process.arch];
if (!arch) throw new Error(`Unsupported macOS architecture: ${process.arch}`);
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const runtime = path.join(root, 'apps/desktop/resources/darwin/keyboard');
const sdkInclude = path.join(root, '.workspace/keyboard-runtime/sdk/include');
const library = path.join(runtime, 'lib/librime.1.dylib');
for (const required of [library, path.join(runtime, 'include/rime_api.h'),
  path.join(sdkInclude, 'rime_api.h'), path.join(sdkInclude, 'rime/build_config.h'),
  path.join(sdkInclude, 'rime/dict/db.h'), path.join(sdkInclude, 'glog/export.h'),
  path.join(sdkInclude, 'boost/version.hpp')]) {
  if (!existsSync(required)) {
    throw new Error(`Required pinned keyboard artifact is missing: ${required}. Stage the keyboard runtime explicitly before running this test.`);
  }
}
const config = JSON.parse(readFileSync(path.join(root, 'apps/desktop/desktop.json'), 'utf8'));
const temporary = mkdtempSync(path.join(os.tmpdir(), 'arion-learning-test-'));
try {
  const learning = buildIMELearning({ root, outputDir: temporary, arch, testing: true,
    sdkInclude, minimumSystemVersion: config.macos.minimumSystemVersion });
  const object = path.join(temporary, 'learning-tests.o');
  const binary = path.join(temporary, 'learning-tests');
  const run = args => execFileSync('xcrun', args, { cwd: root, stdio: 'inherit' });
  run(['clang++', '-std=c++17', '-fobjc-arc', '-O2', '-Wall', '-Wextra', '-Werror',
    '-target', learning.target, '-DGLOG_USE_GLOG_EXPORT', '-DARION_LEARNING_TESTING',
    '-isystem', sdkInclude, '-c', path.join(root, 'apps/desktop/ime/macos/learning_tests.mm'), '-o', object]);
  run([learning.linker, '-target', learning.target, ...learning.objects, object, library,
    '-Xlinker', '-rpath', '-Xlinker', path.dirname(library), ...learning.linkArgs, '-o', binary]);
  // Foundation's synthetic fixtures remain under this runner's temporary root,
  // including when a source assertion fails before its normal cleanup.
  execFileSync(binary, [], { cwd: root, stdio: 'inherit', env: { ...process.env, TMPDIR: `${temporary}${path.sep}` } });
} finally {
  rmSync(temporary, { recursive: true, force: true });
}
