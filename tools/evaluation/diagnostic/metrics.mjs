import { tokens, edits } from "../src/metrics.mjs";

export function diagnosticMetrics(sample, response) {
  const text = response.text;
  const requiredMisses = sample.requiredStrings.filter(
    (s) => !text.includes(s),
  ).length;
  const forbiddenHits = sample.forbiddenStrings.filter((s) =>
    text.includes(s),
  ).length;
  const protectedMisses = sample.protected.filter(
    (s) => !text.includes(s),
  ).length;
  const verbatimViolation =
    sample.mode === "verbatim" && text !== sample.sourceText;
  const unit = { zh: "cer", en: "wer", mixed: "mer" }[sample.language];
  const distance = Math.min(
    ...sample.references.map((ref) => {
      const d = edits(tokens(ref, unit), tokens(text, unit));
      return d.errors / d.referenceUnits;
    }),
  );
  return {
    requiredMisses,
    forbiddenHits,
    protectedMisses,
    verbatimViolation,
    constraintPass:
      !requiredMisses &&
      !forbiddenHits &&
      !protectedMisses &&
      !verbatimViolation &&
      !response.truncated,
    referenceEditRatio: distance,
    generationMs: response.generationMs,
    firstTokenMs: response.firstTokenMs,
    truncated: response.truncated,
    peakRssBytes: response.peakRssBytes,
  };
}
