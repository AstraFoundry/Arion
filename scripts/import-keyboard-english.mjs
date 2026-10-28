// Reproduce the pinned ESDB export. Python/SQLite are source-maintenance dependencies only.
import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import { pinnedFile } from './lib/pinned-file.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const manifest = JSON.parse(readFileSync(path.join(root, 'scripts/keyboard-runtime.json'), 'utf8'));
const specification = manifest.english.export;
const temporary = mkdtempSync(path.join(os.tmpdir(), 'arion-esdb-'));
try {
  execFileSync('tar', ['-xf', pinnedFile(specification.archive), '-C', temporary]);
  const cwd = path.join(temporary, specification.root);
  execFileSync('python3', ['combine.py', 'create-db', 'scowl.db'], { cwd, stdio: 'pipe', maxBuffer: 8 * 1024 * 1024 });
  const data = execFileSync('python3', ['scowl', ...specification.arguments], { cwd, maxBuffer: 16 * 1024 * 1024 });
  const expected = manifest.files.find(file => file.path === specification.output);
  if (!expected || data.length !== expected.sizeBytes || createHash('sha256').update(data).digest('hex') !== expected.sha256) {
    throw new Error('The ESDB source export differs from the reviewed word list.');
  }
  pinnedFile(expected);
  console.log(`Verified reproducible ESDB 70 export (${expected.sizeBytes} bytes).`);
} finally {
  rmSync(temporary, { recursive: true, force: true });
}
