import { spawn } from 'node:child_process';
import { createInterface } from 'node:readline';
import { isDeepStrictEqual } from 'node:util';
import { ranking } from './metrics.mjs';

export function requireValue(condition, message) {
  if (!condition) throw new Error(message);
}

export function validateResponse(response, request) {
  requireValue(response?.protocol === 1 && response.requestId === request.requestId && response.execution === 'live', 'Response identity or live execution missing');
  requireValue(isDeepStrictEqual(response.effectiveFeatures, request.variant.features), 'Driver did not apply requested ablation features');
  requireValue(isDeepStrictEqual(response.effectivePipeline, request.pipeline), 'Driver changed controlled pipeline settings');
  requireValue(response.cacheState === request.cacheState, 'Cache-state attestation mismatch');
  requireValue(typeof response.rawText === 'string' && typeof response.selectedText === 'string' && typeof response.finalText === 'string', 'Missing raw/final transcript');
  requireValue(Array.isArray(response.events) && response.events.length > 0, 'Missing streaming event trace');
  let prior = -1;
  for (const event of response.events) {
    requireValue(typeof event.text === 'string' && Number.isFinite(event.ms) && event.ms >= prior && event.ms >= 0, 'Nonmonotonic stream trace');
    prior = event.ms;
  }
  requireValue(response.events.at(-1).text === response.rawText && response.events.at(-1).final === true, 'Trace must include the actual final ASR event');
  const time = response.timings;
  requireValue(time && ['speechStartMs', 'speechEndMs', 'finalMs', 'reviewMs', 'computeMs', 'loadMs'].every(key => Number.isFinite(time[key]) && time[key] >= 0), 'Invalid monotonic timings');
  requireValue(time.speechStartMs <= time.speechEndMs && time.speechEndMs <= time.finalMs && time.finalMs <= time.reviewMs && prior === time.finalMs, 'Timing order mismatch');
  requireValue(time.firstPartialMs === null || (Number.isFinite(time.firstPartialMs) && time.firstPartialMs >= time.speechStartMs && time.firstPartialMs <= time.finalMs), 'Invalid first-partial timing');
  const partial = response.events.find(event => !event.final && event.text.trim().length > 0);
  requireValue(time.firstPartialMs === (partial?.ms ?? null), 'First-partial time differs from event trace');
  requireValue(time.rankerMs === null || (Number.isFinite(time.rankerMs) && time.rankerMs >= 0), 'Invalid ranker timing');
  requireValue(request.variant.features.ranker === 'source' ? time.rankerMs === null : time.rankerMs !== null, 'Ranker timing missing or fabricated for source baseline');
  requireValue(Array.isArray(response.candidates) && Array.isArray(response.rankedIds), 'Missing actual candidate producer result');
  ranking(response.candidates, response.rankedIds, [], 'mer');
  if (request.variant.features.ranker === 'source') requireValue(isDeepStrictEqual(response.rankedIds, response.candidates.map(c => c.id)), 'Source baseline reordered candidates');
  const selected = response.candidates.find(candidate => candidate.id === response.rankedIds[0]);
  requireValue(response.selectedText === (selected?.text ?? response.rawText), 'Selected transcript differs from top candidate');
  if (request.variant.features.correction === 'none') requireValue(response.finalText === response.selectedText, 'Correction disabled but selected transcript changed');
  requireValue(Number.isFinite(response.resources?.peakRssBytes) && response.resources.peakRssBytes > 0, 'Missing peak process RSS evidence');
  if (request.pipeline.streamingMetrics) {
    requireValue(Number.isFinite(time.audioStartMs) && time.audioStartMs >= 0, 'Streaming replay requires measured audio-start time');
    requireValue(Array.isArray(response.previewEvents) && response.previewEvents.length > 0 && response.previewEvents.every((event, index) => typeof event.text === 'string' && Number.isFinite(event.ms) && event.ms >= 0 && (index === 0 || event.ms >= response.previewEvents[index - 1].ms)), 'Missing or invalid rendered preview trace');
    requireValue(response.previewEvents.at(-1).text === response.rawText && response.previewEvents.at(-1).ms >= time.finalMs && response.previewEvents.at(-1).ms <= time.reviewMs, 'Rendered final ASR preview timing mismatch');
    const stream = response.streaming;
    requireValue(Array.isArray(stream?.endpointEventsMs) && stream.endpointEventsMs.every((ms, index) => Number.isFinite(ms) && ms >= time.audioStartMs && ms <= time.finalMs && (index === 0 || ms >= stream.endpointEventsMs[index - 1])), 'Invalid observed endpoint trace');
    requireValue(Number.isSafeInteger(stream.processedSamples) && stream.processedSamples > 0 && Number.isSafeInteger(stream.droppedSamples) && stream.droppedSamples >= 0 && Number.isFinite(stream.maxQueueAgeMs) && stream.maxQueueAgeMs >= 0, 'Streaming sample/queue evidence missing');
  }
  return response;
}

// Persistent JSONL driver: models can remain resident between explicit warm requests.
// Stdout is protocol only. Stderr is discarded to avoid leaking fixture content.
export class Driver {
  constructor(config, cwd) {
    requireValue(typeof config.executable === 'string' && config.executable.length > 0 && Array.isArray(config.args) && config.args.every(arg => typeof arg === 'string'), 'Driver requires executable and argument array');
    requireValue(Number.isInteger(config.timeoutMs) && config.timeoutMs > 0 && config.timeoutMs <= 43200000, 'Invalid driver timeout');
    this.timeoutMs = config.timeoutMs;
    this.pending = null;
    this.failure = null;
    this.sequence = 0;
    this.process = spawn(config.executable, config.args, { cwd, stdio: ['pipe', 'pipe', 'ignore'], shell: false });
    this.process.stdin.on('error', () => this.fail(new Error('Driver stdin failed')));
    this.process.on('error', () => this.fail(new Error('Driver could not start')));
    this.process.on('exit', code => this.fail(new Error(`Driver exited (${code})`)));
    // Bound a line before readline accumulates unlimited output.
    let lineBytes = 0;
    this.process.stdout.on('data', chunk => {
      for (const byte of chunk) {
        lineBytes = byte === 10 ? 0 : lineBytes + 1;
        if (lineBytes > 8 * 1024 * 1024) { this.fail(new Error('Driver response exceeds limit')); break; }
      }
    });
    this.lines = createInterface({ input: this.process.stdout, crlfDelay: Infinity });
    this.lines.on('line', line => {
      if (!this.pending) { this.fail(new Error('Unsolicited driver response')); return; }
      try {
        const response = JSON.parse(line);
        requireValue(response.requestId === this.pending.id && response.protocol === 1, 'Driver protocol/identity mismatch');
        const { resolve, timer } = this.pending;
        this.pending = null;
        clearTimeout(timer);
        resolve(response);
      } catch { this.fail(new Error('Invalid driver JSONL response')); }
    });
  }
  fail(error) {
    this.failure ??= error;
    if (this.pending) {
      clearTimeout(this.pending.timer);
      this.pending.reject(error);
      this.pending = null;
    }
    this.process.kill();
  }
  request(payload) {
    requireValue(!this.pending, 'Driver requests must be sequential');
    if (this.failure) return Promise.reject(this.failure);
    const requestId = String(++this.sequence);
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => this.fail(new Error('Driver request timed out')), this.timeoutMs);
      this.pending = { id: requestId, resolve, reject, timer };
      this.process.stdin.write(`${JSON.stringify({ ...payload, protocol: 1, requestId })}\n`);
    });
  }
  close() {
    this.lines.close();
    this.process.kill();
  }
}
