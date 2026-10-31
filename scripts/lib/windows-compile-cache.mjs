// Reuse identical MinGW/Clang outputs across keyboard scripts.
// A miss, an unresolved local include, or a compiler failure compiles normally.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { copyFileSync, existsSync, mkdirSync, readFileSync, renameSync, rmSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');
const compilers = new Set(['gcc', 'g++', 'cc', 'c++', 'clang', 'clang++']);
const sourceExtensions = new Set(['.c', '.cc', '.cpp', '.cxx', '.h', '.hh', '.hpp', '.m', '.mm']);
const binaryExtensions = new Set(['.a', '.o', '.obj', '.lib', '.dll', '.dylib', '.so', '.exe']);
const identities = new Map();

export function compileCacheDirectory() {
  return process.env.ARION_WINDOWS_COMPILE_CACHE || path.join(root, '.workspace', 'windows-compile-cache');
}

export function isCompileCommand(command) {
  const base = path.basename(command).toLowerCase().replace(/\.exe$/, '');
  return compilers.has(base);
}

function outputPath(args) {
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === '-o') return args[index + 1] ? path.resolve(args[index + 1]) : null;
    if (arg.startsWith('-o') && arg.length > 2) return path.resolve(arg.slice(2));
  }
  return null;
}

function flagDirectories(args) {
  const directories = [];
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if ((arg === '-I' || arg === '-isystem') && args[index + 1]) directories.push(path.resolve(args[index + 1]));
    else if ((arg.startsWith('-I') || arg.startsWith('-isystem')) && arg.length > 2 && !arg.startsWith('-isystem')) directories.push(path.resolve(arg.slice(2)));
    else if (arg.startsWith('-isystem') && arg.length > 8) directories.push(path.resolve(arg.slice(8)));
  }
  return directories;
}

function resolveInclude(file, name, directories, quoted) {
  const candidates = quoted
    ? [path.resolve(path.dirname(file), name), ...directories.map(directory => path.resolve(directory, name))]
    : directories.map(directory => path.resolve(directory, name));
  return candidates.find(candidate => existsSync(candidate)) ?? null;
}

// null means a quoted include could not be read, so the result must not be cached.
export function compilationInputs(args) {
  const directories = flagDirectories(args);
  const files = [];
  const seen = new Set();
  function add(file) {
    const resolved = path.resolve(file);
    if (seen.has(resolved)) return true;
    if (!existsSync(resolved)) return false;
    seen.add(resolved);
    files.push(resolved);
    const extension = path.extname(resolved).toLowerCase();
    if (!sourceExtensions.has(extension)) return true;
    let text;
    try { text = readFileSync(resolved, 'utf8'); } catch { return false; }
    for (const match of text.matchAll(/^\s*#\s*include\s*(["<])([^">]+)[">]/gm)) {
      const quoted = match[1] === '"';
      const include = resolveInclude(resolved, match[2], directories, quoted);
      if (!include) {
        if (quoted) return false;
        continue;
      }
      if (!add(include)) return false;
    }
    return true;
  }
  for (let index = 0; index < args.length; index += 1) {
    const arg = args[index];
    if (arg === '-include' && args[index + 1] && !add(args[index + 1])) return null;
    if (!arg || arg.startsWith('-') || arg.startsWith('@')) continue;
    if (!existsSync(arg)) continue;
    const extension = path.extname(arg).toLowerCase();
    if (!sourceExtensions.has(extension) && !binaryExtensions.has(extension)) continue;
    if (!add(arg)) return null;
  }
  return files;
}

function compilerIdentity(command) {
  const key = path.resolve(command);
  if (!identities.has(key)) {
    const version = spawnSync(command, ['--version'], { encoding: 'utf8' });
    const target = spawnSync(command, ['-dumpmachine'], { encoding: 'utf8' });
    if (version.error || version.status !== 0 || target.error || target.status !== 0) return null;
    identities.set(key, `${version.stdout ?? ''}\n${target.stdout ?? ''}`);
  }
  return identities.get(key);
}

function cacheKey(command, args, files) {
  const hashedFiles = new Set(files.map(file => path.resolve(file)));
  const hash = createHash('sha256');
  // Paths of sources, objects, and -I directories are omitted. Their resolved
  // bytes are hashed instead, so a new temp directory can reuse a compile.
  hash.update('windows-compile-cache-v1\0');
  hash.update(compilerIdentity(command));
  let skipNext = false;
  for (const arg of args) {
    if (skipNext) { skipNext = false; continue; }
    if (arg === '-o' || arg === '-I' || arg === '-isystem') { skipNext = true; continue; }
    if (arg.startsWith('-o') && arg.length > 2) continue;
    if ((arg.startsWith('-I') || arg.startsWith('-isystem')) && arg.length > 2) continue;
    if (hashedFiles.has(path.resolve(arg))) continue;
    hash.update('\0').update(arg);
  }
  const contents = files.map(file => readFileSync(file));
  contents.sort(Buffer.compare);
  for (const content of contents) hash.update('\0').update(content);
  return hash.digest('hex');
}

function sidecarNames(output) {
  if (!output.toLowerCase().endsWith('.dll')) return [];
  const directory = path.dirname(output);
  const base = path.basename(output, path.extname(output));
  return [`${base}.dll.a`, `lib${base}.dll.a`].filter(name => /^[A-Za-z0-9._+-]+$/.test(name) && existsSync(path.join(directory, name)));
}

function emptyResult(options) {
  const text = Boolean(options.encoding);
  return { status: 0, signal: null, stdout: text ? '' : Buffer.alloc(0), stderr: text ? '' : Buffer.alloc(0) };
}

export function cachedSpawn(command, args, options = {}) {
  const output = isCompileCommand(command) ? outputPath(args) : null;
  const files = output && !args.some(arg => arg.startsWith('@')) ? compilationInputs(args) : null;
  const identity = output && files ? compilerIdentity(command) : null;
  if (!output || !files?.length || !identity) return spawnSync(command, args, options);
  const directory = path.join(compileCacheDirectory(), cacheKey(command, args, files));
  const manifestPath = path.join(directory, 'manifest.json');
  if (existsSync(manifestPath)) {
    const manifest = JSON.parse(readFileSync(manifestPath, 'utf8'));
    mkdirSync(path.dirname(output), { recursive: true });
    copyFileSync(path.join(directory, 'output'), output);
    for (const name of manifest.sidecars ?? []) {
      if (!/^[A-Za-z0-9._+-]+$/.test(name)) throw new Error(`Refusing compile cache sidecar ${name}`);
      copyFileSync(path.join(directory, 'sidecars', name), path.join(path.dirname(output), name));
    }
    process.stderr.write(`windows compile cache hit: ${path.basename(output)}\n`);
    return emptyResult(options);
  }
  const result = spawnSync(command, args, options);
  if (result.error || result.status !== 0 || !existsSync(output)) return result;
  const staging = `${directory}.partial-${process.pid}`;
  rmSync(staging, { recursive: true, force: true });
  mkdirSync(path.join(staging, 'sidecars'), { recursive: true });
  copyFileSync(output, path.join(staging, 'output'));
  const names = sidecarNames(output);
  for (const name of names) copyFileSync(path.join(path.dirname(output), name), path.join(staging, 'sidecars', name));
  writeFileSync(path.join(staging, 'manifest.json'), `${JSON.stringify({ sidecars: names })}\n`);
  mkdirSync(path.dirname(directory), { recursive: true });
  try { renameSync(staging, directory); }
  catch {
    rmSync(staging, { recursive: true, force: true });
  }
  return result;
}
