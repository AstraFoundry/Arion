// Scores arion-rewrite-bench rows against a text corpus with the diagnostic
// exact-string proxies. Delivered text is the validated proposal or the
// completed local cleanup when organization is rejected; proposals are scored
// separately. Legacy rows without cleanup retain their recorded source.
import fs from "node:fs/promises";
import { diagnosticMetrics } from "../diagnostic/metrics.mjs";
import { distribution } from "../src/metrics.mjs";

export function scoreText(corpus, rows) {
  const byId = new Map(corpus.samples.map((s) => [s.id, s]));
  if (rows.length !== corpus.samples.length || new Set(rows.map((r) => r.caseId)).size !== rows.length)
    throw new Error("Rows must cover every corpus case exactly once");
  const scored = rows.map((row) => {
    const sample = byId.get(row.caseId);
    if (!sample) throw new Error(`Unknown case ${row.caseId}`);
    const delivered = diagnosticMetrics(sample, { text: row.text, truncated: false });
    const proposal = diagnosticMetrics(sample, { text: row.bypassed ? row.text : row.proposal, truncated: row.truncated });
    // The rewriter's input: the corrected source when the run applied rule
    // correction, else the corpus text.
    const source = diagnosticMetrics(sample, { text: row.source ?? sample.sourceText, truncated: false });
    const cleanup = diagnosticMetrics(sample, { text: row.cleanup?.version ? row.cleanup.text : row.source ?? sample.sourceText, truncated: false });
    return { row, sample, delivered, proposal, source, cleanup };
  });
  const count = (f) => scored.filter(f).length;
  const mean = (values) => {
    const finite = values.filter(Number.isFinite);
    return finite.length ? finite.reduce((a, b) => a + b, 0) / finite.length : null;
  };
  const finals = scored.filter((s) => !s.row.bypassed).map((s) => s.row.generationMs);
  const lives = scored.flatMap((s) => s.row.live ?? []);
  return {
    cases: scored.length,
    sourcePass: count((s) => s.source.constraintPass),
    cleanupPass: count((s) => s.cleanup.constraintPass),
    cleanupChanged: count((s) => s.row.cleanup?.version && s.row.cleanup.text !== s.row.source),
    proposalPass: count((s) => s.proposal.constraintPass),
    deliveredPass: count((s) => s.delivered.constraintPass),
    accepted: count((s) => s.row.verdict === "accepted"),
    rejected: Object.fromEntries(
      [...new Set(scored.map((s) => s.row.verdict))]
        .filter((v) => v !== "accepted" && v !== "bypassed")
        .map((v) => [v, count((s) => s.row.verdict === v)]),
    ),
    // A rejected-but-passing proposal is lost benefit; an accepted failing
    // one passed validation and still missed a constraint.
    acceptedFailing: count((s) => s.row.verdict === "accepted" && !s.proposal.constraintPass),
    rejectedPassing: count((s) => s.row.verdict !== "accepted" && !s.row.bypassed && s.proposal.constraintPass),
    worsened: count((s) => s.source.constraintPass && !s.delivered.constraintPass),
    referenceEditRatioMean: mean(scored.map((s) => s.delivered.referenceEditRatio)),
    sourceEditRatioMean: mean(scored.map((s) => s.source.referenceEditRatio)),
    finalMs: distribution(finals),
    liveEdits: lives.length,
    liveAccepted: lives.filter((l) => l.verdict === "accepted").length,
    liveMs: distribution(lives.map((l) => l.totalMs)),
    // Rows from runs before scenes were recorded carry no scene.
    sceneAgreement: count((s) => s.row.scene === s.sample.scenario),
    byLanguage: Object.fromEntries(
      [...new Set(scored.map((s) => s.sample.language))].sort().map((language) => {
        const group = scored.filter((s) => s.sample.language === language);
        const n = (f) => group.filter(f).length;
        return [language, { cases: group.length, sourcePass: n((s) => s.source.constraintPass),
          cleanupPass: n((s) => s.cleanup.constraintPass), proposalPass: n((s) => s.proposal.constraintPass),
          deliveredPass: n((s) => s.delivered.constraintPass),
          worsened: n((s) => s.source.constraintPass && !s.delivered.constraintPass) }];
      }),
    ),
    byScene: Object.fromEntries(
      [...new Set(scored.map((s) => s.sample.scenario))].sort().map((scene) => {
        const group = scored.filter((s) => s.sample.scenario === scene);
        const n = (f) => group.filter(f).length;
        return [scene, { cases: group.length, sourcePass: n((s) => s.source.constraintPass), deliveredPass: n((s) => s.delivered.constraintPass),
          worsened: n((s) => s.source.constraintPass && !s.delivered.constraintPass), sceneAgreement: n((s) => s.row.scene === scene) }];
      }),
    ),
    byKind: Object.fromEntries(
      [...new Set(scored.map((s) => s.sample.tags?.[0] ?? "untagged"))].sort().map((kind) => {
        const group = scored.filter((s) => (s.sample.tags?.[0] ?? "untagged") === kind);
        const n = (f) => group.filter(f).length;
        return [kind, { cases: group.length, sourcePass: n((s) => s.source.constraintPass), deliveredPass: n((s) => s.delivered.constraintPass),
          worsened: n((s) => s.source.constraintPass && !s.delivered.constraintPass) }];
      }),
    ),
    perCase: scored.map((s) => ({ id: s.row.caseId, scene: s.row.scene, verdict: s.row.verdict, pass: s.delivered.constraintPass, sourcePass: s.source.constraintPass })),
  };
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const [corpusPath, ...rowPaths] = process.argv.slice(2);
  const corpus = JSON.parse(await fs.readFile(corpusPath, "utf8"));
  for (const p of rowPaths) {
    const rows = (await fs.readFile(p, "utf8")).trim().split("\n").map(JSON.parse);
    const { perCase, ...summary } = scoreText(corpus, rows);
    console.log(p, JSON.stringify(summary));
    if (process.env.VERBOSE)
      for (const c of perCase)
        console.log(" ", c.id.padEnd(18), (c.scene ?? "").padEnd(7), c.verdict.padEnd(20), c.pass ? "PASS" : "fail", c.sourcePass ? "(src pass)" : "");
  }
}
