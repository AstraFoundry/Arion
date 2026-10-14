// Scores arion-ocr rows: character error of the recognized window text
// against each case's true visible text, per recognition level, and the
// recognition time. Whitespace and line order are ignored by comparing
// characters in reading order without spaces.
import fs from "node:fs/promises";
import { edits } from "../src/metrics.mjs";
import { distribution } from "../src/metrics.mjs";

const chars = (s) => [...s.replace(/\s+/g, "")];

export function scoreOCR(corpus, rows) {
  const truth = new Map(corpus.samples.map((s) => [s.id, s.contextFixture?.screenTruth ?? ""]));
  const levels = {};
  for (const row of rows) {
    if (!truth.has(row.caseId)) throw new Error(`Unknown case ${row.caseId}`);
    const ref = chars(truth.get(row.caseId));
    const d = edits(ref, chars(row.text));
    const l = (levels[row.level] ??= { images: 0, errors: 0, referenceChars: 0, failed: 0, ms: [] });
    l.images++;
    l.errors += d.errors;
    l.referenceChars += d.referenceUnits;
    l.failed += row.error ? 1 : 0;
    l.ms.push(row.ocrMs);
  }
  return Object.fromEntries(Object.entries(levels).map(([level, l]) => [level,
    { images: l.images, failed: l.failed, cer: l.errors / l.referenceChars, ocrMs: distribution(l.ms) }]));
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const [corpusPath, rowsPath] = process.argv.slice(2);
  const corpus = JSON.parse(await fs.readFile(corpusPath, "utf8"));
  const rows = (await fs.readFile(rowsPath, "utf8")).trim().split("\n").map(JSON.parse);
  console.log(JSON.stringify(scoreOCR(corpus, rows), null, 1));
}
