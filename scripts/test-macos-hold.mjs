// Validate the production listener lifecycle without changing system consent.
import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') throw new Error('Native hold listener checks require macOS.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const temporary = mkdtempSync(path.join(os.tmpdir(), 'arion-hold-test-'));
try {
  writeFileSync(path.join(temporary, '_cgo_export.h'), 'void arionHoldRaw(int, long long, unsigned long long, int);\n');
  const binary = path.join(temporary, 'hold-test');
  execFileSync('xcrun', ['clang', '-fobjc-arc', '-Wall', '-Wextra', '-Werror', '-I', temporary,
    '-framework', 'Cocoa', '-framework', 'ApplicationServices',
    path.join(root, 'apps/desktop/infrastructure/macos/tests/hold_test.m'), '-o', binary], { stdio: 'inherit' });
  execFileSync(binary, [], { stdio: 'inherit', timeout: 15000 });
} finally {
  rmSync(temporary, { recursive: true, force: true });
}
