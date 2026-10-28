import { createHash } from 'node:crypto';
import { createReadStream } from 'node:fs';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { requireValue } from './protocol.mjs';
import { normalize, tokens } from './metrics.mjs';

export const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
export const digest = value => sha256(JSON.stringify(value));
export const readJson = async filename => JSON.parse(await readFile(filename, 'utf8'));
export async function hashFile(filename) {
  const hash = createHash('sha256');
  for await (const chunk of createReadStream(filename)) hash.update(chunk);
  return hash.digest('hex');
}

export function validateManifest(manifest) {
  requireValue(manifest.version === 1 && typeof manifest.license === 'string' && manifest.license.length > 0 && manifest.consent === true, 'Manifest requires version, license and explicit fixture consent');
  requireValue(Array.isArray(manifest.samples) && manifest.samples.length > 0, 'Corpus must contain samples');
  const ids = new Set();
  const speakers = new Map();
  const sessions = new Map();
  const hashes = new Map();
  for (const sample of manifest.samples) {
    requireValue(typeof sample.id === 'string' && /^[\w-]+$/.test(sample.id) && !ids.has(sample.id), 'Unique safe sample IDs required');
    ids.add(sample.id);
    requireValue(['train', 'dev', 'test'].includes(sample.split) && ['cer', 'wer', 'mer'].includes(sample.unit), 'Invalid split or unit');
    requireValue(['zh', 'en', 'mixed'].includes(sample.language) && Array.isArray(sample.tags) && sample.tags.every(tag => typeof tag === 'string'), 'Language and condition tags required');
    requireValue(sample.unit === ({ zh: 'cer', en: 'wer', mixed: 'mer' })[sample.language], 'Language error-unit mismatch');
    requireValue(typeof sample.reference === 'string' && (sample.reference === '' || normalize(sample.reference).length > 0), 'Invalid reference');
    requireValue(typeof sample.speakerId === 'string' && sample.speakerId.length > 0 && typeof sample.sessionId === 'string' && sample.sessionId.length > 0, 'Speaker and session identity required');
    requireValue(typeof sample.audio?.path === 'string' && /^[a-f0-9]{64}$/.test(sample.audio.sha256) && Number.isFinite(sample.audio.durationMs) && sample.audio.durationMs > 0, 'Pinned audio path/hash/duration required');
    for (const [map, id, label] of [[speakers, sample.speakerId, 'speaker'], [sessions, sample.sessionId, 'session'], [hashes, sample.audio.sha256, 'audio']]) {
      requireValue(!map.has(id) || map.get(id) === sample.split, `Cross-split ${label} leakage`);
      map.set(id, sample.split);
    }
    if (sample.speechIntervals !== undefined) {
      requireValue(Array.isArray(sample.speechIntervals) && sample.speechIntervals.length > 0 && sample.speechIntervals.every((interval, index) => Array.isArray(interval) && interval.length === 2 && interval.every(Number.isFinite) && interval[0] >= 0 && interval[1] > interval[0] && interval[1] <= sample.audio.durationMs && (index === 0 || interval[0] >= sample.speechIntervals[index - 1][1])), 'Invalid speech interval annotations');
      requireValue(['human_verified', 'forced_alignment_reviewed'].includes(sample.timingSource?.kind) && /^[a-f0-9]{64}$/.test(sample.timingSource?.sha256), 'Verified timestamp provenance required');
    }
    if (sample.referenceTurns !== undefined) requireValue(sample.speechIntervals !== undefined && Array.isArray(sample.referenceTurns) && sample.referenceTurns.length > 0 && sample.referenceTurns.every((turn, index) => Number.isFinite(turn.startMs) && Number.isFinite(turn.endMs) && turn.startMs >= 0 && turn.endMs > turn.startMs && turn.endMs <= sample.audio.durationMs && (index === 0 || turn.startMs >= sample.referenceTurns[index - 1].endMs)), 'Invalid intended-turn annotations');
    if (sample.referenceTimeline !== undefined) {
      const referenceTokens = tokens(sample.reference, sample.unit);
      requireValue(sample.speechIntervals !== undefined && Array.isArray(sample.referenceTimeline) && sample.referenceTimeline.length === referenceTokens.length && sample.referenceTimeline.every((entry, index) => entry.text === referenceTokens[index] && Number.isFinite(entry.endMs) && entry.endMs >= 0 && entry.endMs <= sample.audio.durationMs && (index === 0 || entry.endMs >= sample.referenceTimeline[index - 1].endMs)), 'Reference-unit timeline mismatch');
    }
    requireValue(typeof sample.context === 'string' && Array.isArray(sample.approvedMemory), 'Explicit context and approved memory required');
    requireValue(sample.acceptableCandidates === undefined || (Array.isArray(sample.acceptableCandidates) && sample.acceptableCandidates.length > 0 && sample.acceptableCandidates.every(text => typeof text === 'string' && normalize(text).length > 0)), 'Invalid acceptable candidates');
    requireValue(sample.protectedSpans === undefined || (Array.isArray(sample.protectedSpans) && sample.protectedSpans.every(span => ['name', 'number', 'url', 'negation'].includes(span.kind) && typeof span.text === 'string' && span.text.length > 0 && sample.reference.includes(span.text))), 'Invalid protected-span annotation');
  }
  for (const sample of manifest.samples) {
    const memoryIds = new Set();
    for (const record of sample.approvedMemory) {
      requireValue(typeof record.id === 'string' && !memoryIds.has(record.id) && typeof record.text === 'string' && record.text.length > 0 && record.approved === true, 'Invalid approved memory');
      memoryIds.add(record.id);
      requireValue(['train', 'dev', 'user_approved'].includes(record.origin) && Array.isArray(record.sourceSampleIds), 'Memory provenance required; test-derived memory is forbidden');
      for (const id of record.sourceSampleIds) requireValue(manifest.samples.some(s => s.id === id && s.split !== 'test'), 'Memory source is missing or leaks test references');
    }
  }
  return manifest;
}

export async function loadCorpus(filename) {
  const manifest = validateManifest(await readJson(filename));
  for (const sample of manifest.samples) {
    const filenameAudio = path.resolve(path.dirname(filename), sample.audio.path);
    requireValue(await hashFile(filenameAudio) === sample.audio.sha256, `Audio checksum mismatch for ${sample.id}`);
  }
  return manifest;
}

export function validateExperiment(config) {
  requireValue(config.version === 1 && typeof config.id === 'string' && config.id.length > 0, 'Invalid experiment version/identity');
  requireValue(['dev', 'test'].includes(config.split) && Array.isArray(config.seeds) && config.seeds.length > 0 && new Set(config.seeds).size === config.seeds.length && config.seeds.every(Number.isInteger), 'Unique integer seeds and held-out split required');
  requireValue(Number.isInteger(config.orderSeed) && Number.isInteger(config.warmupCount) && config.warmupCount >= 1, 'Ordering seed and explicit warmup required');
  requireValue(Array.isArray(config.cacheStates) && config.cacheStates.length > 0 && new Set(config.cacheStates).size === config.cacheStates.length && config.cacheStates.every(s => ['cold', 'warm'].includes(s)), 'Invalid cache-state strata');
  requireValue(config.pipeline && typeof config.pipeline.asr === 'string' && config.pipeline.pace === 'realtime' && Number.isFinite(config.pipeline.chunkMs) && config.pipeline.chunkMs > 0 && config.pipeline.sampleRate === 16000, 'Pinned realtime pipeline settings required');
  requireValue(Number.isInteger(config.bootstrap?.iterations) && config.bootstrap.iterations >= 100 && Number.isInteger(config.bootstrap.seed), 'Bootstrap settings required');
  requireValue(config.design === undefined || ['ablation','validation'].includes(config.design), 'Unknown experiment design');
  requireValue(Array.isArray(config.variants) && config.variants.length >= (config.design === 'validation' ? 1 : 2), 'Ablation requires variants; single-backend validation must declare its design');
  const ids = new Set();
  for (const variant of config.variants) {
    requireValue(typeof variant.id === 'string' && /^[\w-]+$/.test(variant.id) && !ids.has(variant.id), 'Duplicate or unsafe variant ID');
    ids.add(variant.id);
    // `asr` selects a pinned recognition bundle for model-comparison runs; the
    // other overrides are numeric streaming controls.
    if (variant.pipelineOverrides !== undefined) requireValue(Object.entries(variant.pipelineOverrides).every(([key, value]) => key === 'asr' ? typeof value === 'string' && /^[\w.-]+$/.test(value) : ['chunkMs', 'endpointSilenceMs', 'stabilityHoldMs'].includes(key) && Number.isFinite(value) && value >= 0 && (key !== 'chunkMs' || value > 0)), 'Invalid controlled streaming override');
    requireValue(typeof variant.features?.memory === 'boolean' && ['source', 'laya'].includes(variant.features.ranker) && ['none', 'local'].includes(variant.features.correction), 'Unknown feature treatment');
  }
  requireValue(ids.has(config.baseline), 'Baseline not present');
  requireValue(new Set(config.variants.map(v => digest({ features: v.features, overrides: v.pipelineOverrides ?? {} }))).size === config.variants.length, 'Duplicate feature conditions');
  return config;
}
