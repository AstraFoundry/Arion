import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const directory = mkdtempSync(path.join(os.tmpdir(), 'arion-english-query-'));
try {
  const binary = path.join(directory, process.platform === 'win32' ? 'query.exe' : 'query');
  const compiler = process.env.CXX ?? (process.platform === 'win32' ? 'g++' : 'clang++');
  execFileSync(compiler, ['-std=c++17', '-DARION_ENGLISH_TESTING', '-O2', '-Wall', '-Wextra', '-Werror',
    ...['english_lexicon.cpp', 'english_lexicon_test.cpp'].map(name => path.join(root, 'apps/desktop/ime/shared', name)),
    '-o', binary], { cwd: root, stdio: 'inherit', windowsHide: true });
  execFileSync(binary, [path.join(directory, 'fixture.lexicon')], { stdio: 'inherit', windowsHide: true });
} finally { rmSync(directory, { recursive: true, force: true }); }
