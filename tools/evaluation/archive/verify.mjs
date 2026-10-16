import path from 'node:path';
import assert from 'node:assert/strict';
import { requireCoverage } from '../phonetic/design.mjs';
import { scoreConfidence } from '../phonetic/confidence.mjs';

export async function verifyEvidence(archive) {
  const bytes = await archive.verify();
  const verifiedSources = new Set();
  const provenance = (await archive.json('benchmarks/configs/disfluency-alignment-report-v2.json')).provenance;
  for (const section of ['sources', 'scorers', 'productionAudioSources', 'uiSourceSha256']) {
    for (const [source, expected] of Object.entries(provenance[section] ?? {})) {
      assert.equal(await archive.hash(source), expected, `Changed selected source: ${source}`);
      verifiedSources.add(source);
    }
  }
  const runStatuses = [], linkedFiles = [], audioReferences = [];
  for (const entry of archive.manifest.entries) {
    if (!entry.destination.endsWith('.json')) continue;
    if (!entry.source.endsWith('/run.json') && !entry.destination.startsWith('benchmarks/corpora/')) continue;
    const value = await archive.json(entry.source);
    if (entry.source.startsWith('.workspace/') && entry.source.endsWith('/run.json')) {
      runStatuses.push({ source: entry.source, status: value.status ?? 'unspecified' });
      for (const name of ['results', 'warmups']) {
        const expected = value[`${name}Hash`] ?? value[`${name}Sha256`] ?? value[`${name}HashAtStop`] ?? value[`${name}HashAtAbort`];
        if (!expected) continue;
        const source = path.posix.join(path.posix.dirname(entry.source), `${name}.jsonl`);
        assert.equal(await archive.hash(source), expected, `Changed recorded ${name}: ${entry.source}`);
        linkedFiles.push(source);
      }
    }
    for (const sample of value.samples ?? []) {
      if (!sample.audio?.path || !sample.audio.sha256) continue;
      const source = path.posix.isAbsolute(sample.audio.path) ? sample.audio.path : path.posix.join(path.posix.dirname(entry.source), sample.audio.path);
      assert.equal(await archive.hash(source), sample.audio.sha256, `Changed audio: ${sample.id}`);
      audioReferences.push({ corpus: entry.source, id: sample.id });
    }
  }
  const base = '.workspace/benchmarks/phonetic-asr-v2';
  const parent = await archive.json(`${base}/correction/run.json`);
  const view = await archive.json(`${base}/completed-17b-view/run.json`);
  for (const artifact of view.codeArtifacts) {
    assert.equal(await archive.hash(artifact.path), artifact.sha256, `Changed phonetic source: ${artifact.path}`);
    verifiedSources.add(artifact.path);
  }
  assert.equal(parent.status, 'cancelled_by_user');
  assert.equal(view.status, 'complete');
  assert.equal(view.derivedAnalysisScope.sourceRunHash, await archive.hash(`${base}/correction/run.json`));
  assert.equal(view.derivedAnalysisScope.sourceResultsHash, await archive.hash(`${base}/correction/results.jsonl`));
  assert.equal(view.derivedAnalysisScope.sourceWarmupsHash, await archive.hash(`${base}/correction/warmups.jsonl`));
  const source = (await archive.rows(`${base}/correction/results.jsonl`)).filter(row => row.model !== 'qwen3-4b');
  const selected = await archive.rows(`${base}/completed-17b-view/results.jsonl`);
  assert.deepEqual(selected, source, 'Changed checkpoint selection');
  const corpus = await archive.json(view.corpusFile);
  requireCoverage(selected, corpus.samples.filter(sample => sample.split === 'test'), ['qwen3-1.7b'], view.config.variants);
  const evidence = new Map((await archive.rows(path.posix.join(view.evidenceDirectory, 'results.jsonl'))).map(row => [row.caseId, row.response]));
  const confidence = await archive.json('benchmarks/results/phonetic-asr-1.7b-v2/confidence.json');
  assert.equal(confidence.evidenceHash, await archive.hash(path.posix.join(view.evidenceDirectory, 'results.jsonl')));
  for (const split of ['dev', 'test']) for (const language of ['all', 'zh', 'en', 'mixed']) {
    const samples = corpus.samples.filter(sample => sample.split === split && (language === 'all' || sample.language === language));
    assert.deepEqual(scoreConfidence(samples, evidence, view.threshold), confidence.groups[`${split}/${language}`]);
  }
  return { ...bytes, linkedResultFiles: linkedFiles.length, audioReferences: audioReferences.length,
    verifiedSourceFiles: verifiedSources.size, runStatuses,
    phonetic: { parent: parent.status, selectedModel: 'qwen3-1.7b', rawControls: 154, measuredGenerations: 924, confidenceGroups: 8 },
    limitation: 'This verifies retained bytes, linked observations and scoring evidence; it does not rerun inference or provision model/runtime/executable artifacts.' };
}
