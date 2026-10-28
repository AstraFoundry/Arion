import fs from "node:fs/promises";
import path from "node:path";
import { shuffle } from "../src/statistics.mjs";
import { sha256 as hash } from "../src/manifest.mjs";
import { parseFlags, requireFlags } from "./flags.mjs";

const flags = parseFlags(process.argv.slice(2));
requireFlags(flags, ["results", "corpus", "out"]);
const out = path.resolve(flags.out),
  dir = path.resolve(flags.results);
const run = JSON.parse(await fs.readFile(path.join(dir, "run.json"), "utf8"));
const bytes = await fs.readFile(path.join(dir, "results.jsonl")),
  corpusBytes = await fs.readFile(flags.corpus);
if (
  run.status !== "complete" ||
  hash(bytes) !== run.resultsHash ||
  hash(corpusBytes) !== run.corpusHash
)
  throw new Error("Changed or incomplete diagnostic evidence");
const corpus = JSON.parse(corpusBytes),
  rows = bytes.toString().trim().split("\n").map(JSON.parse);
const tasks = [],
  mapping = [];
for (const [i, r] of shuffle(rows, 1729).entries()) {
  const s = corpus.samples.find((s) => s.id === r.caseId);
  if (!s) throw new Error("Unmatched case");
  const id = `R-${String(i + 1).padStart(4, "0")}`;
  tasks.push({
    id,
    originalDictation: s.originalText ?? s.sourceText,
    sourceTranscript: s.sourceText,
    mode: s.mode,
    scenario: s.scenario,
    context: s.context,
    approvedTerminology: s.approvedMemory,
    candidate: r.response.text,
    deliveryFailed: r.response.truncated,
  });
  mapping.push({ id, caseId: r.caseId, variantId: r.variantId });
}
await fs.mkdir(out, { mode: 0o700 });
await fs.writeFile(
  path.join(out, "tasks.json"),
  JSON.stringify(
    {
      kind: "unreviewed-development-blind-tasks",
      requirements:
        "Independent human raters; assess meaning, style, fluency and unsupported factual changes separately. Do not infer intent from candidate. Exact-string proxies are not truth. These tasks do not satisfy the strict release review protocol.",
      tasks,
    },
    null,
    2,
  ),
  { flag: "wx", mode: 0o600 },
);
await fs.writeFile(
  path.join(out, "mapping-private.json"),
  JSON.stringify(
    { resultsHash: run.resultsHash, corpusHash: run.corpusHash, mapping },
    null,
    2,
  ),
  { flag: "wx", mode: 0o600 },
);
console.log(
  `Exported ${tasks.length} unrated blinded development tasks; keep mapping private`,
);
