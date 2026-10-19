// Load only completed, hash-verified outputs with exact stage lineage.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { hash, indexUnique } from '../correction-chain-v1/report.mjs';
export const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
export const read = file => JSON.parse(fs.readFileSync(file, 'utf8'));
const lines = file => fs.readFileSync(file, 'utf8').trim().split('\n').filter(Boolean).map(JSON.parse);
const check = (ok, message) => { if (!ok) throw new Error(message); };
export function observations(directory) {
  const run = read(path.join(directory, 'run.json'));
  check(run.status === 'completed' && run.study === 'correction-chain-expanded-v2', 'Incomplete or wrong study');
  for (const [file, digest] of Object.entries(run.outputHashes)) check(hash(path.join(directory, file)) === digest, `Changed output: ${file}`);
  const corpus = path.join(root, 'benchmarks/corpora/correction-chain-expanded-v2');
  for (const [file, digest] of [['inputs.json', run.inputsSha256], ['labels.json', run.labelsSha256], ['manifest.json', run.fixtureManifestSha256]]) check(hash(path.join(corpus, file)) === digest, `Changed corpus: ${file}`);
  const samples = read(path.join(corpus, 'inputs.json')).samples.filter(s => s.split === 'test');
  check(samples.length === 100 && new Set(run.caseIds).size === samples.length && samples.every(s => run.caseIds.includes(s.id)), 'Case coverage mismatch');
  const acoustic = indexUnique(lines(path.join(directory, 'acoustic.jsonl')), 'caseId');
  const corrections = indexUnique(lines(path.join(directory, 'correction.jsonl')), 'caseId');
  const rewrites = indexUnique(lines(path.join(directory, 'rewrite.jsonl')), 'caseId');
  const inputs = indexUnique(read(path.join(directory, 'rewrite-inputs.json')).samples, 'id');
  check(acoustic.size === 100 && corrections.size === 100 && rewrites.size === inputs.size, 'Stage coverage mismatch');
  for (const [id, input] of inputs) check(rewrites.get(id)?.source === input.sourceText, 'Changed rewrite source');
  const rows = [];
  for (const sample of samples) {
    const a = acoustic.get(sample.id), c = corrections.get(sample.id);
    check(a && c && a.audioSha256 === sample.audio.sha256, 'Audio identity mismatch');
    check(c.inputText === a.streaming.text && c.model === run.config.correctionModels[sample.language], 'Correction lineage mismatch');
    for (const arm of run.config.arms) {
      const base = arm.startsWith('offline') ? a.offline : a.streaming;
      const corrected = arm.includes('correct');
      const intermediate = corrected ? c.text : base.text;
      const rewrite = arm.endsWith('_llm') ? rewrites.get(`${sample.id}--${arm}`) : null;
      if (arm.endsWith('_llm')) check(rewrite || corrected && c.truncated, 'Missing rewrite');
      if (rewrite) check(rewrite.source === intermediate, 'Broken rewrite lineage');
      rows.push({caseId:sample.id, arm, language:sample.language, task:sample.task, speakerId:sample.speakerId,
        text:arm.endsWith('_llm') ? rewrite?.text ?? '' : intermediate,
        unavailable:arm.endsWith('_llm') && !rewrite, truncated:Boolean(corrected && c.truncated || rewrite?.truncated),
        serialEngineEstimateMs:base.computeMs + (corrected ? c.parentMs : 0) + (rewrite?.generationMs ?? 0),
        correctionMs:corrected ? c.parentMs : 0, rewriteMs:rewrite?.generationMs ?? 0});
    }
  }
  return {run, rows, labels:read(path.join(corpus, 'labels.json')), runSha256:hash(path.join(directory, 'run.json'))};
}
