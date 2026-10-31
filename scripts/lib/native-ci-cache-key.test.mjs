import assert from 'node:assert/strict';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { cacheKey, nativeInput, runnerCacheIdentity, scopedCacheNames } from '../native-ci-cache-key.mjs';

test('cache identity changes with source, build settings and toolchain, independently of file order', t => {
  const root = mkdtempSync(path.join(os.tmpdir(), 'arion-ci-cache-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  writeFileSync(path.join(root, 'source'), 'original');
  writeFileSync(path.join(root, 'flags'), '-O2');
  const identity = { compiler: 'compiler 1', arch: 'arm64' };
  const original = cacheKey(root, ['source', 'flags'], identity);
  assert.equal(cacheKey(root, ['flags', 'source'], identity), original);
  assert.notEqual(cacheKey(root, ['source', 'flags'], { ...identity, compiler: 'compiler 2' }), original);
  assert.notEqual(cacheKey(root, ['source', 'flags'], { ...identity, arch: 'x64' }), original);
  writeFileSync(path.join(root, 'source'), 'patched');
  assert.notEqual(cacheKey(root, ['source', 'flags'], identity), original);
  writeFileSync(path.join(root, 'source'), 'original');
  writeFileSync(path.join(root, 'flags'), '-O3');
  assert.notEqual(cacheKey(root, ['source', 'flags'], identity), original);
  assert.throws(() => cacheKey(root, ['missing'], identity), /ENOENT/);
});

test('native inputs cover runtime pins, build scripts, patches and ASR sources without UI churn', () => {
  for (const file of ['scripts/asr-runtime.json', 'scripts/llm-runtime.json', 'scripts/keyboard-runtime.json',
    'scripts/stage-llm-runtime.mjs', 'scripts/lib/keyboard-correction-patch.mjs', 'scripts/cmake/windows-llm.cmake',
    'apps/desktop/native/mlx-asr/Package.resolved', 'apps/desktop/native/mlx-asr/Sources/main.swift',
    'apps/desktop/ime/shared/keyboard-data/arion_pinyin.schema.yaml', 'apps/desktop/desktop.json']) {
    assert.equal(nativeInput(file), true, file);
  }
  for (const file of ['apps/desktop/ui/app/app.go', 'apps/site/app/page.tsx', 'scripts/lib/native-ci-cache-key.test.mjs']) {
    assert.equal(nativeInput(file), false, file);
  }
});

test('exact cache keys omit checkout path and ImageVersion, and restore stays on the same toolchain', t => {
  const root = mkdtempSync(path.join(os.tmpdir(), 'arion-ci-cache-'));
  const other = mkdtempSync(path.join(os.tmpdir(), 'arion-ci-cache-other-'));
  t.after(() => {
    rmSync(root, { recursive: true, force: true });
    rmSync(other, { recursive: true, force: true });
  });
  writeFileSync(path.join(root, 'source'), 'original');
  writeFileSync(path.join(other, 'source'), 'original');
  const identity = runnerCacheIdentity({
    platform: 'darwin', arch: 'arm64', imageOS: 'macos26', cmake: 'cmake 4',
    environment: { CC: '' }, compiler: ['Apple clang 17'],
  });
  assert.equal(Object.hasOwn(identity, 'root'), false);
  assert.equal(Object.hasOwn(identity, 'image'), false);
  assert.equal(Object.hasOwn(identity, 'ImageVersion'), false);
  const content = cacheKey(root, ['source'], identity);
  assert.equal(cacheKey(other, ['source'], identity), content);
  const names = scopedCacheNames('native', identity, content);
  assert.equal(names.key.startsWith(names.restore), true);
  assert.match(names.restore, /^native-v2-darwin-arm64-macos26-[a-f0-9]{64}-$/);
  writeFileSync(path.join(root, 'source'), 'patched');
  const changed = scopedCacheNames('native', identity, cacheKey(root, ['source'], identity));
  assert.equal(changed.restore, names.restore);
  assert.notEqual(changed.key, names.key);
  const compiler = runnerCacheIdentity({ ...identity, compiler: ['Apple clang 18'] });
  assert.notEqual(scopedCacheNames('native', compiler, content).restore, names.restore);
  assert.throws(() => scopedCacheNames('native', { ...identity, root: '/Users/runner/work/Arion/Arion' }, content), /root/);
  assert.throws(() => scopedCacheNames('native', { ...identity, ImageVersion: '20261028.1' }, content), /ImageVersion/);
  assert.throws(() => scopedCacheNames('native', { ...identity, image: ['macos26', '20261028.1'] }, content), /image/);
});
