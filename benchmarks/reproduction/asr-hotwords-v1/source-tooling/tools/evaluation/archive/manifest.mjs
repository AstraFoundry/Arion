import fs from 'node:fs/promises';
import path from 'node:path';
import crypto from 'node:crypto';

export const sha256 = bytes => crypto.createHash('sha256').update(bytes).digest('hex');
function safeRelative(value) {
  return typeof value === 'string' && value.length > 0 && !path.posix.isAbsolute(value) &&
    !value.includes('\\') && !value.split('/').some(part => part === '..' || part === '.' || part === '') && !/^[A-Za-z]:/.test(value);
}
export async function openArchive(manifestFile, root) {
  root = path.resolve(root);
  const manifest = JSON.parse(await fs.readFile(manifestFile, 'utf8'));
  if (manifest.version !== 1 || manifest.kind !== 'arion-benchmark-archive' || !Array.isArray(manifest.entries)) throw new Error('Invalid archive manifest');
  const bySource = new Map(), byDestination = new Map();
  for (const entry of manifest.entries) {
    if (!safeRelative(entry.source) || !safeRelative(entry.destination) || !entry.destination.startsWith('benchmarks/') ||
      !/^[a-f0-9]{64}$/.test(entry.sha256) || !Number.isSafeInteger(entry.bytes) || entry.bytes < 0 ||
      bySource.has(entry.source) || byDestination.has(entry.destination)) throw new Error('Invalid or duplicate archive identity');
    bySource.set(entry.source, entry); byDestination.set(entry.destination, entry);
  }
  function entryFor(filename) {
    let normalized = filename.replaceAll('\\', '/');
    const currentRoot = root.replaceAll('\\', '/');
    const oldRoot = manifest.historicalRoot.replaceAll('\\', '/');
    if (normalized.startsWith(currentRoot + '/')) normalized = normalized.slice(currentRoot.length + 1);
    else if (normalized.startsWith(oldRoot + '/')) normalized = normalized.slice(oldRoot.length + 1);
    if (!safeRelative(normalized)) throw new Error(`Unmapped archive path: ${filename}`);
    const entry = bySource.get(normalized) ?? byDestination.get(normalized);
    if (!entry) throw new Error(`Unmapped archive path: ${filename}`);
    return entry;
  }
  function resolve(filename) { return path.join(root, entryFor(filename).destination); }
  async function read(filename) {
    const entry = entryFor(filename), target = path.join(root, entry.destination);
    const info = await fs.lstat(target);
    const actual = await fs.realpath(target);
    const relative = path.relative(await fs.realpath(root), actual);
    if (!info.isFile() || info.isSymbolicLink() || relative.startsWith('..' + path.sep) || path.isAbsolute(relative)) throw new Error(`Archive path escapes root: ${entry.destination}`);
    const bytes = await fs.readFile(target);
    if (bytes.length !== entry.bytes || sha256(bytes) !== entry.sha256) throw new Error(`Archive checksum changed: ${entry.destination}`);
    return bytes;
  }
  return {
    manifest, resolve, read,
    json: async filename => JSON.parse(await read(filename)),
    hash: async filename => sha256(await read(filename)),
    rows: async filename => (await read(filename)).toString('utf8').trim().split('\n').filter(Boolean).map(JSON.parse),
    verify: async () => {
      for (const entry of manifest.entries) await read(entry.source);
      return { status: 'passed', files: manifest.entries.length, bytes: manifest.entries.reduce((sum, entry) => sum + entry.bytes, 0) };
    },
  };
}
