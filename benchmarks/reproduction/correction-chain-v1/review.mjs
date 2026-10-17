// Export model-blinded tasks without fabricating semantic ratings.
import fs from 'node:fs';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
import { hash } from './report.mjs';
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
const [summaryPath, destination] = process.argv.slice(2);
if (!summaryPath || !destination) throw new Error('Usage: review.mjs SUMMARY NEW_DIRECTORY');
const summary = JSON.parse(fs.readFileSync(summaryPath, 'utf8'));
if (summary.status !== 'completed') throw new Error('Completed report required');
const labels = JSON.parse(fs.readFileSync(path.join(root, 'benchmarks/corpora/correction-chain-v1/labels.json'), 'utf8'));
fs.mkdirSync(destination, { recursive: false });
const mapping = {}, tasks = [];
for (const row of summary.rows) {
  const id = createHash('sha256').update(`${hash(summaryPath)}:${row.caseId}:${row.arm}`).digest('hex').slice(0, 20);
  mapping[id] = { caseId: row.caseId, arm: row.arm };
  tasks.push({ id, spokenScript: labels[row.caseId].transcript,
    intendedText: labels[row.caseId].references, candidate: row.text,
    task: row.task, language: row.language });
}
tasks.sort((a, b) => a.id.localeCompare(b.id));
fs.writeFileSync(path.join(destination, 'tasks.json'), JSON.stringify({ status: 'unrated', rubric: {
  meaning: '1-5: preserves the final expressed intent, including numbers, negation and uncertainty',
  fluency: '1-5: readable and natural in the original language',
  formatting: '1-5: follows the supplied email/list/chat task without invented content',
  flags: ['unsupported_fact', 'unnecessary_edit', 'lost_fact', 'wrong_language']
}, tasks }, null, 2) + '\n');
fs.writeFileSync(path.join(destination, 'mapping-private.json'), JSON.stringify(mapping, null, 2) + '\n');
console.log(JSON.stringify({ tasks: tasks.length, status: 'unrated; keep mapping-private.json away from raters' }));
