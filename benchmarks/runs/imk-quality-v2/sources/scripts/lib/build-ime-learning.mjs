// Builds the private Rime adapter and the public CryptoKit C boundary. Callers
// link the returned objects with swiftc so Apple Swift runtime linkage is exact.
import { execFileSync } from 'node:child_process';
import { mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function buildIMELearning({
  root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..'),
  outputDir,
  arch,
  minimumSystemVersion = '14.0',
  testing = false,
  sdkInclude = path.join(root, '.workspace/keyboard-runtime/sdk/include'),
}) {
  if (!outputDir || !['arm64', 'x86_64'].includes(arch)) {
    throw new Error('Learning build requires an output directory and an explicit macOS architecture.');
  }
  mkdirSync(outputDir, { recursive: true });
  const source = path.join(root, 'apps/desktop/ime/macos');
  const target = `${arch}-apple-macos${minimumSystemVersion}`;
  const adapter = path.join(outputDir, `learning-${arch}.o`);
  const crypto = path.join(outputDir, `learning-crypto-${arch}.o`);
  const run = args => execFileSync('xcrun', args, { cwd: root, stdio: 'inherit' });
  run(['clang++', '-std=c++17', '-fobjc-arc', '-O2', '-target', target,
    '-DGLOG_USE_GLOG_EXPORT', ...(testing ? ['-DARION_LEARNING_TESTING'] : []),
    '-I', sdkInclude,
    '-c', path.join(source, 'learning.mm'), '-o', adapter]);
  run(['swiftc', '-parse-as-library', '-emit-object', '-O', '-target', target,
    path.join(source, 'learning_crypto.swift'), '-o', crypto]);
  return {
    objects: [adapter, crypto],
    linker: 'swiftc',
    target,
    linkArgs: ['-Xlinker', '-lc++', '-framework', 'Foundation', '-framework', 'Security'],
  };
}
