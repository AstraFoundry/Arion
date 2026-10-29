export const NORMALIZATION = 'arion-nfkc-lower-punctuation-space-v1';

export function normalize(text) {
  if (typeof text !== 'string') throw new Error('Text must be a string');
  return text.normalize('NFKC').toLowerCase().replace(/[\p{P}\p{S}]/gu, ' ').replace(/\s+/gu, ' ').trim();
}

export function tokens(text, unit) {
  const normalized = normalize(text);
  if (unit === 'cer') return [...normalized.replace(/\s/gu, '')];
  if (unit === 'wer') return normalized ? normalized.split(' ') : [];
  if (unit === 'mer') return normalized.match(/\p{Script=Han}|[^\p{Script=Han}\s]+/gu) ?? [];
  throw new Error(`Unsupported error unit: ${unit}`);
}

// Ties prefer substitution, then deletion, then insertion. Total distance is invariant.
export function edits(reference, hypothesis) {
  let previous = Array.from({ length: hypothesis.length + 1 }, (_, i) => ({ s: 0, d: 0, i, n: i }));
  for (let r = 1; r <= reference.length; r++) {
    const current = [{ s: 0, d: r, i: 0, n: r }];
    for (let h = 1; h <= hypothesis.length; h++) {
      if (reference[r - 1] === hypothesis[h - 1]) current.push({ ...previous[h - 1] });
      else {
        const sub = { ...previous[h - 1], s: previous[h - 1].s + 1, n: previous[h - 1].n + 1 };
        const del = { ...previous[h], d: previous[h].d + 1, n: previous[h].n + 1 };
        const ins = { ...current[h - 1], i: current[h - 1].i + 1, n: current[h - 1].n + 1 };
        current.push([sub, del, ins].reduce((best, item) => item.n < best.n ? item : best));
      }
    }
    previous = current;
  }
  const { s, d, i, n } = previous[hypothesis.length];
  return { substitutions: s, deletions: d, insertions: i, errors: n, referenceUnits: reference.length };
}

export function errorCounts(reference, hypothesis, unit) {
  return edits(tokens(reference, unit), tokens(hypothesis, unit));
}

export function quantile(values, probability) {
  if (!values.length) return null;
  if (values.some(v => !Number.isFinite(v))) throw new Error('Quantiles require finite observations');
  if (!(probability >= 0 && probability <= 1)) throw new Error('Invalid quantile');
  const sorted = [...values].sort((a, b) => a - b);
  const position = (sorted.length - 1) * probability;
  const lower = Math.floor(position);
  return sorted[lower] + (sorted[Math.ceil(position)] - sorted[lower]) * (position - lower);
}

export function distribution(values) {
  return { count: values.length, p50: quantile(values, 0.5), p95: quantile(values, 0.95), p99: quantile(values, 0.99) };
}

export function erasure(events, unit) {
  let previous = [];
  let erased = 0;
  for (const event of events) {
    const current = tokens(event.text, unit);
    let common = 0;
    while (common < previous.length && common < current.length && previous[common] === current[common]) common++;
    erased += previous.length - common;
    previous = current;
  }
  return { erased, finalUnits: previous.length, normalized: previous.length ? erased / previous.length : null };
}

export function ranking(candidates, rankedIds, acceptableTexts, unit) {
  const ids = candidates.map(c => c.id);
  if (new Set(ids).size !== ids.length || candidates.some(c => typeof c.id !== 'string' || !c.id || typeof c.text !== 'string' || typeof c.source !== 'string' || !c.source)) throw new Error('Invalid candidate provenance');
  if (rankedIds.length !== ids.length || new Set(rankedIds).size !== ids.length || rankedIds.some(id => !ids.includes(id))) throw new Error('Ranking must be an exact permutation of producer IDs');
  const accepted = new Set(acceptableTexts.map(text => JSON.stringify(tokens(text, unit))));
  const relevant = new Set(candidates.filter(c => accepted.has(JSON.stringify(tokens(c.text, unit)))).map(c => c.id));
  const position = rankedIds.findIndex(id => relevant.has(id));
  return { eligible: 1, oracle: relevant.size ? 1 : 0, top1: position === 0 ? 1 : 0, reciprocalRank: position < 0 ? 0 : 1 / (position + 1) };
}

function occurrences(text, value) {
  let count = 0;
  let index = 0;
  while ((index = text.indexOf(value, index)) >= 0) { count++; index += value.length; }
  return count;
}

export function scoreSample(sample, response) {
  const raw = errorCounts(sample.reference, response.rawText, sample.unit);
  const selected = errorCounts(sample.reference, response.selectedText, sample.unit);
  const final = errorCounts(sample.reference, response.finalText, sample.unit);
  const protectedSpans = sample.protectedSpans ?? [];
  const preserved = protectedSpans.filter(span => occurrences(response.finalText, span.text) >= occurrences(sample.reference, span.text)).length;
  return {
    raw, selected, final, streaming: null,
    erasure: erasure(response.events, sample.unit),
    ranking: sample.reference ? ranking(response.candidates, response.rankedIds, sample.acceptableCandidates ?? [sample.reference], sample.unit) : null,
    silence: sample.reference === '', falseEmission: sample.reference === '' && response.finalText.trim().length > 0,
    harmfulCorrection: final.errors > selected.errors,
    protectedCount: protectedSpans.length, protectedPreserved: preserved,
    firstPartialMs: response.timings.firstPartialMs === null ? null : response.timings.firstPartialMs - response.timings.speechStartMs,
    stopToReviewMs: response.timings.reviewMs - response.timings.speechEndMs,
    rankerMs: response.timings.rankerMs, loadMs: response.timings.loadMs,
    peakRssBytes: response.resources.peakRssBytes, computeMs: response.timings.computeMs, audioDurationMs: sample.audio.durationMs,
    rtf: response.timings.computeMs / sample.audio.durationMs,
  };
}

export function aggregate(rows) {
  const sum = fn => rows.reduce((total, row) => total + fn(row), 0);
  const count = rows.length;
  const rate = (num, den) => den ? num / den : null;
  const units = {};
  for (const unit of ['cer', 'wer', 'mer']) {
    const subset = rows.filter(row => row.unit === unit && !row.metrics.silence);
    for (const stage of ['raw', 'selected', 'final']) {
      const referenceUnits = subset.reduce((n, row) => n + row.metrics[stage].referenceUnits, 0);
      const errors = subset.reduce((n, row) => n + row.metrics[stage].errors, 0);
      units[`${stage}_${unit}`] = { errors, referenceUnits, rate: rate(errors, referenceUnits) };
    }
  }
  const rankingRows = rows.filter(row => row.metrics.ranking !== null);
  const rankingSum = key => rankingRows.reduce((n, row) => n + row.metrics.ranking[key], 0);
  const oracle = rankingSum('oracle');
  const silences = sum(row => Number(row.metrics.silence));
  const protectedCount = sum(row => row.metrics.protectedCount);
  const erased = sum(row => row.metrics.erasure.erased);
  const finalUnits = sum(row => row.metrics.erasure.finalUnits);
  const metricValues = key => rows.map(row => row.metrics[key]).filter(value => value !== null);
  const streamingRows = rows.filter(row => row.metrics.streaming !== null);
  const streams = streamingRows.map(row => row.metrics.streaming);
  const streamSum = key => streams.reduce((n, stream) => n + stream[key], 0);
  const streamValues = key => streams.map(stream => stream[key]).filter(value => value !== null);
  return {
    count, ...units,
    streaming: {
      count: streams.length, prematureEndpoints: streamSum('prematureEndpoints'), missedEndpointRate: rate(streamSum('missedEndpoints'), streamSum('expectedEndpoints')),
      endpointLagMs: distribution(streams.flatMap(s => s.endpointLagsMs)),
      annotatedFirstPartialMs: distribution(streamValues('annotationFirstPartialMs')), annotatedStopToReviewMs: distribution(streamValues('annotationStopToReviewMs')),
      previewFirstPartialMs: distribution(streamValues('previewFirstPartialMs')), previewNormalizedErasure: rate(streamSum('previewErased'), streamSum('previewFinalUnits')),
      stableUnitDelayMs: distribution(streams.flatMap(s => s.stableUnitDelaysMs)), stableUnitCoverage: rate(streams.reduce((n, s) => n + s.stableUnitDelaysMs.length, 0), streamSum('stableEligibleUnits')),
      droppedSampleRate: rate(streamSum('droppedSamples'), streamSum('processedSamples') + streamSum('droppedSamples')),
      maxQueueAgeMs: distribution(streamValues('maxQueueAgeMs')), tailExactMatchRate: rate(streams.filter(s => s.tailExactMatch === true).length, streams.filter(s => s.tailExactMatch !== null).length),
    },
    ranking: { count: rankingRows.length, oracleRecall: rate(oracle, rankingRows.length), top1: rate(rankingSum('top1'), rankingRows.length), conditionalTop1: rate(rankingSum('top1'), oracle), mrr: rate(rankingSum('reciprocalRank'), rankingRows.length) },
    silence: { count: silences, falseEmissionRate: rate(sum(row => Number(row.metrics.falseEmission)), silences) },
    harmfulCorrectionRate: rate(sum(row => Number(row.metrics.harmfulCorrection)), count),
    protectedSpanRecall: rate(sum(row => row.metrics.protectedPreserved), protectedCount),
    normalizedErasure: rate(erased, finalUnits),
    firstPartialMissingCount: sum(row => Number(!row.metrics.silence && row.metrics.firstPartialMs === null)),
    corpusRtf: rate(sum(row => row.metrics.computeMs), sum(row => row.metrics.audioDurationMs)),
    peakRssBytes: distribution(metricValues('peakRssBytes')), loadMs: distribution(metricValues('loadMs')),
    firstPartialMs: distribution(metricValues('firstPartialMs')),
    stopToReviewMs: distribution(metricValues('stopToReviewMs')),
    rankerMs: distribution(metricValues('rankerMs')), rtf: distribution(metricValues('rtf')),
  };
}
