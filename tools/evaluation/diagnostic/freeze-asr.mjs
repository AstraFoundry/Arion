import fs from "node:fs/promises";
import path from "node:path";
import { digest, readJson as read, sha256 as hash } from "../src/manifest.mjs";
import { parseFlags, requireFlags } from "./flags.mjs";

const flags = parseFlags(process.argv.slice(2));
requireFlags(flags, ["corpus", "audio-corpus", "native", "qwen-asr", "out"]);
const base = await read(flags.corpus),
  audioBytes = await fs.readFile(flags["audio-corpus"]),
  audio = JSON.parse(audioBytes);
if (
  base.kind !== "arion-qwen-diagnostic-v1" ||
  base.annotation !== "developer_authored_unreviewed"
)
  throw new Error("Requires authorized development labels");
const records = [],
  sourceRuns = [];
for (const [dir, kind] of [
  [flags.native, "native"],
  [flags["qwen-asr"], "qwen"],
]) {
  const meta = await read(path.join(dir, "run.json")),
    bytes = await fs.readFile(path.join(dir, "results.jsonl"));
  if (
    meta.status !== "complete" ||
    hash(bytes) !==
      (kind === "native" ? meta.resultsSha256 : meta.resultsHash) ||
    meta.corpusHash !== (kind === "native" ? digest(audio) : hash(audioBytes))
  )
    throw new Error("Incomplete, changed, or unmatched ASR evidence");
  sourceRuns.push({
    kind,
    runHash: hash(await fs.readFile(path.join(dir, "run.json"))),
    resultsHash: hash(bytes),
  });
  for (const line of bytes.toString().trim().split("\n")) {
    const row = JSON.parse(line),
      caseId = kind === "native" ? row.caseId : row.sampleId;
    const sample = base.samples.find((s) => s.id === caseId);
    if (!sample) throw new Error("Unmatched ASR case");
    if (row.response.execution !== "live")
      throw new Error("Inference attestation missing");
    records.push({
      ...sample,
      id: row.variantId + "__" + caseId,
      originalText: sample.sourceText,
      sourceText: row.response.finalText,
      inputKind: "frozen_asr",
      sourcePipelineId: row.variantId,
      asrStopToResultMs:
        row.response.timings.reviewMs - row.response.timings.speechEndMs,
      tags: [...sample.tags, "frozen-asr", "asr:" + row.variantId],
    });
  }
}
const ids = new Set(records.map((s) => s.id));
if (
  ids.size !== records.length ||
  records.length !== audio.samples.filter((s) => s.split === "dev").length * 4
)
  throw new Error("Expected four complete source pipelines");
await fs.writeFile(
  flags.out,
  JSON.stringify(
    {
      ...base,
      sourceRuns,
      samples: records,
      limitations: [
        ...base.limitations,
        "Frozen ASR inputs; no native combined pipeline execution or preview",
        "Serial latency sums are estimates from separately executed stages",
      ],
    },
    null,
    2,
  ),
  { flag: "wx", mode: 0o600 },
);
console.log(`Froze ${records.length} real ASR source cases`);
