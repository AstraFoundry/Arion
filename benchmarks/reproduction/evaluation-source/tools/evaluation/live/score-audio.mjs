// Scores arion-live-bench rows: acoustic error of the recognized text
// against the spoken script, exact-string proxies of the inserted text
// against the text corpus, stop-to-final latency and live preview coverage.
import fs from "node:fs/promises";
import { diagnosticMetrics } from "../diagnostic/metrics.mjs";
import { distribution, errorCounts } from "../src/metrics.mjs";

const UNIT = { zh: "cer", en: "wer", mixed: "mer" };

export function scoreAudio(manifest, corpus, rows) {
  const text = new Map(corpus.samples.map((s) => [s.id, s]));
  const audio = new Map(manifest.samples.map((s) => [s.id, s]));
  if (rows.length !== manifest.samples.length || new Set(rows.map((r) => r.caseId)).size !== rows.length)
    throw new Error("Rows must cover every fixture exactly once");
  const policies = new Set(rows.map((r) => r.removeFillers ?? null));
  if (policies.size > 1) throw new Error("Mixed filler policies in one audio run");
  const asr = {};
  const scored = rows.map((row) => {
    const sample = text.get(row.caseId);
    const fixture = audio.get(row.caseId);
    if (!sample || !fixture) throw new Error(`Unknown case ${row.caseId}`);
    if (row.error) return { row, sample, fixture, failed: true };
    const unit = UNIT[sample.language];
    const counts = errorCounts(fixture.speech.replaceAll("|", " "), row.rawText, unit);
    const key = `${sample.language}:${unit}`;
    asr[key] ??= { errors: 0, referenceUnits: 0 };
    asr[key].errors += counts.errors;
    asr[key].referenceUnits += counts.referenceUnits;
    return {
      row, sample, fixture,
      final: diagnosticMetrics(sample, { text: row.finalText, truncated: false }),
      source: diagnosticMetrics(sample, { text: row.sourceText, truncated: false }),
    };
  });
  const ok = scored.filter((s) => !s.failed);
  const count = (f) => ok.filter(f).length;
  const multi = ok.filter((s) => s.fixture.segments > 1);
  return {
    cases: rows.length,
    removeFillers: [...policies][0] ?? null,
    failed: scored.length - ok.length,
    asrErrorRate: Object.fromEntries(Object.entries(asr).map(([k, v]) => [k, { ...v, rate: v.errors / v.referenceUnits }])),
    sourcePass: count((s) => s.source.constraintPass),
    finalPass: count((s) => s.final.constraintPass),
    worsened: count((s) => s.source.constraintPass && !s.final.constraintPass),
    improved: count((s) => !s.source.constraintPass && s.final.constraintPass),
    rewrite: Object.fromEntries([...new Set(ok.map((s) => s.row.rewrite || "none"))].map((v) => [v, count((s) => (s.row.rewrite || "none") === v)])),
    stopToFinalMs: distribution(ok.map((s) => s.row.stopToFinalMs)),
    firstPartialMs: distribution(ok.map((s) => s.row.firstPartialMs).filter((x) => x !== null)),
    // Live editing: multi-segment fixtures that showed a validated preview
    // before the speaker stopped.
    multiSegment: multi.length,
    previewBeforeStop: multi.filter((s) => s.row.firstPreviewMs !== null && s.row.firstPreviewMs < s.row.speechEndMs).length,
    previewEventsMean: multi.length ? multi.reduce((n, s) => n + s.row.previewEvents, 0) / multi.length : null,
    perCase: ok.map((s) => ({ id: s.row.caseId, pass: s.final.constraintPass, sourcePass: s.source.constraintPass, rewrite: s.row.rewrite, raw: s.row.rawText, final: s.row.finalText })),
  };
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const [manifestPath, corpusPath, ...rowPaths] = process.argv.slice(2);
  const manifest = JSON.parse(await fs.readFile(manifestPath, "utf8"));
  const corpus = JSON.parse(await fs.readFile(corpusPath, "utf8"));
  for (const p of rowPaths) {
    const rows = (await fs.readFile(p, "utf8")).trim().split("\n").map(JSON.parse);
    const { perCase, ...summary } = scoreAudio(manifest, corpus, rows);
    console.log(p, JSON.stringify(summary));
    if (process.env.VERBOSE)
      for (const c of perCase) console.log(" ", c.id.padEnd(16), c.pass ? "PASS" : "fail", c.sourcePass ? "(src)" : "     ", (c.rewrite ?? "").padEnd(10), c.raw, "=>", c.final.replaceAll("\n", "⏎"));
  }
}
