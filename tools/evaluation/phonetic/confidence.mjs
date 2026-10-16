import { errorCounts } from "../src/metrics.mjs";

// Pairwise form gives half credit to ties and makes score direction explicit.
export function riskAuc(observations) {
  const positives = observations.filter(row => row.error);
  const negatives = observations.filter(row => !row.error);
  if (!positives.length || !negatives.length) return null;
  let wins = 0;
  for (const positive of positives) for (const negative of negatives)
    wins += positive.risk > negative.risk ? 1 : positive.risk === negative.risk ? 0.5 : 0;
  return wins / (positives.length * negatives.length);
}

export function scoreConfidence(samples, evidence, threshold) {
  const observations = [];
  let missingScores = 0;
  for (const sample of samples.filter(sample => sample.reference !== "")) {
    const result = evidence.get(sample.id);
    if (!result) throw new Error("Missing acoustic observation");
    const scores = result.segments.flatMap(segment => segment.logScores);
    if (!scores.length) { missingScores++; continue; }
    if (scores.some(score => !Number.isFinite(score))) throw new Error("Invalid acoustic score");
    observations.push({ error: errorCounts(sample.reference, result.asrText, sample.unit).errors > 0,
      risk: -Math.min(...scores), flagged: scores.some(score => score < threshold) });
  }
  const count = predicate => observations.filter(predicate).length;
  const tp = count(row => row.error && row.flagged), fp = count(row => !row.error && row.flagged);
  const fn = count(row => row.error && !row.flagged), tn = count(row => !row.error && !row.flagged);
  return { samples: observations.length, missingScores, errorSamples: count(row => row.error),
    threshold, risk: "negative minimum emitted-token log score", auc: riskAuc(observations),
    truePositive: tp, falsePositive: fp, falseNegative: fn, trueNegative: tn,
    precision: tp + fp ? tp / (tp + fp) : null, recall: tp + fn ? tp / (tp + fn) : null,
    falsePositiveRate: fp + tn ? fp / (fp + tn) : null };
}
