// Model-free scoring only. Targets never enter the native driver protocol.
import { candidateCoverage } from './candidate-ranges.mjs';
import { readFileSync, existsSync } from 'node:fs';
import { createHash } from 'node:crypto';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
export const hash = file => createHash('sha256').update(readFileSync(file)).digest('hex');
export const read = file => JSON.parse(readFileSync(file, 'utf8'));
const normalize = text => text.normalize('NFKC').toLowerCase().replace(/[\p{P}\p{S}]/gu, ' ').replace(/\s+/g, ' ').trim();
export const percentile = (values, probability) => {
  if (!values.length) return null;
  const sorted = [...values].sort((a, b) => a - b), index = (sorted.length - 1) * probability;
  return sorted[Math.floor(index)] + (sorted[Math.ceil(index)] - sorted[Math.floor(index)]) * (index % 1);
};
function distance(left, right) {
  let previous = Array.from({ length: right.length + 1 }, (_, i) => i);
  for (let i = 1; i <= left.length; i++) {
    const current = [i];
    for (let j = 1; j <= right.length; j++) current[j] = Math.min(previous[j] + 1, current[j - 1] + 1, previous[j - 1] + (left[i - 1] !== right[j - 1]));
    previous = current;
  }
  return previous[right.length];
}
function distribution(values) {
  if (values.some(value => !Number.isFinite(value) || value < 0)) throw new Error('Invalid native monotonic timing.');
  return { count: values.length, p50: percentile(values, .5), p95: percentile(values, .95), p99: percentile(values, .99), maximum: values.length ? Math.max(...values) : null };
}
function aggregate(rows) {
  const count = rows.length, total = key => rows.reduce((sum, row) => sum + row[key], 0);
  return { observations: count, distinctInputs: new Set(rows.map(row => row.id)).size,
    commitExactCount: total('commitExact'), commitExact: total('commitExact') / count,
    rawCommitExactCount: total('rawExact'), rawCommitExact: total('rawExact') / count,
    firstCandidateExactCount: total('top1'), firstCandidateExact: total('top1') / count,
    firstPageOracleHits: total('oracle'), firstPageOracleRecallAt9: total('oracle') / count,
    firstPageMRRAt9: total('reciprocalRank') / count,
    ...(rows[0]?.fullInputRecall === undefined ? {} : { fullInputOracleHits: total('fullInputRecall'),
      fullInputOracleRecallAt9: total('fullInputRecall') / count, partialOnlyOracleHits: total('partialOnlyRecall'),
      fullInputTop1Count: total('fullInputTop1') }),
    characterErrors: total('errors'), referenceCharacters: total('units'), corpusCER: total('errors') / total('units'),
    unfinishedConfirmationCount: rows.filter(row => row.terminal !== 'empty-preedit').length };
}
function bootstrap(rows) {
  const groups = [...new Set(rows.map(row => row.id))].map(id => rows.filter(row => row.id === id));
  let random = 83017;
  const next = () => { random ^= random << 13; random ^= random >>> 17; random ^= random << 5; return (random >>> 0) / 4294967296; };
  const accuracy = [], cer = [];
  for (let draw = 0; draw < 2000; draw++) {
    const sample = [];
    for (let i = 0; i < groups.length; i++) sample.push(...groups[Math.floor(next() * groups.length)]);
    const value = aggregate(sample); accuracy.push(value.commitExact); cer.push(value.corpusCER);
  }
  return { method: '2000 percentile draws, seed83017, resample authored input IDs with all three repeats; exploratory synthetic-case uncertainty only',
    commitExact95: [percentile(accuracy, .025), percentile(accuracy, .975)], corpusCER95: [percentile(cer, .025), percentile(cer, .975)] };
}
export function evaluate(directory, corpusFile) {
  const status = read(path.join(directory, 'status.json'));
  if (status.status !== 'completed') throw new Error('Incomplete or failed runs cannot produce a completed report.');
  const corpus = read(corpusFile), index = read(path.join(directory, 'observations.json'));
  const metadata = read(path.join(directory, 'metadata.json'));
  if (hash(corpusFile) !== metadata.corpusSha256) throw new Error('Corpus differs from executed frozen bytes.');
  if (corpus.cases.length !== 100 || index.length !== 600) throw new Error('Expected100 distinct fixtures and600 paired observations.');
  const lookup = new Map(corpus.cases.map(row => [row.id, row]));
  const identities = new Set(), rows = [];
  for (const record of index) {
    const identity = `${record.seed}/${record.stratum}/${record.id}`;
    if (identities.has(identity)) throw new Error(`Duplicate observation: ${identity}`);
    identities.add(identity);
    if (![1701, 1702, 1703].includes(record.seed) || !['cold-process', 'warm-process'].includes(record.stratum)) throw new Error('Changed treatment or repeat.');
    const fixture = lookup.get(record.id), file = path.join(directory, record.result);
    if (!fixture || !existsSync(file) || hash(file) !== record.sha256 || record.exitStatus !== 0) throw new Error(`Missing/changed observation: ${identity}`);
    const actual = read(file);
    if (actual.status !== 'completed' || actual.id !== fixture.id || actual.input !== fixture.input || !Array.isArray(actual.steps) || actual.steps.length !== fixture.input.length) throw new Error(`Invalid native result: ${identity}`);
    const request = read(path.join(directory, record.request));
    if (Object.keys(request).sort().join(',') !== 'id,input,warmups' || request.id !== fixture.id || request.input !== fixture.input || JSON.stringify(request.warmups) !== JSON.stringify(record.stratum === 'cold-process' ? [] : ['xuetang', 'heishan'])) throw new Error('Driver reference leakage or warmup change.');
    if (actual.warmups.length !== request.warmups.length) throw new Error('Unpaired warmup count.');
    const reference = normalize(fixture.reference), committed = normalize(actual.committed);
    const candidates = actual.beforeConfirmation.candidates.map(normalize), rank = candidates.indexOf(reference);
    rows.push({ id: fixture.id, category: fixture.category, seed: record.seed, stratum: record.stratum,
      input: fixture.input, reference: fixture.reference, actual: actual.committed,
      top1Candidate: actual.beforeConfirmation.candidates[0] ?? null, candidates: actual.beforeConfirmation.candidates,
      commitExact: Number(committed === reference), rawExact: Number(actual.committed === fixture.reference),
      top1: Number(candidates[0] === reference), oracle: Number(rank >= 0), reciprocalRank: rank < 0 ? 0 : 1 / (rank + 1),
      errors: distance([...reference.replace(/\s/g, '')], [...committed.replace(/\s/g, '')]), units: [...reference.replace(/\s/g, '')].length,
      terminal: actual.terminal, timings: { keys: actual.steps.map(step => step.keyToSnapshotMs), firstKey: actual.steps[0].keyToSnapshotMs,
        confirmations: actual.confirmations.map(step => step.keyToSnapshotMs), startup: actual.startupToSessionMs,
        warmup: actual.warmupMs, typing: actual.inputKeysToFinalSnapshotMs, teardown: actual.teardownMs, childWall: record.childWallMs },
      ...candidateCoverage(actual, metadata, fixture.reference, normalize), peakRSSBytes: actual.peakRSSBytes });
  }
  const strata = {};
  for (const stratum of ['cold-process', 'warm-process']) {
    const selected = rows.filter(row => row.stratum === stratum);
    if (selected.length !== 300 || [...lookup.keys()].some(id => selected.filter(row => row.id === id).length !== 3)) throw new Error('Incomplete paired fixture/repeat coverage.');
    strata[stratum] = { quality: aggregate(selected), uncertainty: bootstrap(selected),
      categories: Object.fromEntries([...new Set(rows.map(row => row.category))].map(category => [category, aggregate(selected.filter(row => row.category === category))])),
      latencyMs: { keyToSnapshot: distribution(selected.flatMap(row => row.timings.keys)), firstKeyToSnapshot: distribution(selected.map(row => row.timings.firstKey)),
        confirmationKeyToSnapshot: distribution(selected.flatMap(row => row.timings.confirmations)),
        initializationToSession: distribution(selected.map(row => row.timings.startup)),
        initializationPlusFirstKey: distribution(selected.map(row => row.timings.startup + row.timings.firstKey)),
        inputKeysToFinalSnapshot: distribution(selected.map(row => row.timings.typing)), warmups: distribution(selected.map(row => row.timings.warmup)),
        processWall: distribution(selected.map(row => row.timings.childWall)), teardown: distribution(selected.map(row => row.timings.teardown)) },
      peakRSSBytes: distribution(selected.map(row => row.peakRSSBytes)),
      outputsVaryAcrossRepeats: [...lookup.keys()].filter(id => new Set(selected.filter(row => row.id === id).map(row => row.actual)).size > 1) };
  }
  return { schemaVersion: 1, status: 'completed', corpus: corpus.id, normalization: 'arion-nfkc-lower-punctuation-space-v1; CER removes remaining whitespace',
    uniqueFixtures: 100, pairedRepeats: 3, completedNativeProcesses: 600, strata, failures: rows.filter(row => row.commitExact === 0 && row.seed === 1701),
    limits: ['Original synthetic exposed development set, one author-intended reading, no independent annotation or representative population claim',
      'Warm means a process-resident engine after two cancelled unrelated warmups; every fixture/repeat gets fresh synthetic-key storage and process to prevent learned test memory',
      'Cold means fresh process/engine; OS file caches were not flushed and dynamic-loader cost is excluded from native startup timing',
      'Key-to-snapshot is synchronous engine processing/copying; no OS event dispatch, IMK client, screen rendering or end-to-end latency',
      'Only the actual first candidate page (at most9) is inspected; oracle recall/MRR are page-bounded, not full lexicon oracle',
      'Shared host, no CPU/power isolation, no simultaneous-ASR condition, no acceptance threshold or competitor-parity claim'],
    rows };
}
if (process.argv[1] && pathToFileURL(path.resolve(process.argv[1])).href === import.meta.url) {
  if (process.argv.length !== 4) throw new Error('usage: evaluate.mjs RUN_DIRECTORY CORPUS_JSON');
  process.stdout.write(`${JSON.stringify(evaluate(process.argv[2], process.argv[3]), null, 2)}\n`);
}
