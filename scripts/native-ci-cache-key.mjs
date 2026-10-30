import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { appendFileSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export function cacheKey(root, files, identity) {
  const hash = createHash('sha256').update(JSON.stringify(identity));
  for (const file of [...files].sort()) {
    hash.update('\0').update(file).update('\0').update(readFileSync(path.join(root, file)));
  }
  return hash.digest('hex');
}

export function nativeInput(file) {
  return file === 'scripts/native-ci-cache-key.mjs' || file === 'apps/desktop/desktop.json' ||
    file.startsWith('apps/desktop/runtime/') || file.startsWith('apps/desktop/native/') || file.startsWith('apps/desktop/ime/shared/') ||
    (file.startsWith('scripts/') && !file.endsWith('.test.mjs') &&
      (/^scripts\/(?:stage-|.*runtime\.json$)/.test(file) || file.startsWith('scripts/lib/') || file.startsWith('scripts/cmake/')));
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (!['darwin', 'win32'].includes(process.platform)) throw new Error('Native CI caches require macOS or Windows.');
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
  const run = (command, args) => execFileSync(command, args, { cwd: root, encoding: 'utf8' }).trim();
  const files = run('git', ['ls-files', '-z']).split('\0').filter(Boolean);
  const identity = {
    version: 1, platform: process.platform, arch: process.arch, root,
    image: [process.env.ImageOS, process.env.ImageVersion],
    cmake: run('cmake', ['--version']),
    environment: Object.fromEntries(['CC', 'CXX', 'CFLAGS', 'CXXFLAGS', 'LDFLAGS', 'CGO_CFLAGS', 'CGO_CXXFLAGS', 'CGO_LDFLAGS', 'MACOSX_DEPLOYMENT_TARGET'].map(key => [key, process.env[key] ?? ''])),
    compiler: process.platform === 'darwin'
      ? [run('xcodebuild', ['-version']), run('xcrun', ['clang++', '--version']), run('xcrun', ['--sdk', 'macosx', '--show-sdk-version']), run('swift', ['--version'])]
      : [run(process.env.CXX ?? 'g++', ['--version']), run(process.env.CC ?? 'gcc', ['--version'])],
  };
  const native = `native-v1-${cacheKey(root, files.filter(nativeInput), identity)}`;
  const go = `standard-user-go-v1-${cacheKey(root, files.filter(file => /\.(?:go|mod|sum|h|c|cc|cpp|m|mm)$/.test(file) || file === 'go.work' || file.startsWith('scripts/') && file.endsWith('.ps1')), { ...identity, go: run('go', ['version']) })}`;
  console.log(JSON.stringify({ native, go, identity }, null, 2));
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `native=${native}\ngo=${go}\n`);
}
