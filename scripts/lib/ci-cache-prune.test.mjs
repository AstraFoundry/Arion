import assert from 'node:assert/strict';
import test from 'node:test';
import { obsoleteNativeCaches } from '../ci-cache-prune.mjs';

test('retain the newest entries per platform and cache family; preserve unrelated and PR caches', () => {
  const caches = [];
  for (const family of ['native-v2-darwin-arm64-macos26', 'native-v2-win32-x64-win22', 'standard-user-go-v2-win32-x64-win22']) {
    for (let id = 1; id <= 4; id += 1) caches.push({ id, key: `${family}-toolchain-${id}`, ref: 'refs/heads/main', created_at: `2026-01-0${id}T00:00:00Z` });
  }
  caches.push({ id: 99, key: 'setup-go-macOS-arm64-go-1.27.3', ref: 'refs/heads/main' });
  caches.push({ id: 100, key: 'native-v2-darwin-arm64-macos26-toolchain-old', ref: 'refs/pull/1/merge' });
  assert.deepEqual(obsoleteNativeCaches(caches).map(cache => cache.id), [2, 1, 2, 1, 2, 1]);
  assert.throws(() => obsoleteNativeCaches(caches, 0), /positive/);
});


test('one Windows shard cannot evict the other shard compiler cache', () => {
  const caches = [];
  for (const suite of ['desktop', 'keyboard']) {
    for (let id = 1; id <= 3; id += 1) caches.push({ id, key: `standard-user-go-v2-win32-x64-win22-toolchain-${suite}-${String(id).repeat(64)}`, ref: 'refs/heads/main', created_at: `2026-01-0${id}T00:00:00Z` });
  }
  assert.deepEqual(obsoleteNativeCaches(caches).map(cache => cache.id), [1, 1]);
});
