import path from 'node:path';
import { isDeepStrictEqual } from 'node:util';
import { digest } from './manifest.mjs';
import { requireValue as need } from './protocol.mjs';
import { REWRITE_SCHEMA } from './rewrite-manifest.mjs';

export function rewriteRequest(sample, block, config, corpusFile, warmup = false) {
  const f = block.variant.features;
  // No reference, expected style labels, protected entities or gold rubric enters inference.
  const input = {
    sourceText: sample.sourceText, userInstruction: sample.userInstruction,
    context: f.context ? sample.context.text : '',
    approvedMemory: f.memory ? sample.approvedMemory.map(m => ({ id: m.id, text: m.text, approved: true })) : [],
    scenario: f.scenario ? sample.scenario : 'generic',
  };
  return {
    op: 'rewrite', schema: REWRITE_SCHEMA, sampleId: sample.id, seed: block.seed, warmup,
    cacheState: block.cacheState, loadState: block.loadState,
    variant: block.variant, pipeline: { ...config.pipeline, ...block.variant.pipelineOverrides }, input, inputHash: digest(input),
    identity: { session: sample.id, generation: 1, revision: 1 },
    controls: warmup ? [] : sample.controls,
    asrLoad: block.loadState === 'concurrent_asr' ? { ...sample.asrLoad, path: path.resolve(path.dirname(corpusFile), sample.asrLoad.path) } : null,
  };
}

function monotonic(events) {
  return Array.isArray(events) && events.every((e, i) => Number.isFinite(e.ms) && e.ms >= 0 && (i === 0 || e.ms >= events[i - 1].ms));
}

export function validateRewriteResponse(r, q) {
  need(r?.protocol === 1 && r.schema === REWRITE_SCHEMA && r.execution === 'live' && typeof r.requestId === 'string' && r.requestId.length > 0 && r.requestId === q.requestId, 'Rewrite response requires live identity/schema');
  need(isDeepStrictEqual(r.effectiveFeatures, q.variant.features) && isDeepStrictEqual(r.effectivePipeline, q.pipeline) && r.inputHash === q.inputHash && r.cacheState === q.cacheState && r.loadState === q.loadState, 'Rewrite treatment/input/cache/load mismatch');
  need(isDeepStrictEqual(r.identity, q.identity), 'Stale rewrite result identity');
  need(['rewritten', 'unchanged', 'rejected', 'refused', 'cancelled'].includes(r.decision) && typeof r.modelText === 'string' && (r.candidateText === null || typeof r.candidateText === 'string'), 'Invalid rewrite decision/text');
  need(['rewritten', 'unchanged'].includes(r.decision) ? r.candidateText !== null : r.candidateText === null, 'Decision/candidate mismatch');
  if (r.decision === 'unchanged') need(r.candidateText === q.input.sourceText, 'Unchanged decision changed source');
  if (q.variant.features.engine === 'verbatim' && !q.controls.length) need(r.decision === 'unchanged', 'Verbatim control must return exact source');
  const t = r.timings;
  need(t && ['loadMs', 'requestStartMs', 'prefillStartMs', 'prefillEndMs', 'decodeEndMs', 'validationEndMs', 'reviewMs'].every(k => Number.isFinite(t[k]) && t[k] >= 0), 'Measured rewrite timing stages required');
  need(t.requestStartMs <= t.prefillStartMs && t.prefillStartMs <= t.prefillEndMs && t.prefillEndMs <= t.decodeEndMs && t.decodeEndMs <= t.validationEndMs && t.validationEndMs <= t.reviewMs && t.loadMs <= t.prefillStartMs - t.requestStartMs, 'Rewrite monotonic stage order mismatch');
  need(monotonic(r.tokenEvents) && r.tokenEvents.every(e => Number.isSafeInteger(e.tokenId) && e.tokenId >= 0 && typeof e.delta === 'string' && e.ms >= t.prefillEndMs && e.ms <= t.decodeEndMs), 'Real token IDs/deltas/timestamps required');
  need(Number.isSafeInteger(r.usage?.inputTokens) && r.usage.inputTokens >= 0 && r.usage.inputTokens <= q.pipeline.maxInputTokens && r.usage.outputTokens === r.tokenEvents.length && r.usage.outputTokens <= q.pipeline.maxNewTokens, 'Rewrite tokenizer usage/budget mismatch');
  need(t.firstTokenMs === (r.tokenEvents[0]?.ms ?? null), 'First-token time differs from token trace');
  if (q.variant.features.engine === 'llm') need(r.tokenEvents.map(e => e.delta).join('') === r.modelText, 'Generated token trace differs from model output');
  else need(r.tokenEvents.length === 0 && r.usage.inputTokens === 0, 'Rule/verbatim controls cannot claim LLM tokens');
  need(monotonic(r.previewEvents) && r.previewEvents.every(e => typeof e.text === 'string' && isDeepStrictEqual(e.identity, q.identity) && e.ms >= t.requestStartMs && e.ms <= t.reviewMs), 'Actual native preview observations required');
  need(monotonic(r.lifecycleEvents) && r.lifecycleEvents.every(e => ['invalidated', 'cancel_ack', 'revision_discard', 'native_destroy'].includes(e.type)), 'Measured lifecycle evidence required');
  if (q.controls.length) {
    const controlMs = t.requestStartMs + q.controls[0].atMs;
    need(r.decision === 'cancelled' && r.lifecycleEvents.some(e => e.type === 'invalidated' && e.ms >= controlMs) && r.lifecycleEvents.some(e => e.type === 'cancel_ack' && e.ms >= controlMs), 'Cancellation/invalidation acknowledgment required');
    need(r.previewEvents.every(e => e.ms < controlMs), 'Stale preview published after invalidation');
  } else if (r.candidateText !== null) need(r.previewEvents.length > 0 && r.previewEvents.at(-1).text === r.candidateText && r.previewEvents.at(-1).ms >= t.validationEndMs, 'Final reviewed candidate must appear in native preview');
  need(Array.isArray(r.commitEvents) && r.commitEvents.length === 0, 'Candidate benchmark must never commit to a target');
  need(Number.isFinite(r.resources?.peakRssBytes) && r.resources.peakRssBytes > 0 && (r.resources.peakDeviceBytes === null || Number.isFinite(r.resources.peakDeviceBytes) && r.resources.peakDeviceBytes >= 0), 'RSS and explicit device-memory availability required');
  need(['process_high_water', 'process_request_sampled', 'process_tree_request_sampled'].includes(r.resources.rssScope), 'Explicit RSS measurement scope required');
  need(r.resources.rssScope === 'process_high_water' ? r.resources.rssSamplingIntervalMs === null : Number.isFinite(r.resources.rssSamplingIntervalMs) && r.resources.rssSamplingIntervalMs > 0, 'RSS sampling interval/scope mismatch');
  if (q.cacheState === 'cold' && q.variant.features.engine === 'llm') need(r.residency?.reset === true && r.residency.kvCacheReset === true && r.residency.modelReloaded === true, 'Cold LLM requests must release/reset and reload model/cache');
  if (q.variant.features.engine === 'llm') {
    need(r.residency?.kvCacheReset === true, 'Every independent rewrite fixture must reset session KV state');
    if (q.cacheState === 'warm' && !q.warmup) need(r.residency.modelReloaded === false && r.residency.reset === false, 'Measured warm requests must keep the model resident');
  }
  if (q.loadState === 'concurrent_asr') {
    const a = r.asr;
    need(a?.execution === 'live' && a.audioSha256 === q.asrLoad.sha256 && a.model === q.pipeline.asr && Number.isFinite(a.startedMs) && a.startedMs <= t.prefillStartMs && Number.isFinite(a.endedMs) && a.endedMs >= t.decodeEndMs && a.endedMs - a.startedMs >= q.asrLoad.durationMs * 0.98, 'ASR must overlap entire prefill/decode with real audio replay');
    need(monotonic(a.events) && a.events.length > 0 && a.events.every(e => typeof e.text === 'string' && e.ms >= a.startedMs && e.ms <= a.endedMs) && Number.isSafeInteger(a.processedSamples) && a.processedSamples > 0 && Number.isSafeInteger(a.droppedSamples) && a.droppedSamples >= 0 && Number.isFinite(a.maxQueueAgeMs) && a.maxQueueAgeMs >= 0, 'Concurrent ASR emissions/queue/sample evidence required');
  } else need(r.asr === null, 'Solo condition unexpectedly ran ASR');
  return r;
}
