// Pull-request path gate for the native shell. Main and release always run it.
import { execFileSync } from 'node:child_process';
import { appendFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

// Marketing site, docs, and the root product READMEs. Every other path,
// including desktop, IME, Go modules, scripts, and workflow files, runs native.
const documentationPrefixes = ['apps/site/', 'docs/'];
const documentationFiles = new Set(['README.md', 'README.zh-CN.md', 'README.ja.md']);

export function isDocumentationOnlyPath(file) {
  if (typeof file !== 'string') return false;
  const normalized = file.replaceAll('\\', '/');
  if (!normalized || normalized.startsWith('/') || normalized.split('/').includes('..')) return false;
  if (documentationFiles.has(normalized)) return true;
  return documentationPrefixes.some(prefix => normalized.startsWith(prefix));
}

export function nativeShellRequired(files) {
  if (!Array.isArray(files) || files.length === 0) return true;
  return files.some(file => !isDocumentationOnlyPath(file));
}

export function changedFiles(base, cwd = root) {
  if (!/^[0-9a-f]{40}$/.test(base)) throw new Error('BASE_SHA must be a 40-character commit id.');
  execFileSync('git', ['fetch', '--no-tags', '--depth=1', 'origin', base], { cwd, stdio: 'inherit' });
  const diff = execFileSync('git', ['diff', '-z', '--name-only', base, 'HEAD'], { cwd });
  return diff.toString('utf8').split('\0').filter(Boolean);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const base = process.env.BASE_SHA;
  if (!base) throw new Error('ci-native-scope.mjs requires BASE_SHA.');
  const native = nativeShellRequired(changedFiles(base));
  const line = `native=${native ? 'true' : 'false'}\n`;
  if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, line);
  console.log(line.trim());
}
