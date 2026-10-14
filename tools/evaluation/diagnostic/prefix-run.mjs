import fs from "node:fs/promises";
import path from "node:path";
import crypto from "node:crypto";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { Driver, validateResponse } from "../src/protocol.mjs";
import { verifyDriverArtifacts, inferenceRequest } from "../src/runner.mjs";
import { loadCorpus, digest, hashFile } from "../src/manifest.mjs";
import { distribution, errorCounts } from "../src/metrics.mjs";
import { shuffle } from "../src/statistics.mjs";
import { Worker } from "./worker.mjs";
import { buildPrompt, systemPrefix } from "./prompt.mjs";
import { diagnosticMetrics } from "./metrics.mjs";

if (process.platform !== "darwin")
  throw new Error(
    "This diagnostic adapter requires macOS; other target OS measurements remain pending",
  );

const root = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../../..",
);
const hash = (b) => crypto.createHash("sha256").update(b).digest("hex");
const read = async (p) => JSON.parse(await fs.readFile(p, "utf8"));
const flags = Object.fromEntries(
  Array.from({ length: (process.argv.length - 2) / 2 }, (_, i) => [
    process.argv[2 + i * 2].replace(/^--/, ""),
    process.argv[3 + i * 2],
  ]),
);
for (const k of [
  "config",
  "corpus",
  "audio-corpus",
  "driver",
  "models",
  "python",
  "prompts",
  "out",
])
  if (!flags[k]) throw new Error(`--${k} required`);
const config = await read(flags.config),
  corpus = await read(flags.corpus),
  audio = await loadCorpus(path.resolve(flags["audio-corpus"])),
  driverConfig = await read(flags.driver),
  prompts = await read(flags.prompts);
if (
  config.sharedMaxLength !== config.maxInputTokens + config.maxNewTokens ||
  config.maxInputTokens !== 2048 ||
  config.maxNewTokens !== 128 ||
  config.threads !== 4 ||
  corpus.annotation !== "developer_authored_unreviewed"
)
  throw new Error("Unsupported prefix diagnostic settings");
const cases = config.selectedCaseIds.map((id) => ({
  sample: corpus.samples.find((s) => s.id === id),
  audio: audio.samples.find((s) => s.id === id),
}));
if (cases.length !== 6 || cases.some((c) => !c.sample || !c.audio))
  throw new Error("Six declared matched diagnostic cases required");
const out = path.resolve(flags.out);
await fs.mkdir(out, { mode: 0o700 });
const artifacts = await read(path.join(flags.models, "artifacts.json"));
for (const a of artifacts)
  if ((await hashFile(a.path)) !== a.sha256)
    throw new Error("LLM artifact digest mismatch");
const original = path.resolve(flags.models, config.model),
  modelPath = path.join(out, "model");
await fs.mkdir(modelPath);
for (const f of await fs.readdir(original))
  if (f !== "genai_config.json")
    await fs.symlink(path.join(original, f), path.join(modelPath, f));
const runtimeConfig = await read(path.join(original, "genai_config.json"));
runtimeConfig.model.decoder.session_options.intra_op_num_threads = 4;
runtimeConfig.model.decoder.session_options.inter_op_num_threads = 1;
const runtimeBytes = JSON.stringify(runtimeConfig, null, 2);
await fs.writeFile(path.join(modelPath, "genai_config.json"), runtimeBytes);
const workerFile = path.join(out, "worker.py");
await fs.copyFile(
  path.join(root, "tools/evaluation/diagnostic/qwen_worker.py"),
  workerFile,
);
const log = await fs.open(path.join(out, "stderr.log"), "wx");
const llm = new Worker(path.resolve(flags.python), log.fd, workerFile),
  asr = new Driver(driverConfig, path.dirname(path.resolve(flags.driver)));
const variant = {
  id: "prefix-intent",
  prompt: config.prompt,
  context: true,
  memory: true,
  scenario: true,
};
const asrConfig = {
  pipeline: {
    asr: config.asr,
    sampleRate: 16000,
    chunkMs: 20,
    endpoint: "sherpa-rules-v1",
    pace: "realtime",
    threads: 4,
    endpointSilenceMs: 800,
  },
};
const block = {
  seed: 11,
  cacheState: "warm",
  variant: {
    id: config.asr,
    features: { memory: false, ranker: "source", correction: "none" },
  },
};
const options = {
  maxInputTokens: config.maxInputTokens,
  maxNewTokens: config.maxNewTokens,
  sharedMaxLength: config.sharedMaxLength,
  enableThinking: false,
};
const meta = {
  kind: "arion-prefix-prefill-development-v1",
  status: "running",
  config,
  configHash: hash(await fs.readFile(flags.config)),
  corpusHash: hash(await fs.readFile(flags.corpus)),
  audioCorpusHash: digest(audio),
  promptHash: hash(await fs.readFile(flags.prompts)),
  workerHash: await hashFile(workerFile),
  controllerHash: hash(await fs.readFile(fileURLToPath(import.meta.url))),
  artifacts,
  limitations: [
    "Six synthetic cases, unreviewed developer annotations",
    "No native rendered preview, insertion, cancellation or queue/drop counters",
    "ASR-result to candidate is observed; stop estimate adds separately reported acoustic finalization and excludes its response serialization",
  ],
};
const rows = [];
const inhibitor = spawn("caffeinate", ["-i", "-w", String(process.pid)], {
  stdio: "ignore",
});
await new Promise((resolve, reject) => {
  inhibitor.once("spawn", resolve);
  inhibitor.once("error", reject);
});
meta.idleSleepInhibited = true;
try {
  const hashes = await verifyDriverArtifacts(
    driverConfig,
    path.dirname(path.resolve(flags.driver)),
  );
  meta.asrProvenance = await asr.request({
    op: "hello",
    purpose: "benchmark",
    artifacts: hashes,
  });
  if (meta.asrProvenance.execution !== "live")
    throw new Error("Actual ASR required");
  meta.llmProvenance = await llm.request({
    op: "load",
    modelPath,
    artifacts: artifacts.filter((a) => path.dirname(a.path) === original),
  });
  for (const s of audio.samples
    .filter((s) => s.split === "train")
    .slice(0, 2)) {
    const request = inferenceRequest(
      s,
      block,
      asrConfig,
      path.resolve(flags["audio-corpus"]),
      true,
    );
    const response = await asr.request(request);
    validateResponse(response, {
      ...request,
      requestId: response.requestId,
      protocol: 1,
    });
    const sample = {
      sourceText: response.finalText,
      mode: "clean",
      scenario: "chat",
      context: "",
      approvedMemory: [],
    };
    const generation = await llm.request({
      op: "generate",
      prompt: buildPrompt(sample, variant, prompts),
      ...options,
    });
    await fs.appendFile(
      path.join(out, "warmups.jsonl"),
      JSON.stringify({ asr: response, llm: generation }) + "\n",
    );
  }
  for (const { sample, audio: fixture, mode } of shuffle(
    cases.flatMap((c) =>
      ["sequential", "prefill"].map((mode) => ({ ...c, mode })),
    ),
    config.orderSeed,
  )) {
    const start = performance.now();
    const request = inferenceRequest(
      fixture,
      block,
      asrConfig,
      path.resolve(flags["audio-corpus"]),
    );
    let asrReturnedMs;
    const audioTask = asr.request(request).then((response) => {
      asrReturnedMs = performance.now() - start;
      return response;
    });
    const prefixTask =
      mode === "prefill"
        ? llm
            .request({
              op: "prepare_prefix",
              prompt: systemPrefix(variant, prompts),
              ...options,
            })
            .then((response) => ({
              ...response,
              readyMs: performance.now() - start,
            }))
        : Promise.resolve(null);
    const [recognition, prefix] = await Promise.all([audioTask, prefixTask]);
    validateResponse(recognition, {
      ...request,
      requestId: recognition.requestId,
      protocol: 1,
    });
    if (asrReturnedMs < fixture.audio.durationMs * 0.98)
      throw new Error("Realtime replay missing");
    const input = { ...sample, sourceText: recognition.finalText };
    const prompt = buildPrompt(input, variant, prompts);
    const generation = await llm.request({
      op: mode === "prefill" ? "finish_prefix" : "generate",
      prompt,
      ...options,
    });
    const candidateMs = performance.now() - start;
    const row = {
      caseId: sample.id,
      mode,
      sourceHash: hash(recognition.finalText),
      inputHash: hash(prompt),
      recognition,
      prefix,
      generation,
      metrics: diagnosticMetrics(input, generation),
      timeline: {
        asrReturnedMs,
        candidateMs,
        asrResultToCandidateMs: candidateMs - asrReturnedMs,
        serialExplicitStopEstimateMs:
          recognition.timings.reviewMs -
          recognition.timings.speechEndMs +
          candidateMs -
          asrReturnedMs,
        prefixReadyBeforeASRResult: prefix
          ? prefix.readyMs <= asrReturnedMs
          : null,
      },
    };
    rows.push(row);
    await fs.appendFile(
      path.join(out, "results.jsonl"),
      JSON.stringify(row) + "\n",
    );
    console.log(`Prefix diagnostic: ${rows.length}/12`);
    if (row.timeline.asrResultToCandidateMs - generation.generationMs > 1000)
      throw new Error(
        "Uncontrolled host scheduling or clock interruption; latency comparison invalid",
      );
  }
  // The ASR benchmark protocol has no cleanup operation. Its only owned
  // resource is this subprocess; verify OS termination instead of inventing an ACK.
  const asrExited = new Promise((resolve, reject) => {
    const timer = setTimeout(
      () => reject(new Error("ASR subprocess termination unconfirmed")),
      10000,
    );
    asr.process.once("exit", (code, signal) => {
      clearTimeout(timer);
      resolve({ code, signal });
    });
  });
  asr.close();
  meta.asrTermination = await asrExited;
  await llm.close();
  asr.close();
  await log.close();
  const summaries = {};
  for (const mode of ["sequential", "prefill"]) {
    const group = rows.filter((r) => r.mode === mode);
    summaries[mode] = {
      count: group.length,
      constraintPassCount: group.filter((r) => r.metrics.constraintPass).length,
      asrResultToCandidateMs: distribution(
        group.map((r) => r.timeline.asrResultToCandidateMs),
      ),
      serialExplicitStopEstimateMs: distribution(
        group.map((r) => r.timeline.serialExplicitStopEstimateMs),
      ),
      firstPartialMs: distribution(
        group
          .map((r) => r.recognition.timings.firstPartialMs)
          .filter((x) => x !== null),
      ),
      prefixReadyBeforeASRResult: group.filter(
        (r) => r.timeline.prefixReadyBeforeASRResult,
      ).length,
      prefixPrefillMs: distribution(
        group.filter((r) => r.prefix).map((r) => r.prefix.prefixPrefillMs),
      ),
      tokenizerParity: group.every((r) => r.generation.tokenizerParity),
      rawAsrErrors: group.map((r) => {
        const s = cases.find((c) => c.sample.id === r.caseId).audio;
        const e = errorCounts(s.reference, r.recognition.finalText, s.unit);
        return { language: s.language, ...e };
      }),
    };
  }
  const pairs = cases.map((c) => {
    const a = rows.find(
        (r) => r.caseId === c.sample.id && r.mode === "sequential",
      ),
      b = rows.find((r) => r.caseId === c.sample.id && r.mode === "prefill");
    return {
      sameASR: a.sourceHash === b.sourceHash,
      sameInput: a.inputHash === b.inputHash,
      sameOutput: a.generation.text === b.generation.text,
    };
  });
  meta.status = "complete";
  meta.resultsHash = hash(await fs.readFile(path.join(out, "results.jsonl")));
  await fs.writeFile(
    path.join(out, "summary.json"),
    JSON.stringify(
      {
        kind: meta.kind,
        limitations: meta.limitations,
        configHash: meta.configHash,
        corpusHash: meta.corpusHash,
        promptHash: meta.promptHash,
        workerHash: meta.workerHash,
        resultsHash: meta.resultsHash,
        summaries,
        pairedConsistency: {
          sameASR: pairs.filter((p) => p.sameASR).length,
          sameInput: pairs.filter((p) => p.sameInput).length,
          sameOutput: pairs.filter((p) => p.sameOutput).length,
          pairs: pairs.length,
        },
      },
      null,
      2,
    ),
  );
} catch (error) {
  meta.status = "failed";
  meta.failure = error.message;
  asr.close();
  llm.child.kill();
  throw error;
} finally {
  inhibitor.kill();
  await fs.writeFile(path.join(out, "run.json"), JSON.stringify(meta, null, 2));
}
