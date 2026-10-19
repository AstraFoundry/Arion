// Exercises production composition against a real AppKit text store on macOS.
import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.platform !== 'darwin') throw new Error('InputMethodKit tests require macOS.');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const source = path.join(root, 'apps/desktop/ime/macos');
const temporary = mkdtempSync(path.join(os.tmpdir(), 'arion-imk-test-'));
try {
  const binary = path.join(temporary, 'composition-test');
  execFileSync('clang', ['-fobjc-arc', '-Wall', '-Wextra', '-Werror', '-framework', 'Cocoa', '-framework', 'InputMethodKit',
    path.join(source, 'composition.m'), path.join(source, 'tests/composition_test.m'), '-o', binary], { stdio: 'inherit' });
  execFileSync(binary, [], { stdio: 'inherit' });
} finally {
  rmSync(temporary, { recursive: true, force: true });
}
