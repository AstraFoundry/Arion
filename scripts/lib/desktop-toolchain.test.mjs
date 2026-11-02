import test from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { desktopToolchainProblems } from './desktop-toolchain.mjs';

test('preflight reports all missing tools and the active Node version together', () => {
  const problems = desktopToolchainProblems({ nodeVersion: '22.23.2', run: () => ({ error: { code: 'ENOENT' } }) });
  assert.equal(problems.length, 5);
  assert.match(problems[0], /Node 24.*22\.23\.2/);
  assert.match(problems.find(problem => problem.startsWith('cargo ')), /not found on PATH.*rustup/);
});

test('installed but failing tools retain their diagnostic rather than reporting missing', () => {
  const problems = desktopToolchainProblems({ nodeVersion: '24.0.0', run: command => command === 'cargo'
    ? { status: 1, stderr: 'no default toolchain configured' }
    : { status: 0 } });
  assert.equal(problems.length, 1);
  assert.match(problems[0], /cargo could not run: no default toolchain configured/);
  assert.deepEqual(desktopToolchainProblems({ nodeVersion: '24.0.0', run: () => ({ status: 0 }) }), []);
});

test('desktop entry points stop before signing and resource staging with an empty PATH', {
  skip: !['darwin', 'win32'].includes(process.platform) || (process.platform === 'darwin' && process.arch !== 'arm64'),
}, () => {
  for (const [script, args] of [['desktop-command', ['dev']], ['stage-desktop-resources', []]]) {
    const result = spawnSync(process.execPath, [fileURLToPath(new URL(`../${script}.mjs`, import.meta.url)), ...args], {
      encoding: 'utf8', env: { ...process.env, PATH: '' },
    });
    assert.equal(result.status, 1, result.stderr);
    assert.match(result.stderr, /Desktop toolchain check failed before resource preparation/);
    assert.match(result.stderr, /cargo was not found on PATH/);
    assert.doesNotMatch(result.stdout, /Center profile|Staged/);
    assert.doesNotMatch(result.stderr, /node:internal|spawnSync cargo ENOENT/);
  }
});
