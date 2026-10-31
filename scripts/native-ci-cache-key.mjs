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

// Identity is content and toolchain only. The checkout path is not included:
// hosted runners use a stable workspace, and staging reconfigures absolute
// paths before it trusts a restored tree. ImageVersion changes on unrelated
// runner rolls; compiler and SDK versions below still miss the cache.
export function runnerCacheIdentity({ platform, arch, imageOS, cmake, environment, compiler, go } = {}) {
  const identity = { version: 2, platform, arch, imageOS, cmake, environment, compiler };
  if (go !== undefined) identity.go = go;
  return identity;
}

function scopeToken(value, label) {
  const token = String(value ?? '');
  if (!/^[A-Za-z0-9._-]+$/.test(token)) throw new Error(`Cache scope ${label} is not a safe token.`);
  return token;
}

export function scopedCacheNames(label, identity, contentHash) {
  if (!/^[a-z0-9-]+$/.test(label)) throw new Error(`Invalid cache label: ${label}`);
  for (const key of ['root', 'image', 'ImageVersion']) {
    if (Object.hasOwn(identity, key)) throw new Error(`Cache identity must not include ${key}.`);
  }
  if (!/^[a-f0-9]{64}$/.test(contentHash)) throw new Error('Cache content hash must be SHA-256 hex.');
  const platform = scopeToken(identity.platform, 'platform');
  const arch = scopeToken(identity.arch, 'arch');
  const imageOS = scopeToken(identity.imageOS, 'imageOS');
  const toolchain = cacheKey('.', [], identity);
  const restore = `${label}-v2-${platform}-${arch}-${imageOS}-${toolchain}-`;
  return { key: `${restore}${contentHash}`, restore };
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  if (!['darwin', 'win32'].includes(process.platform)) throw new Error('Native CI caches require macOS or Windows.');
  const imageOS = process.env.ImageOS;
  if (!imageOS) throw new Error('Native CI cache keys require ImageOS.');
  const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
  const run = (command, args) => execFileSync(command, args, { cwd: root, encoding: 'utf8' }).trim();
  const files = run('git', ['ls-files', '-z']).split('\0').filter(Boolean);
  const identity = runnerCacheIdentity({
    platform: process.platform,
    arch: process.arch,
    imageOS,
    cmake: run('cmake', ['--version']),
    environment: Object.fromEntries(['CC', 'CXX', 'CFLAGS', 'CXXFLAGS', 'LDFLAGS', 'CGO_CFLAGS', 'CGO_CXXFLAGS', 'CGO_LDFLAGS', 'MACOSX_DEPLOYMENT_TARGET'].map(key => [key, process.env[key] ?? ''])),
    compiler: process.platform === 'darwin'
      ? [run('xcodebuild', ['-version']), run('xcrun', ['clang++', '--version']), run('xcrun', ['--sdk', 'macosx', '--show-sdk-version']), run('swift', ['--version'])]
      : [run(process.env.CXX ?? 'g++', ['--version']), run(process.env.CC ?? 'gcc', ['--version'])],
  });
  const native = scopedCacheNames('native', identity, cacheKey(root, files.filter(nativeInput), identity));
  const goIdentity = runnerCacheIdentity({ ...identity, go: run('go', ['version']) });
  const go = scopedCacheNames('standard-user-go', goIdentity, cacheKey(root, files.filter(file => /\.(?:go|mod|sum|h|c|cc|cpp|m|mm)$/.test(file) || file === 'go.work' || file.startsWith('scripts/') && file.endsWith('.ps1')), goIdentity));
  console.log(JSON.stringify({ native, go, identity, imageVersion: process.env.ImageVersion ?? '' }, null, 2));
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `native=${native.key}\nnative-restore=${native.restore}\ngo=${go.key}\ngo-restore=${go.restore}\n`);
}
