import { quantile } from './metrics.mjs';

export function random(seed) {
  let state = seed >>> 0;
  return () => {
    state += 0x6D2B79F5;
    let t = state;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function shuffle(items, seed) {
  const result = [...items];
  const rng = random(seed);
  for (let i = result.length - 1; i > 0; i--) {
    const j = Math.floor(rng() * (i + 1));
    [result[i], result[j]] = [result[j], result[i]];
  }
  return result;
}

// Keep every utterance and repetition of a speaker in the same resampled cluster.
export function pairedBootstrap(baseline, treatment, statistic, { iterations = 2000, seed = 1729 } = {}) {
  if (!Number.isInteger(iterations) || iterations < 100) throw new Error('At least 100 bootstrap draws required');
  const key = row => `${row.caseId}:${row.seed}:${row.cacheState}`;
  const index = new Map(treatment.map(row => [key(row), row]));
  if (index.size !== treatment.length || baseline.length !== treatment.length || new Set(baseline.map(key)).size !== baseline.length) throw new Error('Unpaired or duplicated observations');
  const clusters = new Map();
  for (const row of baseline) {
    const paired = index.get(key(row));
    if (!paired || paired.clusterId !== row.clusterId || paired.unit !== row.unit) throw new Error('Pair/cluster mismatch');
    if (!clusters.has(row.clusterId)) clusters.set(row.clusterId, []);
    clusters.get(row.clusterId).push([row, paired]);
  }
  if (clusters.size < 2) return { status: 'insufficient_clusters', clusters: clusters.size, delta: null, ci95: null };
  const groups = [...clusters.values()];
  const rng = random(seed);
  const deltas = [];
  const pointA = statistic(baseline);
  const pointB = statistic(treatment);
  if (pointA === null || pointB === null) return { status: 'unavailable_metric', clusters: groups.length, delta: null, ci95: null };
  for (let i = 0; i < iterations; i++) {
    const pairs = Array.from({ length: groups.length }, () => groups[Math.floor(rng() * groups.length)]).flat();
    const a = statistic(pairs.map(pair => pair[0]));
    const b = statistic(pairs.map(pair => pair[1]));
    // Undefined strata must not silently disappear from the sampling distribution.
    if (a === null || b === null) return { status: 'undefined_resample', clusters: groups.length, delta: pointB - pointA, ci95: null };
    deltas.push(b - a);
  }
  return { status: 'ok', clusters: groups.length, iterations, seed, delta: pointB - pointA, ci95: [quantile(deltas, 0.025), quantile(deltas, 0.975)] };
}
