import fs from 'node:fs/promises';
import path from 'node:path';
import { sha256 } from './manifest.mjs';

// One-time, explicit import. The archive manifest is immutable and never overwritten.
const [planFile, manifestFile] = process.argv.slice(2);
if (!planFile || !manifestFile) throw new Error('Usage: import.mjs <plan.json> <manifest.json>');
const root = process.cwd();
const plan = JSON.parse(await fs.readFile(planFile, 'utf8'));
const allowed = new Set(['.json', '.jsonl', '.md', '.py', '.mjs', '.go', '.sh', '.sha256', '.txt', '.log', '.out', '.error', '.wav', '.aiff', '.pcm', '.png', '.html', '.lock']);
const sources = new Map();
for (const mapping of plan.mappings) {
  async function walk(source) {
    const info = await fs.lstat(source);
    if (info.isSymbolicLink()) return;
    if (info.isDirectory()) {
      for (const name of (await fs.readdir(source)).sort()) {
        if (name === '__pycache__' || name === 'node_modules') continue;
        await walk(path.posix.join(source, name));
      }
    } else if (info.isFile()) {
      const previous = sources.get(source);
      if (!previous || mapping.source.length > previous.source.length) sources.set(source, mapping);
    }
  }
  await walk(mapping.source);
}
const entries = [], excluded = [];
for (const [source, mapping] of [...sources].sort(([a], [b]) => a.localeCompare(b))) {
  const bytes = await fs.readFile(source), ext = path.extname(source);
  const identity = { source, bytes: bytes.length, sha256: sha256(bytes) };
  if (!allowed.has(ext)) {
    excluded.push({ ...identity, reason: 'Executable, bytecode or unsupported artifact; retain its identity without bundling bytes' });
    continue;
  }
  const relative = source === mapping.source ? '' : path.posix.relative(mapping.source, source);
  let destination = relative ? path.posix.join(mapping.destination, relative) : mapping.destination;
  if (mapping.reproduction && ['.go', '.py', '.mjs', '.sh'].includes(ext)) destination = path.posix.join(mapping.reproduction, relative);
  if (!destination.startsWith('benchmarks/') || destination.split('/').includes('..')) throw new Error(`Unsafe destination: ${destination}`);
  if (entries.some(entry => entry.destination === destination)) throw new Error(`Archive collision: ${destination}`);
  const category = ['.wav', '.aiff', '.pcm'].includes(ext) ? 'audio' : ['.png', '.html'].includes(ext) ? 'context-fixture' : 'text';
  entries.push({ ...identity, destination, category, study: mapping.study });
}
await fs.access(manifestFile).then(() => { throw new Error('Archive manifest already exists'); }, error => { if (error.code !== 'ENOENT') throw error; });
// Validate all planned destinations before copying anything.
for (const entry of entries) {
  if (entry.source !== entry.destination) await fs.access(entry.destination).then(() => { throw new Error(`Destination exists: ${entry.destination}`); }, error => { if (error.code !== 'ENOENT') throw error; });
}
for (const entry of entries) {
  if (entry.source === entry.destination) continue;
  const bytes = await fs.readFile(entry.source);
  if (sha256(bytes) !== entry.sha256) throw new Error(`Source changed during import: ${entry.source}`);
  await fs.mkdir(path.dirname(entry.destination), { recursive: true });
  await fs.writeFile(entry.destination, bytes, { flag: 'wx' });
}
const totals = entries.reduce((totals, entry) => {
  const value = totals[entry.category] ??= { files: 0, bytes: 0 };
  value.files++; value.bytes += entry.bytes;
  return totals;
}, {});
const manifest = { version: 1, kind: 'arion-benchmark-archive', historicalRoot: root, planSha256: sha256(await fs.readFile(planFile)), preservation: 'Original bytes; no source files deleted or historical metadata rewritten', entries, excluded, totals };
await fs.writeFile(manifestFile, JSON.stringify(manifest, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify({ files: entries.length, excluded: excluded.length, totals }, null, 2));
