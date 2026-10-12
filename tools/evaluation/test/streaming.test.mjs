import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { validateExperiment, validateManifest } from '../src/manifest.mjs';
import { stableUnitDelays, streamingMetrics } from '../src/streaming.mjs';
import { inferenceRequest, plan } from '../src/runner.mjs';

const config = JSON.parse(await readFile(new URL('../../../benchmarks/configs/desktop-streaming.json', import.meta.url), 'utf8'));
const sample = { id: 'a', speakerId: 's1', sessionId: 'ss1', split: 'test', language: 'en', tags: ['internal-pause'], unit: 'wer', reference: 'one cat', context: '', approvedMemory: [], audio: { path: 'a.wav', sha256: 'a'.repeat(64), durationMs: 2500 }, speechIntervals: [[0, 900], [1200, 2000]], referenceTurns: [{ startMs: 0, endMs: 2000 }], timingSource: { kind: 'human_verified', sha256: 'b'.repeat(64) }, referenceTimeline: [{ text: 'one', endMs: 800 }, { text: 'cat', endMs: 2000 }] };
const response = { rawText: 'one cat', events: [{ text: 'one', ms: 850 }, { text: 'one dog', ms: 1500 }, { text: 'one cat', ms: 2200, final: true }], previewEvents: [{ text: 'one', ms: 870 }, { text: 'one dog', ms: 1530 }, { text: 'one cat', ms: 2250, final: true }], timings: { audioStartMs: 0, firstPartialMs: 850, reviewMs: 2260 }, streaming: { endpointEventsMs: [1000, 2200], processedSamples: 40000, droppedSamples: 0, maxQueueAgeMs: 20 } };

test('streaming sweep controls chunk and endpoint while fixing model treatments', () => {
  validateExperiment(config);
  assert.equal(config.variants.length, 15);
  assert.equal(plan(config, [sample]).measuredRequests, 45);
  const block = plan(config).blocks.find(block => block.variant.id === 'chunk320-end800');
  const request = inferenceRequest(sample, block, config, '/tmp/manifest.json');
  assert.equal(request.pipeline.chunkMs, 320);
  assert.equal(request.pipeline.endpointSilenceMs, 800);
  assert.equal(JSON.stringify(request).includes('referenceTurns'), false);
  assert.equal(JSON.stringify(request).includes('speechIntervals'), false);
});

test('thinking-pause endpoint is premature; real end is matched once', () => {
  const metrics = streamingMetrics(sample, response, config.pipeline);
  assert.equal(metrics.prematureEndpoints, 1);
  assert.equal(metrics.expectedEndpoints, 1);
  assert.equal(metrics.missedEndpoints, 0);
  assert.deepEqual(metrics.endpointLagsMs, [200]);
  assert.equal(metrics.annotationFirstPartialMs, 850);
  assert.equal(metrics.previewFirstPartialMs, 870);
  assert.equal(metrics.previewErased, 1);
  assert.equal(metrics.annotationStopToReviewMs, 260);
  assert.equal(metrics.tailExactMatch, true);
  assert.equal(streamingMetrics(sample, { ...response, streaming: { ...response.streaming, endpointEventsMs: [] } }, config.pipeline).missedEndpoints, 1);
});

test('stable latency requires final exact correctness and includes its coverage', () => {
  assert.deepEqual(stableUnitDelays(sample.referenceTimeline, response.events, 0, 'wer'), [50, 200]);
  assert.deepEqual(stableUnitDelays(sample.referenceTimeline, [{ text: 'one dog', ms: 2200 }], 0, 'wer'), []);
  assert.deepEqual(stableUnitDelays(undefined, response.events, 0, 'wer'), []);
});

test('streaming annotations cannot be silently fabricated or misaligned', () => {
  const manifest = { version: 1, license: 'synthetic-testing-only', consent: true, samples: [sample] };
  validateManifest(manifest);
  assert.throws(() => validateManifest({ ...manifest, samples: [{ ...sample, timingSource: { kind: 'unreviewed_vad', sha256: 'b'.repeat(64) } }] }), /timestamp provenance/);
  assert.throws(() => validateManifest({ ...manifest, samples: [{ ...sample, referenceTimeline: [{ text: 'wrong', endMs: 400 }] }] }), /timeline/);
  assert.throws(() => validateManifest({ ...manifest, samples: [{ ...sample, speechIntervals: [[0, 900], [800, 2000]] }] }), /interval/);
});
