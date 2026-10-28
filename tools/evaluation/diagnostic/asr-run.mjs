import fs from "node:fs/promises";
import path from "node:path";
import { Driver, validateResponse } from "../src/protocol.mjs";
import { verifyDriverArtifacts } from "../src/runner.mjs";
import { loadCorpus, sha256 as hash } from "../src/manifest.mjs";
import { parseFlags, requireFlags } from "./flags.mjs";

if (process.platform !== "darwin")
  throw new Error(
    "This diagnostic adapter requires macOS; other target OS measurements remain pending",
  );

const flags = parseFlags(process.argv.slice(2));
requireFlags(flags, ["corpus", "driver", "out"]);
const corpusFile = path.resolve(flags.corpus),
  driverFile = path.resolve(flags.driver),
  out = path.resolve(flags.out);
const corpus = await loadCorpus(corpusFile),
  cfg = JSON.parse(await fs.readFile(driverFile, "utf8"));
const artifacts = await verifyDriverArtifacts(cfg, path.dirname(driverFile));
await fs.mkdir(out, { mode: 0o700 });
const driver = new Driver(cfg, path.dirname(driverFile));
const pipeline = {
  asr: "qwen3-asr-06",
  sampleRate: 16000,
  chunkMs: 20,
  endpoint: "explicit-stop-v1",
  pace: "realtime",
  threads: 4,
};
const variant = {
  id: "qwen3-asr-06",
  features: { memory: false, ranker: "source", correction: "none" },
};
const meta = {
  kind: "arion-offline-asr-diagnostic-v1",
  status: "running",
  corpusHash: hash(await fs.readFile(corpusFile)),
  driverHash: hash(await fs.readFile(driverFile)),
  pipeline,
  variant,
  notes:
    "No streaming recognition or native preview. Actual batch inference after explicit-stop realtime fixture wait.",
};
try {
  meta.provenance = await driver.request({
    op: "hello",
    purpose: "benchmark",
    artifacts,
  });
  if (
    JSON.stringify(meta.provenance.artifacts) !== JSON.stringify(artifacts) ||
    meta.provenance.execution !== "live"
  )
    throw new Error("Artifact attestation mismatch");
  await fs.writeFile(path.join(out, "run.json"), JSON.stringify(meta, null, 2));
  const warmups = corpus.samples.filter((s) => s.split === "train").slice(0, 2);
  if (warmups.length !== 2) throw new Error("Separate warmups required");
  for (const sample of [
    ...warmups,
    ...corpus.samples.filter((s) => s.split === "dev"),
  ]) {
    const warmup = sample.split === "train";
    const request = {
      op: "infer",
      sampleId: sample.id,
      seed: 11,
      warmup,
      cacheState: "warm",
      variant,
      pipeline,
      input: {
        audio: {
          ...sample.audio,
          path: path.resolve(path.dirname(corpusFile), sample.audio.path),
        },
        context: "",
        approvedMemory: [],
      },
    };
    const start = performance.now(),
      response = await driver.request(request);
    if (performance.now() - start < sample.audio.durationMs * 0.98)
      throw new Error("Realtime fixture wait missing");
    validateResponse(response, {
      ...request,
      protocol: 1,
      requestId: response.requestId,
    });
    await fs.appendFile(
      path.join(out, warmup ? "warmups.jsonl" : "results.jsonl"),
      JSON.stringify({ sampleId: sample.id, variantId: variant.id, response }) +
        "\n",
    );
    console.log(`Qwen3-ASR: ${sample.id}`);
  }
  const clean = await driver.request({ op: "cleanup" });
  if (!clean.cleanupComplete) throw new Error("Cleanup incomplete");
  meta.status = "complete";
  meta.resultsHash = hash(await fs.readFile(path.join(out, "results.jsonl")));
  meta.warmupsHash = hash(await fs.readFile(path.join(out, "warmups.jsonl")));
} catch (e) {
  meta.status = "failed";
  meta.failure = e.message;
  throw e;
} finally {
  driver.close();
  await fs.writeFile(path.join(out, "run.json"), JSON.stringify(meta, null, 2));
}
