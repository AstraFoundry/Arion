import assert from 'node:assert/strict';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { cachedSpawn, compilationInputs } from './windows-compile-cache.mjs';

function compiler(directory) {
  const file = path.join(directory, 'g++');
  const count = path.join(directory, 'count');
  writeFileSync(count, '0');
  writeFileSync(file, `#!/bin/sh
count="${count}"
if [ "$1" = "--version" ] || [ "$1" = "-dumpmachine" ]; then
  echo fake-compiler
  exit 0
fi
out=
prev=
src=
for arg in "$@"; do
  if [ "$prev" = "-o" ]; then out=$arg; fi
  case "$arg" in
    *.cpp|*.c|*.h) src=$arg ;;
  esac
  prev=$arg
done
printf '%s' "$(($(cat "$count") + 1))" > "$count"
: > "$out"
for arg in "$@"; do
  case "$arg" in
    *.cpp|*.c|*.h) cat "$arg" >> "$out" ;;
  esac
done
`, { mode: 0o755 });
  return { file, count };
}

test('a repeated compile reuses the output and a header change compiles again', t => {
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-compile-cache-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const cache = path.join(directory, 'cache');
  const { file, count } = compiler(directory);
  const header = path.join(directory, 'widget.h');
  const source = path.join(directory, 'widget.cpp');
  writeFileSync(header, '#define VALUE 1\n');
  writeFileSync(source, '#include "widget.h"\nint value = VALUE;\n');
  const output = path.join(directory, 'widget.o');
  const args = ['-std=c++17', '-I', directory, '-c', source, '-o', output];
  const options = { env: { ...process.env, ARION_WINDOWS_COMPILE_CACHE: cache }, encoding: 'utf8' };
  // compileCacheDirectory reads the environment of this process, not spawn options.
  const previous = process.env.ARION_WINDOWS_COMPILE_CACHE;
  process.env.ARION_WINDOWS_COMPILE_CACHE = cache;
  t.after(() => { if (previous === undefined) delete process.env.ARION_WINDOWS_COMPILE_CACHE; else process.env.ARION_WINDOWS_COMPILE_CACHE = previous; });
  assert.equal(cachedSpawn(file, args, options).status, 0);
  assert.equal(readFileSync(count, 'utf8'), '1');
  const first = readFileSync(output);
  rmSync(output);
  assert.equal(cachedSpawn(file, [args[0], '-I', directory, '-c', source, '-o', path.join(directory, 'other.o')], options).status, 0);
  assert.equal(readFileSync(count, 'utf8'), '1');
  assert.deepEqual(readFileSync(path.join(directory, 'other.o')), first);
  const copyDir = path.join(directory, 'copy');
  mkdirSync(copyDir);
  writeFileSync(path.join(copyDir, 'widget.h'), readFileSync(header));
  writeFileSync(path.join(copyDir, 'widget.cpp'), readFileSync(source));
  assert.equal(cachedSpawn(file, ['-std=c++17', '-I', copyDir, '-c', path.join(copyDir, 'widget.cpp'), '-o', path.join(copyDir, 'widget.o')], options).status, 0);
  assert.equal(readFileSync(count, 'utf8'), '1');
  writeFileSync(header, '#define VALUE 2\n');
  assert.equal(cachedSpawn(file, args, options).status, 0);
  assert.equal(readFileSync(count, 'utf8'), '2');
});

test('an unresolved quoted include is never cached', t => {
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-compile-cache-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  process.env.ARION_WINDOWS_COMPILE_CACHE = path.join(directory, 'cache');
  const { file, count } = compiler(directory);
  const source = path.join(directory, 'missing.cpp');
  writeFileSync(source, '#include "missing.h"\n');
  const args = ['-c', source, '-o', path.join(directory, 'missing.o')];
  assert.equal(compilationInputs(args), null);
  assert.equal(cachedSpawn(file, args, { encoding: 'utf8' }).status, 0);
  assert.equal(cachedSpawn(file, args, { encoding: 'utf8' }).status, 0);
  assert.equal(readFileSync(count, 'utf8'), '2');
});

test('running a test binary is not treated as a compile', t => {
  const directory = mkdtempSync(path.join(tmpdir(), 'arion-compile-cache-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const program = path.join(directory, 'fixture');
  writeFileSync(program, '#!/bin/sh\necho ran\n', { mode: 0o755 });
  const result = cachedSpawn(program, ['-o', 'not-a-compiler'], { encoding: 'utf8' });
  assert.equal(result.status, 0);
  assert.match(result.stdout, /ran/);
});
