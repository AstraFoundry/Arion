import assert from 'node:assert/strict';
import test from 'node:test';
import { cacheReport } from '../ci-cache-report.mjs';

test('cache reports name the exact hit, the miss, and a disabled cache', () => {
  assert.equal(cacheReport({ name: 'setup-go', exact: 'true' }), 'ci-cache setup-go: exact-hit');
  assert.equal(cacheReport({ name: 'native', exact: 'false', key: 'native-key', restore: 'native-prefix-' }), 'ci-cache native: no-exact-hit key=native-key restore=native-prefix-');
  assert.equal(cacheReport({ name: 'setup-go', exact: '' }), 'ci-cache setup-go: disabled');
  assert.throws(() => cacheReport({ name: 'Native', exact: 'true' }), /cache name/);
  assert.throws(() => cacheReport({ name: 'setup-go', exact: 'maybe' }), /exact flag/);
});
