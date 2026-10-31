// Boundary tests use an explicit engine double. Real conversion/editor gates
// remain separate and must run with the pinned production Rime runtime.
import { execFileSync } from 'node:child_process';
import { cachedSpawn } from './lib/windows-compile-cache.mjs';
import { existsSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const platform = process.platform === 'darwin' ? 'darwin' : process.platform === 'win32' ? 'windows-amd64' : null;
if (!platform) throw new Error('Shared keyboard checks currently support macOS and Windows.');
const args = process.argv.slice(2);
if (args.length && (args.length !== 2 || args[0] !== '--include')) throw new Error('Usage: node scripts/test-shared-pinyin.mjs [--include HEADER_DIRECTORY]');
const include = args.length ? path.resolve(args[1]) : path.join(root, 'apps/desktop/resources', platform, 'keyboard/include');
if (!existsSync(path.join(include, 'rime_api.h'))) throw new Error('Stage the pinned keyboard SDK first.');
const output = path.join(root, '.workspace/tests/shared-pinyin');
mkdirSync(output, { recursive: true });
const binary = path.join(output, process.platform === 'win32' ? 'shared-pinyin-test.exe' : 'shared-pinyin-test');
const compiler = process.env.CC ?? (process.platform === 'win32' ? 'gcc' : 'clang');
const compiled = cachedSpawn(compiler, ['-std=c11', '-O2', '-Wall', '-Wextra', '-Werror', '-I', include,
  ...['pinyin.c', 'input.c', 'menu.c', 'pinyin_test.c'].map(file => path.join(root, 'apps/desktop/ime/shared', file)), '-o', binary],
{ cwd: root, stdio: 'inherit' });
if (compiled.error) throw compiled.error;
if (compiled.status !== 0) throw new Error(`${compiler} failed (${compiled.status})`);
execFileSync(binary, [], { cwd: root, stdio: 'inherit' });
