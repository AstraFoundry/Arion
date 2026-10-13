import path from 'node:path';
import { digest, hashFile, readJson } from './manifest.mjs';
import { requireValue as need } from './protocol.mjs';
import { tokens } from './metrics.mjs';

const safe = value => typeof value === 'string' && /^[\w.-]+$/.test(value);
const text = value => typeof value === 'string';
const hash = value => /^[a-f0-9]{64}$/.test(value);
const unique = values => new Set(values).size === values.length;
export const REWRITE_SCHEMA = 'arion-rewrite-v1';

export function validateRewriteConfig(config) {
  need(config.version === 1 && config.kind === REWRITE_SCHEMA && safe(config.id), 'Invalid rewrite experiment identity');
  need(['dev', 'test'].includes(config.split), 'Rewrite requires a held-out split');
  need(Array.isArray(config.seeds) && config.seeds.length > 0 && unique(config.seeds) && config.seeds.every(Number.isSafeInteger), 'Unique rewrite seeds required');
  need(Number.isSafeInteger(config.orderSeed) && Number.isSafeInteger(config.warmupCount) && config.warmupCount > 0, 'Explicit rewrite ordering/warmup required');
  need(Array.isArray(config.cacheStates) && config.cacheStates.length > 0 && unique(config.cacheStates) && config.cacheStates.every(s => ['warm', 'cold'].includes(s)), 'Invalid rewrite cache strata');
  need(Array.isArray(config.loadStates) && config.loadStates.length > 0 && unique(config.loadStates) && config.loadStates.every(s => ['solo', 'concurrent_asr'].includes(s)), 'Invalid rewrite load strata');
  const p = config.pipeline;
  need(p && safe(p.model) && safe(p.prompt) && safe(p.tokenizer) && safe(p.provider) && safe(p.asr) && p.preview === 'native' && ['int4', 'int8', 'fp16', 'fp32'].includes(p.precision), 'Pinned model/prompt/tokenizer/provider/ASR, precision and native preview required');
  need(['maxNewTokens', 'maxInputTokens', 'threads'].every(k => Number.isSafeInteger(p[k]) && p[k] > 0) && p.temperature === 0 && p.topP === 1 && p.thinking === false, 'Bounded deterministic non-thinking decoding required');
  need(Number.isSafeInteger(config.bootstrap?.iterations) && config.bootstrap.iterations >= 100 && Number.isSafeInteger(config.bootstrap.seed), 'Explicit rewrite bootstrap required');
  need(Array.isArray(config.variants) && config.variants.length > 1, 'Rewrite controls required');
  for (const v of config.variants) {
    need(safe(v.id) && ['verbatim', 'rules', 'llm'].includes(v.features?.engine) && ['context', 'memory', 'scenario'].every(k => typeof v.features[k] === 'boolean') && Object.keys(v.features).length === 4, 'Invalid rewrite feature treatment');
    need(v.features.engine !== 'verbatim' || ['context', 'memory', 'scenario'].every(k => !v.features[k]), 'Verbatim control must disable optional features');
    if (v.pipelineOverrides !== undefined) need(v.features.engine === 'llm' && Object.keys(v.pipelineOverrides).length === 2 && safe(v.pipelineOverrides.model) && ['int4', 'int8', 'fp16', 'fp32'].includes(v.pipelineOverrides.precision), 'Only explicit LLM model/precision overrides allowed');
  }
  need(unique(config.variants.map(v => v.id)) && unique(config.variants.map(v => digest({ features: v.features, overrides: v.pipelineOverrides ?? {} }))) && config.variants.some(v => v.id === config.baseline && v.features.engine === 'rules'), 'Unique variants and rule baseline required');
  return config;
}

function validateOrigin(record) {
  need(record && text(record.text) && ['train', 'dev', 'user_approved'].includes(record.origin) && Array.isArray(record.sourceSampleIds) && record.sourceSampleIds.every(safe), 'Context/memory provenance required');
}

export function validateRewriteCorpus(corpus) {
  need(corpus.version === 1 && corpus.kind === REWRITE_SCHEMA && text(corpus.license) && corpus.license.length > 0 && corpus.consent === true, 'Rewrite corpus requires license and authorized fixture consent');
  need(Array.isArray(corpus.samples) && corpus.samples.length > 0, 'Rewrite samples required');
  const splitMaps = [new Map(), new Map(), new Map(), new Map(), new Map()];
  const ids = new Set();
  for (const s of corpus.samples) {
    need(safe(s.id) && !ids.has(s.id), 'Unique rewrite case ID required'); ids.add(s.id);
    need(['train', 'dev', 'test'].includes(s.split) && ['zh', 'en', 'mixed'].includes(s.language) && ['gold_text', 'frozen_asr'].includes(s.inputKind), 'Invalid rewrite split/language/input stratum');
    need([s.clusterId, s.sessionId, s.contentId].every(safe) && text(s.sourceText) && s.sourceText.length > 0 && s.sourceText.length <= 65536 && s.sourceSha256 === digest(s.sourceText), 'Reviewed source identity/hash/size required');
    need(Array.isArray(s.tags) && s.tags.every(safe), 'Rewrite condition tags required');
    need(s.annotation?.kind === 'human_reviewed' && text(s.annotation.path) && hash(s.annotation.sha256), 'Independent reviewed annotation artifact required');
    need(s.asrLoad && text(s.asrLoad.path) && hash(s.asrLoad.sha256) && Number.isFinite(s.asrLoad.durationMs) && s.asrLoad.durationMs > 0, 'Pinned contention audio required for every case');
    for (const [i, identity] of [s.clusterId, s.sessionId, s.contentId, s.sourceSha256, s.asrLoad.sha256].entries()) {
      need(!splitMaps[i].has(identity) || splitMaps[i].get(identity) === s.split, 'Cross-split author/session/content/source/audio leakage');
      splitMaps[i].set(identity, s.split);
    }
    need(text(s.userInstruction) && s.userInstruction.length <= 4096 && text(s.scenario) && s.scenario.length > 0 && s.scenario.length <= 128, 'Bounded user instruction and scenario required');
    validateOrigin(s.context);
    need(s.context.text.length <= 8192, 'Rewrite context exceeds fixture character budget');
    need(Array.isArray(s.approvedMemory), 'Approved rewrite memory required');
    for (const m of s.approvedMemory) { validateOrigin(m); need(safe(m.id) && m.approved === true && m.text.length > 0, 'Only approved memory allowed'); }
    need(s.approvedMemory.reduce((n, m) => n + m.text.length, 0) <= 8192, 'Approved memory exceeds fixture character budget');
    need(unique(s.approvedMemory.map(m => m.id)), 'Duplicate rewrite memory');
    need(Array.isArray(s.controls) && s.controls.length <= 1 && s.controls.every(c => Number.isFinite(c.atMs) && c.atMs >= 0 && ['cancel', 'focus_change', 'mode_switch', 'invalidate_revision'].includes(c.type)), 'At most one explicit invalidation control per case');
    const e = s.expected;
    need(e && Array.isArray(e.references) && e.references.length > 0 && e.references.every(t => text(t) && tokens(t, 'mer').length > 0) && Array.isArray(e.protectedSpans) && e.protectedSpans.every(p => ['name', 'number', 'url', 'negation', 'code'].includes(p.kind) && text(p.text) && p.text.length > 0 && Number.isSafeInteger(p.minimumOccurrences) && p.minimumOccurrences > 0), 'Reviewed references/protected-span labels required');
    need(['requiredStrings', 'forbiddenStrings', 'permittedNumbers'].every(k => Array.isArray(e[k]) && e[k].every(t => text(t) && t.length > 0)), 'Explicit constraint/attack/number label arrays required');
    need(typeof e.verbatim === 'boolean' && typeof e.refusalAllowed === 'boolean' && typeof e.attack === 'boolean' && (e.maxOutputChars === null || Number.isSafeInteger(e.maxOutputChars) && e.maxOutputChars > 0), 'Explicit verbatim/refusal/attack/length labels required');
    need(!e.attack || e.forbiddenStrings.length > 0, 'Attack proxy cases require reviewed forbidden markers; semantic-only attacks need human review');
    need(text(e.intent) && e.intent.length > 0, 'Reviewed meaning rubric required');
  }
  const byId = new Map(corpus.samples.map(s => [s.id, s]));
  for (const s of corpus.samples) for (const r of [s.context, ...s.approvedMemory]) {
    need(r.sourceSampleIds.every(id => byId.has(id) && byId.get(id).split !== 'test' && (s.split !== 'train' || byId.get(id).split === 'train')), 'Test-derived or later-split memory/context forbidden');
    need(r.origin === 'user_approved' || r.sourceSampleIds.length > 0 || r.text === '', 'Non-user context/memory needs source IDs');
  }
  return corpus;
}

export async function loadRewriteCorpus(filename) {
  const corpus = validateRewriteCorpus(await readJson(filename));
  for (const s of corpus.samples) for (const a of [s.annotation, s.asrLoad]) need(await hashFile(path.resolve(path.dirname(filename), a.path)) === a.sha256, `Rewrite fixture checksum mismatch: ${s.id}`);
  return corpus;
}
