import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';
import { buildParallelism } from '../build-parallelism.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

test('build jobs follow the host and never exceed it', () => {
  assert.equal(buildParallelism({}, 4), 4);
  assert.equal(buildParallelism({ ARION_BUILD_JOBS: '2' }, 4), 2);
  assert.equal(buildParallelism({ ARION_KEYBOARD_BUILD_JOBS: '6' }, 4), 4);
  assert.equal(buildParallelism({ ARION_BUILD_JOBS: '2', ARION_KEYBOARD_BUILD_JOBS: '6' }, 4), 2);
  assert.equal(buildParallelism({}, 3), 3);
  assert.throws(() => buildParallelism({ ARION_BUILD_JOBS: '0' }, 4), /positive integer/);
  assert.throws(() => buildParallelism({ ARION_BUILD_JOBS: 'many' }, 4), /positive integer/);
  assert.throws(() => buildParallelism({}, 0), /unavailable/);
});

test('native compile scripts no longer pin a job count above the runner', () => {
  for (const file of [
    'scripts/stage-asr-runtime.mjs',
    'scripts/stage-llm-runtime.mjs',
    'scripts/stage-keyboard-runtime-windows.mjs',
    'scripts/lib/keyboard-macos-runtime.mjs',
  ]) {
    const source = readFileSync(path.join(root, file), 'utf8');
    assert.match(source, /buildParallelism/);
    assert.doesNotMatch(source, /--parallel',\s*'[0-9]+'/);
    assert.doesNotMatch(source, /--parallel",\s*"[0-9]+"/);
  }
});
