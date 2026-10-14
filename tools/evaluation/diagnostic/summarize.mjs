import fs from "node:fs/promises";
import path from "node:path";
import crypto from "node:crypto";
import { diagnosticMetrics } from "./metrics.mjs";
import { distribution, errorCounts } from "../src/metrics.mjs";
import { digest } from "../src/manifest.mjs";

const hash = (b) => crypto.createHash("sha256").update(b).digest("hex");
const read = async (p) => JSON.parse(await fs.readFile(p, "utf8"));
const flags = Object.fromEntries(
  Array.from({ length: (process.argv.length - 2) / 2 }, (_, i) => [
    process.argv[2 + i * 2].replace(/^--/, ""),
    process.argv[3 + i * 2],
  ]),
);
for (const k of ["runs", "out"])
  if (!flags[k]) throw new Error(`--${k} required`);
const config = await read(flags.runs);
const families = {};
let host;
for (const family of config.rewrite) {
  const run = await read(path.join(family.results, "run.json"));
  const bytes = await fs.readFile(path.join(family.results, "results.jsonl"));
  const corpusBytes = await fs.readFile(family.corpus),
    corpus = JSON.parse(corpusBytes);
  if (
    run.status !== "complete" ||
    hash(bytes) !== run.resultsHash ||
    hash(corpusBytes) !== run.corpusHash
  )
    throw new Error("Incomplete or changed rewrite evidence");
  const rows = bytes.toString().trim().split("\n").map(JSON.parse);
  const pairs = new Set(
    corpus.samples.flatMap((s) =>
      run.config.variants.map((v) => s.id + ":" + v.id),
    ),
  );
  if (
    rows.length !== pairs.size ||
    new Set(rows.map((r) => r.caseId + ":" + r.variantId)).size !==
      pairs.size ||
    rows.some((r) => !pairs.has(r.caseId + ":" + r.variantId))
  )
    throw new Error("Invalid rewrite coverage");
  const aggregate = (items) => ({
    count: items.length,
    constraintPassCount: items.filter((m) => m.constraintPass).length,
    protectedMisses: items.reduce((n, m) => n + m.protectedMisses, 0),
    verbatimViolations: items.filter((m) => m.verbatimViolation).length,
    truncated: items.filter((m) => m.truncated).length,
    referenceEditRatioMean:
      items.reduce((n, m) => n + m.referenceEditRatio, 0) / items.length,
    generationMs: distribution(items.map((m) => m.generationMs)),
    firstTokenMs: distribution(
      items.map((m) => m.firstTokenMs).filter((x) => x !== null),
    ),
    peakRssBytes: distribution(
      items.map((m) => m.peakRssBytes).filter((x) => x !== null),
    ),
  });
  const variants = {};
  for (const v of run.config.variants) {
    const selected = rows.filter((r) => r.variantId === v.id);
    const metrics = selected.map((r) => {
      const s = corpus.samples.find((s) => s.id === r.caseId);
      return {
        ...diagnosticMetrics(s, r.response),
        tags: s.tags,
        mode: s.mode,
        source: s.sourcePipelineId ?? "gold-text",
      };
    });
    variants[v.id] = {
      treatment: v,
      overall: aggregate(metrics),
      cleanMode: aggregate(metrics.filter((m) => m.mode === "clean")),
      byTag: Object.fromEntries(
        [...new Set(metrics.flatMap((m) => m.tags))].map((t) => [
          t,
          aggregate(metrics.filter((m) => m.tags.includes(t))),
        ]),
      ),
      bySourcePipeline: Object.fromEntries(
        [...new Set(metrics.map((m) => m.source))].map((s) => [
          s,
          aggregate(metrics.filter((m) => m.source === s)),
        ]),
      ),
      inferenceCalls: selected.reduce(
        (n, r) =>
          n +
          (r.response.modelInferenceExecuted === false
            ? 0
            : (r.response.stageResponses?.length ?? Number(v.model !== null))),
        0,
      ),
    };
  }
  if (run.host.cpu)
    host = {
      platform: run.host.platform,
      architecture: run.host.architecture,
      cpu: run.host.cpu,
      ramBytes: run.host.ramBytes,
      osRelease: run.host.osRelease,
      lowPowerMode: Number(
        run.host.powerMode.match(/lowpowermode\s+(\d+)/)?.[1] ?? NaN,
      ),
    };
  families[family.id] = {
    corpusHash: run.corpusHash,
    configHash: run.configHash,
    promptHash: run.promptHash,
    workerHash: run.workerHash,
    controllerHash: run.controllerHash ?? null,
    runHash: hash(await fs.readFile(path.join(family.results, "run.json"))),
    resultsHash: run.resultsHash,
    runtimes: run.runtimes,
    cacheState: run.config.cacheState,
    loadState: run.config.loadState,
    maxNewTokensPerStage: run.config.maxNewTokens,
    variants,
  };
}
const audioBytes = await fs.readFile(config.audioCorpus),
  audio = JSON.parse(audioBytes);
const audioPipelines = {};
for (const source of config.asr) {
  const run = await read(path.join(source.results, "run.json"));
  const bytes = await fs.readFile(path.join(source.results, "results.jsonl"));
  const native = source.kind === "native";
  if (
    run.status !== "complete" ||
    hash(bytes) !== (native ? run.resultsSha256 : run.resultsHash) ||
    run.corpusHash !== (native ? digest(audio) : hash(audioBytes))
  )
    throw new Error("Unmatched ASR evidence");
  const rows = bytes.toString().trim().split("\n").map(JSON.parse);
  for (const variant of [...new Set(rows.map((r) => r.variantId))]) {
    const group = rows.filter((r) => r.variantId === variant);
    const cases = audio.samples.filter((s) => s.split === "dev");
    if (
      group.length !== cases.length ||
      new Set(group.map((r) => (native ? r.caseId : r.sampleId))).size !==
        cases.length
    )
      throw new Error("ASR coverage mismatch");
    const languages = {};
    for (const lang of ["zh", "en", "mixed"]) {
      const counts = group
        .filter(
          (r) =>
            cases.find((s) => s.id === (native ? r.caseId : r.sampleId))
              .language === lang,
        )
        .map((r) => {
          const s = cases.find(
            (s) => s.id === (native ? r.caseId : r.sampleId),
          );
          return errorCounts(s.reference, r.response.finalText, s.unit);
        });
      const errors = counts.reduce((n, c) => n + c.errors, 0),
        units = counts.reduce((n, c) => n + c.referenceUnits, 0);
      languages[lang] = {
        samples: counts.length,
        errors,
        referenceUnits: units,
        rate: units ? errors / units : null,
      };
    }
    audioPipelines[variant] = {
      languages,
      explicitStopToResultMs: distribution(
        group.map(
          (r) => r.response.timings.reviewMs - r.response.timings.speechEndMs,
        ),
      ),
      firstPartialMs: distribution(
        group
          .map((r) => r.response.timings.firstPartialMs)
          .filter((x) => x !== null),
      ),
      resultsHash: hash(bytes),
      scope: native
        ? "production replay coordinator without native preview"
        : "offline batch inference after realtime fixture wait; no partials",
    };
  }
}
const summary = {
  version: 1,
  kind: "arion-qwen-development-aggregates",
  scope:
    "Local CPU inference development proxies; no release or Typeless parity claim",
  host,
  provider: "CPUExecutionProvider",
  threads: { intraOp: 4, interOp: 1 },
  annotation: "developer_authored_unreviewed",
  goldTextCases: 24,
  speechCases: 14,
  semanticQuality: "pending independent human review",
  typelessReference: "not executed",
  heldOutTest: "not executed",
  confidenceIntervals: "not computed: one developer author cluster",
  latencyScope:
    "warm engine generation only; two-stage sums exclude IPC/native rendering; thermals/other host tasks not isolated",
  memoryScope:
    "worker lifetime process high-water; not request delta or device footprint",
  families,
  audioPipelines,
};
await fs.mkdir(flags.out, { mode: 0o700 });
await fs.writeFile(
  path.join(flags.out, "summary.json"),
  JSON.stringify(summary, null, 2),
  { flag: "wx" },
);
console.log("Wrote content-free development aggregates for review");
