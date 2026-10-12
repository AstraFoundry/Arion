import { erasure, tokens } from './metrics.mjs';

// Match reference units to the longest prefix that never changes afterwards.
// Requires independently reviewed unit timestamps; not a model confidence proxy.
export function stableUnitDelays(referenceTimeline, events, audioStartMs, unit) {
  if (!referenceTimeline) return [];
  const eventTokens = events.map(event => tokens(event.text, unit));
  const final = eventTokens.at(-1);
  if (final.length !== referenceTimeline.length || final.some((token, i) => token !== referenceTimeline[i].text)) return [];
  return final.map((_, index) => {
    const eventIndex = eventTokens.findIndex((current, i) => current.length > index && current.slice(0, index + 1).every((token, j) => token === final[j]) && eventTokens.slice(i).every(next => next.length > index && next.slice(0, index + 1).every((token, j) => token === final[j])));
    return events[eventIndex].ms - audioStartMs - referenceTimeline[index].endMs;
  });
}

export function streamingMetrics(sample, response, pipeline) {
  if (!pipeline.streamingMetrics) return null;
  const start = response.timings.audioStartMs;
  const intervals = sample.speechIntervals ?? [];
  const endpoints = response.streaming.endpointEventsMs.map(ms => ms - start);
  // Intended turns include internal thinking pauses; acoustic silence alone is
  // not a ground-truth permission to finalize a user's sentence.
  const turns = sample.referenceTurns ?? [];
  const premature = endpoints.filter(ms => turns.some(turn => ms >= turn.startMs && ms < turn.endMs - 50)).length;
  const expected = turns.map(turn => turn.endMs);
  const lags = [];
  let missed = 0;
  const used = new Set();
  for (let i = 0; i < expected.length; i++) {
    const end = expected[i];
    const nextStart = turns[i + 1]?.startMs ?? Infinity;
    const deadline = Math.min(end + pipeline.endpointSilenceMs + 500, nextStart);
    const index = endpoints.findIndex((ms, j) => !used.has(j) && ms >= end - 50 && ms <= deadline);
    if (index < 0) missed++;
    else { used.add(index); lags.push(endpoints[index] - end); }
  }
  const onset = intervals[0]?.[0];
  const end = intervals.at(-1)?.[1];
  const previewErasure = erasure(response.previewEvents, sample.unit);
  const previewFirst = response.previewEvents.find(event => event.text.trim().length > 0 && !event.final)?.ms;
  const referenceTail = tokens(sample.reference, sample.unit).slice(-3);
  const finalTail = tokens(response.rawText, sample.unit).slice(-3);
  return {
    previewFirstPartialMs: onset === undefined || previewFirst === undefined ? null : previewFirst - start - onset,
    previewErased: previewErasure.erased, previewFinalUnits: previewErasure.finalUnits,
    tailExactMatch: referenceTail.length === 0 ? null : JSON.stringify(referenceTail) === JSON.stringify(finalTail),
    endpointCount: endpoints.length, prematureEndpoints: premature, expectedEndpoints: expected.length, missedEndpoints: missed, endpointLagsMs: lags,
    annotationFirstPartialMs: onset === undefined || response.timings.firstPartialMs === null ? null : response.timings.firstPartialMs - start - onset,
    annotationStopToReviewMs: end === undefined ? null : response.timings.reviewMs - start - end,
    stableUnitDelaysMs: stableUnitDelays(sample.referenceTimeline, response.events, start, sample.unit),
    stableEligibleUnits: sample.referenceTimeline?.length ?? 0,
    maxQueueAgeMs: response.streaming.maxQueueAgeMs,
    processedSamples: response.streaming.processedSamples, droppedSamples: response.streaming.droppedSamples,
  };
}
