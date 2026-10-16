// Aggregates live benchmark runs into a content-free summary: per-run
// proxy counts, latency distributions and the hashes of every input, so
// the published report can be traced to private rows without containing
// any dictated text.
import fs from "node:fs/promises";
import crypto from "node:crypto";
import { scoreText } from "./score-text.mjs";
import { scoreAudio } from "./score-audio.mjs";

const hash = (b) => crypto.createHash("sha256").update(b).digest("hex");
const readJSON = async (p) => JSON.parse(await fs.readFile(p, "utf8"));
const readRows = async (p) => {
  const bytes = await fs.readFile(p);
  return { bytes, rows: bytes.toString().trim().split("\n").map(JSON.parse) };
};

const [runsPath, outPath] = process.argv.slice(2);
if (!runsPath || !outPath) throw new Error("usage: report.mjs <runs.json> <summary.json>");
const config = await readJSON(runsPath);
const runs = [];
for (const run of config.runs) {
  const corpusBytes = await fs.readFile(run.corpus);
  const corpus = JSON.parse(corpusBytes);
  if (run.corpusSha256 && hash(corpusBytes) !== run.corpusSha256) throw new Error(`${run.id}: corpus checksum changed`);
  const { bytes, rows } = await readRows(run.results);
  if (run.resultsSha256 && hash(bytes) !== run.resultsSha256) throw new Error(`${run.id}: results checksum changed`);
  const entry = { id: run.id, kind: run.kind, split: run.split, variant: run.variant, corpusSha256: hash(corpusBytes), resultsSha256: hash(bytes) };
  if (run.kind === "text") {
    const { perCase, ...summary } = scoreText(corpus, rows);
    runs.push({ ...entry, ...summary });
  } else if (run.kind === "audio") {
    const manifestBytes = await fs.readFile(run.manifest);
    if (run.manifestSha256 && hash(manifestBytes) !== run.manifestSha256) throw new Error(`${run.id}: manifest checksum changed`);
    const manifest = JSON.parse(manifestBytes);
    if (manifest.textCorpusSha256 !== entry.corpusSha256) throw new Error(`${run.id}: manifest built from another corpus`);
    const { perCase, ...summary } = scoreAudio(manifest, corpus, rows);
    if (typeof run.removeFillers === "boolean" && summary.removeFillers !== run.removeFillers) throw new Error(`${run.id}: filler policy mismatch`);
    runs.push({ ...entry, manifestSha256: hash(manifestBytes), ...summary });
  } else throw new Error(`${run.id}: unknown kind ${run.kind}`);
}
const report = { kind: "arion-live-rewrite-report-v1", generated: new Date().toISOString(), host: config.host, provenance: config.provenance, limitations: config.limitations, runs };
await fs.writeFile(outPath, JSON.stringify(report, null, 2) + "\n", { flag: "wx" });
console.log(`Wrote ${runs.length} runs to ${outPath}`);
