import assert from 'node:assert/strict';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { cacheKey, nativeInput } from '../native-ci-cache-key.mjs';

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
