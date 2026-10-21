import { spawnSync } from 'node:child_process';
import { createReadStream } from 'node:fs';
import { readFile, stat } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const study = path.join(root, 'benchmarks/reproduction/qwen17-accessibility-v1');
const operation = process.argv[2];
if (operation === 'verify') {
  const manifest = JSON.parse(await readFile(path.join(root, 'benchmarks/results/qwen17-accessibility-v1/archive.json'), 'utf8'));
  if (manifest.version !== 1 || manifest.status !== 'complete' || !Array.isArray(manifest.files) || manifest.files.length === 0) throw new Error('Incomplete archive manifest');
  const seen = new Set();
  for (const entry of manifest.files) {
    const filename = path.resolve(root, entry.path);
    if (!filename.startsWith(root + path.sep) || seen.has(filename)) throw new Error('Invalid archive path');
    seen.add(filename);
    const hash = createHash('sha256');
    for await (const block of createReadStream(filename)) hash.update(block);
    if ((await stat(filename)).size !== entry.sizeBytes || hash.digest('hex') !== entry.sha256) throw new Error(`Archive mismatch: ${entry.path}`);
  }
  console.log(`Verified ${seen.size} archived files; no model weights required.`);
} else {
  if (!['report', 'test'].includes(operation)) throw new Error('Use report --run PATH --out NEWFILE, test, or verify');
  const result = spawnSync(process.env.EVAL_PYTHON || 'python3', ['-B', path.join(study, operation === 'test' ? 'test_report.py' : 'report.py'), ...process.argv.slice(3)], { cwd: root, stdio: 'inherit', shell: false });
  if (result.error) throw result.error;
  process.exitCode = result.status ?? 1;
}
