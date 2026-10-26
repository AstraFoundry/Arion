// Compile the current production engine and the reference-free existing driver.
import { execFileSync, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdirSync, readFileSync, writeFileSync, copyFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { buildIMELearning } from '../../../scripts/lib/build-ime-learning.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const workspace = path.resolve(process.argv[2]);
const out = path.join(workspace, 'native-build');
if (process.platform !== 'darwin' || process.arch !== 'arm64') throw new Error('Native Apple silicon macOS required');
mkdirSync(out);
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const runtime = path.join(root, 'apps/desktop/resources/darwin/keyboard');
const native = path.join(root, 'apps/desktop/ime/macos');
const files = ['keyboard.m', 'ranking.m', 'learning.mm', 'learning_crypto.swift', 'keyboard.h', 'ranking.h', 'learning.h'].map(file => path.join(native, file));
const driver = path.join(root, 'benchmarks/reproduction/imk-quality-v1/driver.mm');
const commands = [];
const compile = args => {
  const result = spawnSync('xcrun', args, { cwd: root, encoding: 'utf8' });
  commands.push({ args: ['xcrun', ...args], code: result.status, stdout: result.stdout, stderr: result.stderr });
  writeFileSync(path.join(out, 'commands.json'), JSON.stringify(commands, null, 2));
  if (result.status !== 0) throw new Error('Native compilation failed; retained commands.json');
};
const learning = buildIMELearning({ root, outputDir: out, arch: 'arm64', testing: true });
const objects = [];
for (const [i, source] of [path.join(native, 'keyboard.m'), path.join(native, 'ranking.m'), driver].entries()) {
  const cxx = source.endsWith('.mm');
  const object = path.join(out, `source-${i}.o`);
  compile([cxx ? 'clang++' : 'clang', ...(cxx ? ['-std=c++17'] : []), '-target', learning.target,
    '-O2', '-fobjc-arc', '-Wall', '-Wextra', '-Werror', '-DARION_LEARNING_TESTING', '-DGLOG_USE_GLOG_EXPORT',
    '-I', native, '-I', path.join(runtime, 'include'), '-isystem', path.join(root, '.workspace/keyboard-runtime/sdk/include'),
    '-c', source, '-o', object]);
  objects.push(object);
}
compile([learning.linker, '-target', learning.target, ...objects, ...learning.objects, path.join(runtime, 'lib/librime.1.dylib'),
  '-Xlinker', '-rpath', '-Xlinker', path.join(runtime, 'lib'), '-framework', 'Cocoa', '-framework', 'InputMethodKit', ...learning.linkArgs,
  '-o', path.join(out, 'driver')]);
const sourceHashes = {};
for (const source of [...files, driver, path.join(root, 'scripts/lib/build-ime-learning.mjs')]) {
  const name = path.relative(root, source), destination = path.join(out, 'sources', name);
  mkdirSync(path.dirname(destination), { recursive: true }); copyFileSync(source, destination); sourceHashes[name] = hash(source);
}
writeFileSync(path.join(out, 'identities.json'), JSON.stringify({ sourceHashes, driverSha256: hash(path.join(out, 'driver')),
  compiler: execFileSync('xcrun', ['clang', '--version'], { encoding: 'utf8' }),
  swift: execFileSync('xcrun', ['swiftc', '--version'], { encoding: 'utf8' }) }, null, 2));
