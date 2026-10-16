import fs from 'node:fs/promises';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { hashFile, loadCorpus } from '../../../tools/evaluation/src/manifest.mjs';
import { requireCoverage, promptFor } from '../../../tools/evaluation/phonetic/design.mjs';
import { scoreConfidence } from '../../../tools/evaluation/phonetic/confidence.mjs';
const base = path.resolve('.workspace/benchmarks/phonetic-asr-v2');
const published = path.resolve('benchmarks/results/phonetic-asr-1.7b-v2');
const read = async file => JSON.parse(await fs.readFile(file, 'utf8'));
const lines = async file => (await fs.readFile(file, 'utf8')).trim().split('\n').filter(Boolean).map(JSON.parse);
const check = (condition, message) => { if (!condition) throw new Error(message); };
const parent = await read(path.join(base, 'correction/run.json'));
const view = await read(path.join(base, 'completed-17b-view/run.json'));
const summary = await read(path.join(base, 'summary.json'));
check(parent.status === 'cancelled_by_user', 'Parent must remain cancelled');
check(view.derivedAnalysisScope.sourceRunHash === await hashFile(path.join(base, 'correction/run.json')), 'Changed source provenance');
check(parent.resultsHashAtStop === await hashFile(path.join(base, 'correction/results.jsonl')), 'Changed stopped results');
check(parent.warmupsHashAtStop === await hashFile(path.join(base, 'correction/warmups.jsonl')), 'Changed stopped warmups');
check(summary.runHash === await hashFile(path.join(base, 'completed-17b-view/run.json')), 'Changed selected scope');
const sourceRows = (await lines(path.join(base, 'correction/results.jsonl'))).filter(row => row.model !== 'qwen3-4b');
const rows = await lines(path.join(base, 'completed-17b-view/results.jsonl'));
check(JSON.stringify(sourceRows) === JSON.stringify(rows), 'Selected observations differ from original inference');
const corpus = await loadCorpus(view.corpusFile);
const test = corpus.samples.filter(sample => sample.split === 'test');
requireCoverage(rows, test, ['qwen3-1.7b'], view.config.variants);
const evidence = new Map((await lines(path.join(view.evidenceDirectory, 'results.jsonl'))).map(row => [row.caseId, row.response]));
for (const row of rows) {
 if (row.model === 'raw') check(row.response.text === evidence.get(row.caseId).asrText, 'Changed raw source');
 else check(row.prompt === promptFor(evidence.get(row.caseId), row.variant, view.threshold), 'Changed prompt');
}
const artifacts = [...view.codeArtifacts, ...view.runtimeArtifacts, ...view.modelArtifacts['qwen3-1.7b']];
for (const artifact of artifacts) check(artifact.sha256 === await hashFile(artifact.path), `Changed pinned artifact: ${path.basename(artifact.path)}`);
for (const name of ['metrics', 'statistics', 'manifest']) {
 const file = `tools/evaluation/src/${name}.mjs`;
 const original = execFileSync('git', ['show', `${view.gitRevision}:${file}`]);
 check(createHash('sha256').update(original).digest('hex') === await hashFile(file), 'Shared evaluator differs from recorded revision');
}
const confidence = await read(path.join(published, 'confidence.json'));
check(confidence.evidenceHash === await hashFile(path.join(view.evidenceDirectory, 'results.jsonl')), 'Changed confidence input');
for (const split of ['dev', 'test']) for (const language of ['all', 'zh', 'en', 'mixed']) {
 const samples = corpus.samples.filter(sample => sample.split === split && (language === 'all' || sample.language === language));
 check(JSON.stringify(scoreConfidence(samples, evidence, view.threshold)) === JSON.stringify(confidence.groups[`${split}/${language}`]), 'Confidence recomputation mismatch');
}
for (const name of ['summary', 'supplemental']) check(await hashFile(path.join(base, `${name}.json`)) === await hashFile(path.join(published, `${name}.json`)), 'Published aggregate mismatch');
const result = { status: 'passed', measuredGenerations: rows.filter(row => row.model !== 'raw').length, rawControls: rows.filter(row => row.model === 'raw').length,
 verifiedPinnedArtifacts: artifacts.length, unchangedSharedEvaluators: 3, parentStatus: parent.status, selectedModel: 'qwen3-1.7b', confidenceRecomputedGroups: 8,
 publishedHashes: Object.fromEntries(await Promise.all(['summary', 'supplemental', 'confidence'].map(async name => [name, await hashFile(path.join(published, `${name}.json`))]))) };
await fs.writeFile(path.join(base, 'verification.json'), JSON.stringify(result, null, 2) + '\n', { flag: 'wx' });
console.log(JSON.stringify(result));
