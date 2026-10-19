// Read-only report replay; optional negative checks mutate only disposable copies.
import { createHash } from 'node:crypto';
import { cpSync, existsSync, lstatSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import assert from 'node:assert/strict';
import { evaluate } from './evaluate.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const args = process.argv.slice(2);
const archiveOption = args.indexOf('--archive');
if (archiveOption >= 0 && (!args[archiveOption + 1] || args[archiveOption + 1].startsWith('--'))) throw new Error('Missing archive directory.');
const archiveRoot = archiveOption >= 0 ? path.resolve(args[archiveOption + 1]) : path.join(root, 'benchmarks/runs/imk-quality-v2');
const allowed = args.filter((_, index) => index !== archiveOption && index !== (archiveOption >= 0 ? archiveOption + 1 : -1));
if (allowed.some(value => value !== '--negative-tests')) throw new Error('usage: verify.mjs [--archive DIRECTORY] [--negative-tests]');
const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
const read = file => JSON.parse(readFileSync(file, 'utf8'));
const archive = read(path.join(archiveRoot, 'archive.json'));
const mapped = new Set();
for (const record of archive.files) {
  if (path.isAbsolute(record.path) || record.path.split('/').some(part => !part || part === '..' || part === '.')) throw new Error('Unsafe archive member.');
  if (mapped.has(record.path)) throw new Error(`Duplicate archived member: ${record.path}`);
  mapped.add(record.path);
  const file = path.join(archiveRoot, record.path), stat = lstatSync(file);
  if (!stat.isFile() || stat.size !== record.sizeBytes || hash(file) !== record.sha256) throw new Error(`Changed original archive bytes: ${record.path}`);
}
const sourceHashes = read(path.join(archiveRoot, 'preparation/source-identities.json'));
for (const [name, expected] of Object.entries(sourceHashes)) if (hash(path.join(archiveRoot, 'sources', name)) !== expected) throw new Error(`Changed executed source snapshot: ${name}`);
const run = path.join(archiveRoot, 'run'), corpus = path.join(archiveRoot, 'sources/benchmarks/corpora/imk-quality-v1/corpus.json');
const replay = evaluate(run, corpus);
assert.deepEqual(replay, read(path.join(run, 'report.json')), 'Independent model-free evaluation must reproduce every recorded score and timing summary.');
const expectedSummary = Object.fromEntries(Object.entries(replay).filter(([key]) => !['rows', 'failures'].includes(key)));
expectedSummary.fullReport = { path: `../../runs/${path.basename(archiveRoot)}/run/report.json`, sha256: hash(path.join(run, 'report.json')) };
if (!archive.qualityValidity) assert.deepEqual(read(path.join(root, 'benchmarks/results/imk-quality-v1/summary.json')), expectedSummary);
const audit = read(path.join(archiveRoot, 'post-run-audit.json'));
assert.deepEqual(audit.benchmarkInputIntersection, []);
console.log(`PASS: ${mapped.size} original-file SHA-256 mappings,600 reference-free native observations,100 fixtures, source identities and exact model-free report replay.`);

if (process.argv.includes('--negative-tests')) {
  const temp = mkdtempSync(path.join(os.tmpdir(), 'arion-imk-benchmark-verification-'));
  try {
    const copied = path.join(temp, 'run'); cpSync(run, copied, { recursive: true });
    const indexFile = path.join(copied, 'observations.json'), originalIndex = readFileSync(indexFile);
    const index = read(indexFile), rawFile = path.join(copied, index[0].result), originalRaw = readFileSync(rawFile);
    writeFileSync(rawFile, Buffer.concat([originalRaw, Buffer.from('\n')]));
    assert.throws(() => evaluate(copied, corpus), /Missing\/changed observation/);
    writeFileSync(rawFile, originalRaw);
    const duplicate = [...index]; duplicate[1] = duplicate[0]; writeFileSync(indexFile, JSON.stringify(duplicate));
    assert.throws(() => evaluate(copied, corpus), /Duplicate observation/);
    writeFileSync(indexFile, originalIndex);
    const statusFile = path.join(copied, 'status.json'), originalStatus = readFileSync(statusFile);
    writeFileSync(statusFile, JSON.stringify({ status: 'failed' }));
    assert.throws(() => evaluate(copied, corpus), /Incomplete or failed/);
    writeFileSync(statusFile, originalStatus);
    const requestFile = path.join(copied, index[0].request), request = read(requestFile);
    request.reference = 'Evaluator-only label must never enter driver input.'; writeFileSync(requestFile, JSON.stringify(request));
    assert.throws(() => evaluate(copied, corpus), /reference leakage/);
    console.log('PASS: changed raw output, duplicate pairing, incomplete status and reference-leaking request are rejected; originals unchanged.');
  } finally { rmSync(temp, { recursive: true, force: true }); }
}
