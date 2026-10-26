import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, renameSync, rmSync, statSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

// A local path, a cache hit and a download have the same size and digest requirements.
export function pinnedFile(spec) {
  const valid = candidate => statSync(candidate).size === spec.sizeBytes && createHash('sha256').update(readFileSync(candidate)).digest('hex') === spec.sha256;
  if (spec.path) {
    const file = path.resolve(root, spec.path);
    if (!existsSync(file) || !valid(file)) throw new Error(`Pinned local file size or SHA-256 mismatch: ${spec.path}`);
    return file;
  }
  const cache = path.join(process.env.XDG_CACHE_HOME ?? path.join(os.homedir(), '.cache'), 'arion', 'llm-runtime');
  mkdirSync(cache, { recursive: true });
  const file = path.join(cache, spec.sha256);
  if (existsSync(file) && valid(file)) return file;
  const partial = `${file}.partial`;
  try {
    execFileSync('curl', ['--fail', '--location', '--silent', '--show-error', '--max-filesize', String(spec.sizeBytes), '--output', partial, spec.url], { stdio: 'inherit' });
    if (!valid(partial)) throw new Error(`Pinned file size or SHA-256 mismatch: ${spec.url}`);
    renameSync(partial, file);
  } finally {
    rmSync(partial, { force: true });
  }
  return file;
}
