import { createHash } from 'node:crypto';
import { lstatSync, readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

if (process.argv.length !== 2) throw new Error('Usage: node scripts/verify-public-fixtures.mjs');
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const manifest = JSON.parse(readFileSync(path.join(root, 'scripts/testdata/public-fixtures.json'), 'utf8'));
if (manifest.schema !== 1 || !Array.isArray(manifest.fixtures) || manifest.fixtures.length === 0 || !Array.isArray(manifest.notices)) {
  throw new Error('Public fixture manifest is incomplete.');
}
const resolve = name => {
  if (typeof name !== 'string' || name.length === 0 || path.isAbsolute(name)) throw new Error('Public fixture path must be relative.');
  const file = path.resolve(root, name);
  const relative = path.relative(root, file);
  if (relative === '..' || relative.startsWith(`..${path.sep}`) || path.isAbsolute(relative)) throw new Error('Public fixture path leaves the repository.');
  return file;
};
const seen = new Set();
for (const fixture of [...manifest.fixtures, ...manifest.notices]) {
  if (seen.has(fixture.path)) throw new Error(`Duplicate public fixture: ${fixture.path}`);
  seen.add(fixture.path);
  const file = resolve(fixture.path);
  if (!lstatSync(file).isFile()) throw new Error(`Public fixture must be a regular file: ${fixture.path}`);
  const bytes = readFileSync(file);
  if (bytes.length !== fixture.sizeBytes || createHash('sha256').update(bytes).digest('hex') !== fixture.sha256) {
    throw new Error(`Public fixture differs from its recorded bytes: ${fixture.path}`);
  }
  if (fixture.licensePath && !lstatSync(resolve(fixture.licensePath)).isFile()) throw new Error(`Public fixture license is missing: ${fixture.path}`);
}
console.log(`Verified ${manifest.fixtures.length} public regression fixtures and ${manifest.notices.length} pinned notices.`);
