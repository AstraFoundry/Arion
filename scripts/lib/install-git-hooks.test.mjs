import assert from 'node:assert/strict';
import { mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { installGitHooks } from '../install-git-hooks.mjs';

test('hook install points git at .githooks and stays out of GitHub Actions', t => {
  const rootDir = mkdtempSync(path.join(os.tmpdir(), 'arion-hooks-'));
  t.after(() => rmSync(rootDir, { recursive: true, force: true }));
  const calls = [];
  const exists = target => target === path.join(rootDir, '.git');
  const run = (command, args, options) => calls.push({ command, args, options });
  assert.equal(installGitHooks({ env: {}, rootDir, exists, run }), true);
  assert.deepEqual(calls, [{ command: 'git', args: ['config', 'core.hooksPath', '.githooks'], options: { cwd: rootDir, stdio: 'inherit' } }]);
  calls.length = 0;
  assert.equal(installGitHooks({ env: { GITHUB_ACTIONS: 'true' }, rootDir, run: () => calls.push('run') }), false);
  assert.deepEqual(calls, []);
  assert.throws(() => installGitHooks({ env: {}, rootDir, exists: () => false, run: () => {} }), /no \.git/);
});
