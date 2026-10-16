import fs from "node:fs/promises";
import path from "node:path";
import crypto from "node:crypto";
import os from "node:os";
import { buildPrompt } from "./diagnostic/prompt.mjs";
import { Worker } from "./diagnostic/worker.mjs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { distribution, tokens } from "./src/metrics.mjs";
import { shuffle } from "./src/statistics.mjs";
import { hashFile } from "./src/manifest.mjs";
import { diagnosticMetrics } from "./diagnostic/metrics.mjs";
export { diagnosticMetrics } from "./diagnostic/metrics.mjs";

const root = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "../..",
);
const hash = (bytes) => crypto.createHash("sha256").update(bytes).digest("hex");
const read = async (p) => JSON.parse(await fs.readFile(p, "utf8"));
const flags = Object.fromEntries(
  Array.from({ length: (process.argv.length - 3) / 2 }, (_, i) => [
    process.argv[3 + i * 2].replace(/^--/, ""),
    process.argv[4 + i * 2],
  ]),
);
const promptFile = path.resolve(
  flags.prompts ?? path.join(root, "tools/evaluation/diagnostic/prompts.json"),
);
const promptBytes = await fs.readFile(promptFile);
const prompts = JSON.parse(promptBytes);

export function makePrompt(sample, variant) {
  return buildPrompt(sample, variant, prompts);
}

async function run() {
  for (const key of ["corpus", "models", "python", "out"])
    if (!flags[key]) throw new Error(`--${key} required`);
  const configPath = path.resolve(
    flags.config ?? "benchmarks/configs/qwen-diagnostic.json",
  );
  const config = await read(configPath);
  const corpusBytes = await fs.readFile(flags.corpus);
  const corpus = JSON.parse(corpusBytes);
  if (
    config.kind !== "arion-qwen-diagnostic-v1" ||
    corpus.kind !== config.kind ||
    corpus.annotation !== "developer_authored_unreviewed" ||
    corpus.split !== "dev" ||
    corpus.consent !== true ||
    !corpus.license ||
    !corpus.samples.length
  )
    throw new Error(
      "Requires honestly labeled, authorized development diagnostic corpus",
    );
  if (new Set(corpus.samples.map((s) => s.id)).size !== corpus.samples.length)
    throw new Error("Duplicate case identity");
  const stringList = (x) =>
    Array.isArray(x) && x.every((s) => typeof s === "string" && s.length > 0);
  for (const s of corpus.samples)
    if (
      typeof s.id !== "string" ||
      !s.id ||
      typeof s.sourceText !== "string" ||
      !s.sourceText ||
      s.sourceText.length > 65536 ||
      !["zh", "en", "mixed"].includes(s.language) ||
      !["clean", "verbatim"].includes(s.mode) ||
      typeof s.context !== "string" ||
      !stringList(s.approvedMemory) ||
      !stringList(s.references) ||
      !s.references.length ||
      s.references.some(
        (r) =>
          !tokens(r, { zh: "cer", en: "wer", mixed: "mer" }[s.language]).length,
      ) ||
      !stringList(s.requiredStrings) ||
      !stringList(s.forbiddenStrings) ||
      !stringList(s.protected) ||
      !Array.isArray(s.tags)
    )
      throw new Error("Invalid development case schema");
  if (
    !Array.isArray(config.variants) ||
    config.variants.filter((v) => v.model === null).length !== 1 ||
    new Set(config.variants.map((v) => v.id)).size !== config.variants.length
  )
    throw new Error(
      "Unique treatment identities and one source control required",
    );
  for (const v of config.variants)
    if (
      !v.id ||
      [v.context, v.memory, v.scenario].some((x) => typeof x !== "boolean") ||
      (v.model !== null &&
        (!Object.hasOwn(prompts, v.prompt) ||
          (v.secondPrompt && !Object.hasOwn(prompts, v.secondPrompt))))
    )
      throw new Error("Unsupported prompt or treatment");
  for (const k of ["maxInputTokens", "maxNewTokens", "threads", "warmupCount"])
    if (!Number.isInteger(config[k]) || config[k] <= 0)
      throw new Error("Positive explicit inference budgets required");

  if (
    config.maxInputTokens > 2048 ||
    config.maxNewTokens > 256 ||
    config.threads > 8
  )
    throw new Error("Diagnostic budgets exceed implemented bounds");
  if (
    config.cacheState !== "warm" ||
    config.loadState !== "solo" ||
    config.provider !== "CPUExecutionProvider"
  )
    throw new Error("This diagnostic adapter only implements warm solo CPU");
  // Input validation is portable; actual inference still requires the measured OS.
  if (process.platform !== "darwin")
    throw new Error(
      "This diagnostic version is measured on macOS only; native execution on this OS remains pending",
    );
  const out = path.resolve(flags.out);
  await fs.mkdir(out, { mode: 0o700 });
  const artifacts = await read(path.join(flags.models, "artifacts.json"));
  for (const a of artifacts)
    if ((await hashFile(a.path)) !== a.sha256)
      throw new Error("Artifact digest mismatch");
  const manifest = {
    kind: config.kind,
    status: "running",
    evidence: "real_native_cpu_engine_via_python_without_native_preview",
    corpusHash: hash(corpusBytes),
    configHash: hash(await fs.readFile(configPath)),
    promptHash: hash(promptBytes),
    workerHash: hash(
      await fs.readFile(
        path.join(root, "tools/evaluation/diagnostic/qwen_worker.py"),
      ),
    ),
    artifacts,
    config,
    annotation: corpus.annotation,
    limitations: corpus.limitations,
    host: {
      platform: process.platform,
      architecture: process.arch,
      cpu: os.cpus()[0].model,
      ramBytes: os.totalmem(),
      osRelease: os.release(),
      powerMode: execFileSync("pmset", ["-g"], { encoding: "utf8" }).trim(),
    },
    controllerHash: hash(await fs.readFile(fileURLToPath(import.meta.url))),
    gitRevision: execFileSync("git", ["rev-parse", "HEAD"], {
      encoding: "utf8",
    }).trim(),
    gitDirty:
      execFileSync("git", ["status", "--porcelain"], {
        encoding: "utf8",
      }).trim().length > 0,
    runtimes: {},
  };
  await fs.writeFile(path.join(out, "prompts.json"), promptBytes, {
    flag: "wx",
  });
  const workerFile = path.join(out, "worker.py");
  await fs.copyFile(
    path.join(root, "tools/evaluation/diagnostic/qwen_worker.py"),
    workerFile,
  );
  await fs.writeFile(
    path.join(out, "run.json"),
    JSON.stringify(manifest, null, 2),
  );
  const trace = await fs.open(path.join(out, "results.jsonl"), "wx");
  const warmups = await fs.open(path.join(out, "warmups.jsonl"), "wx");
  let worker;
  try {
    const source = config.variants.find((v) => v.model === null);
    for (const sample of corpus.samples) {
      const response = {
        text: sample.sourceText,
        generationMs: 0,
        firstTokenMs: null,
        truncated: false,
        peakRssBytes: null,
      };
      await trace.write(
        JSON.stringify({
          caseId: sample.id,
          variantId: source.id,
          response,
          metrics: diagnosticMetrics(sample, response),
        }) + "\n",
      );
    }
    // Model residency is blocked by checkpoint; treatments/cases are interleaved within each checkpoint.
    for (const modelId of [
      ...new Set(config.variants.map((v) => v.model).filter(Boolean)),
    ]) {
      const runtimeDir = path.join(out, modelId);
      await fs.mkdir(runtimeDir);
      const original = path.resolve(flags.models, modelId);
      for (const file of await fs.readdir(original))
        if (file !== "genai_config.json")
          await fs.symlink(
            path.join(original, file),
            path.join(runtimeDir, file),
          );
      const runtimeConfig = await read(
        path.join(original, "genai_config.json"),
      );
      runtimeConfig.model.decoder.session_options.intra_op_num_threads =
        config.threads;
      runtimeConfig.model.decoder.session_options.inter_op_num_threads = 1;
      const bytes = JSON.stringify(runtimeConfig, null, 2);
      await fs.writeFile(path.join(runtimeDir, "genai_config.json"), bytes);
      const modelArtifacts = artifacts.filter(
        (a) => path.dirname(a.path) === original,
      );
      modelArtifacts.push({
        path: path.join(runtimeDir, "genai_config.json"),
        sha256: hash(bytes),
      });
      const diagnostic = await fs.open(
        path.join(out, `${modelId}-stderr.log`),
        "wx",
      );
      worker = new Worker(
        path.resolve(flags.python),
        diagnostic.fd,
        workerFile,
      );
      const load = await worker.request({
        op: "load",
        modelPath: runtimeDir,
        artifacts: modelArtifacts,
      });
      manifest.runtimes[modelId] = { ...load, derivedConfigHash: hash(bytes) };
      await fs.writeFile(
        path.join(out, "run.json"),
        JSON.stringify(manifest, null, 2),
      );
      const variants = config.variants.filter((v) => v.model === modelId);
      for (let i = 0; i < config.warmupCount; i++) {
        const sample = {
          sourceText: i
            ? "嗯测试已经完成，可以继续。"
            : "Um the local editor is ready.",
          mode: "clean",
          scenario: "chat",
          context: "",
          approvedMemory: [],
        };
        const response = await worker.request({
          op: "generate",
          prompt: makePrompt(sample, variants[0]),
          maxInputTokens: config.maxInputTokens,
          maxNewTokens: config.maxNewTokens,
          enableThinking: variants[0].thinking === true,
        });
        await warmups.write(JSON.stringify({ modelId, response }) + "\n");
      }
      const jobs = shuffle(
        corpus.samples.flatMap((sample) =>
          variants.map((variant) => ({ sample, variant })),
        ),
        config.orderSeed,
      );
      let count = 0;
      for (const { sample, variant } of jobs) {
        const prompt = makePrompt(sample, variant);
        let response;
        if (variant.verbatimBypass && sample.mode === "verbatim") {
          response = {
            text: sample.sourceText,
            rawText: sample.sourceText,
            events: [],
            inputTokens: 0,
            outputTokens: 0,
            generationMs: 0,
            firstTokenMs: null,
            truncated: false,
            peakRssBytes: null,
            modelInferenceExecuted: false,
            policy: "explicit_verbatim_bypass",
          };
        } else {
          response = await worker.request({
            op: "generate",
            prompt,
            maxInputTokens: config.maxInputTokens,
            maxNewTokens: config.maxNewTokens,
            enableThinking: variant.thinking === true,
          });
          response.modelInferenceExecuted = true;
          if (variant.secondPrompt) {
            const first = response;
            if (first.truncated || !first.text) {
              response = {
                ...first,
                text: "",
                truncated: true,
                stageDecision: "no_first_candidate",
                stageResponses: [first],
              };
            } else {
              const secondPrompt = makePrompt(
                { ...sample, sourceText: first.text },
                { ...variant, prompt: variant.secondPrompt },
              );
              const second = await worker.request({
                op: "generate",
                prompt: secondPrompt,
                maxInputTokens: config.maxInputTokens,
                maxNewTokens: config.maxNewTokens,
                enableThinking: false,
              });
              response = {
                ...second,
                modelInferenceExecuted: true,
                generationMs: first.generationMs + second.generationMs,
                firstTokenMs:
                  second.firstTokenMs === null
                    ? null
                    : first.generationMs + second.firstTokenMs,
                events: second.events.map((e) => ({
                  ...e,
                  ms: e.ms + first.generationMs,
                })),
                inputTokens: first.inputTokens + second.inputTokens,
                outputTokens: first.outputTokens + second.outputTokens,
                peakRssBytes: Math.max(first.peakRssBytes, second.peakRssBytes),
                stageResponses: [first, second],
                secondInputHash: hash(secondPrompt),
              };
            }
          }
        }
        await trace.write(
          JSON.stringify({
            caseId: sample.id,
            variantId: variant.id,
            inputHash: hash(prompt),
            response,
            metrics: diagnosticMetrics(sample, response),
          }) + "\n",
        );
        console.log(`${modelId}: ${++count}/${jobs.length}`);
      }
      await worker.close();
      worker = null;
      await diagnostic.close();
    }
    await trace.close();
    await warmups.close();
    manifest.status = "complete";
    manifest.resultsHash = hash(
      await fs.readFile(path.join(out, "results.jsonl")),
    );
    manifest.warmupsHash = hash(
      await fs.readFile(path.join(out, "warmups.jsonl")),
    );
    await fs.writeFile(
      path.join(out, "run.json"),
      JSON.stringify(manifest, null, 2),
    );
  } catch (error) {
    worker?.child.kill();
    await trace.close();
    await warmups.close();
    manifest.status = "failed";
    manifest.failure = error.message;
    await fs.writeFile(
      path.join(out, "run.json"),
      JSON.stringify(manifest, null, 2),
    );
    throw error;
  }
}

async function report() {
  if (!flags.results || !flags.corpus)
    throw new Error("--results and --corpus required");
  const out = path.resolve(flags.results),
    manifest = await read(path.join(out, "run.json"));
  const bytes = await fs.readFile(path.join(out, "results.jsonl")),
    corpusBytes = await fs.readFile(flags.corpus);
  if (
    manifest.status !== "complete" ||
    hash(bytes) !== manifest.resultsHash ||
    hash(corpusBytes) !== manifest.corpusHash
  )
    throw new Error("Incomplete or changed run");
  const corpus = JSON.parse(corpusBytes),
    rows = bytes.toString().trim().split("\n").map(JSON.parse);
  const expected = corpus.samples.length * manifest.config.variants.length;
  const expectedPairs = new Set(
    corpus.samples.flatMap((s) =>
      manifest.config.variants.map((v) => s.id + ":" + v.id),
    ),
  );
  if (
    rows.length !== expected ||
    new Set(rows.map((r) => r.caseId + ":" + r.variantId)).size !== expected ||
    rows.some((r) => !expectedPairs.has(r.caseId + ":" + r.variantId))
  )
    throw new Error("Invalid coverage");
  const summaries = {};
  for (const v of manifest.config.variants) {
    const group = rows.filter((r) => r.variantId === v.id);
    const metrics = group.map((r) => {
      const s = corpus.samples.find((s) => s.id === r.caseId);
      if (!s) throw new Error("Unknown case");
      return {
        ...diagnosticMetrics(s, r.response),
        tags: s.tags,
        language: s.language,
        sourcePipelineId: s.sourcePipelineId ?? "gold-text",
      };
    });
    const aggregate = (m) => ({
      count: m.length,
      constraintPassCount: m.filter((x) => x.constraintPass).length,
      referenceEditRatioMean:
        m.reduce((n, x) => n + x.referenceEditRatio, 0) / m.length,
      requiredMisses: m.reduce((n, x) => n + x.requiredMisses, 0),
      forbiddenHits: m.reduce((n, x) => n + x.forbiddenHits, 0),
      protectedMisses: m.reduce((n, x) => n + x.protectedMisses, 0),
      verbatimViolations: m.filter((x) => x.verbatimViolation).length,
      truncated: m.filter((x) => x.truncated).length,
      generationMs: distribution(m.map((x) => x.generationMs)),
      firstTokenMs: distribution(
        m.map((x) => x.firstTokenMs).filter((x) => x !== null),
      ),
      peakRssBytes: distribution(
        m.map((x) => x.peakRssBytes).filter((x) => x !== null),
      ),
    });
    summaries[v.id] = {
      overall: aggregate(metrics),
      byLanguage: Object.fromEntries(
        [...new Set(metrics.map((m) => m.language))].map((language) => [
          language,
          aggregate(metrics.filter((m) => m.language === language)),
        ]),
      ),
      bySourcePipeline: Object.fromEntries(
        [...new Set(metrics.map((m) => m.sourcePipelineId))].map((source) => [
          source,
          aggregate(metrics.filter((m) => m.sourcePipelineId === source)),
        ]),
      ),
      byTag: Object.fromEntries(
        [...new Set(corpus.samples.flatMap((s) => s.tags))].map((tag) => [
          tag,
          aggregate(metrics.filter((m) => m.tags.includes(tag))),
        ]),
      ),
    };
  }
  const summary = {
    kind: manifest.kind,
    status: "complete",
    scope: manifest.evidence,
    observations: rows.length,
    corpusHash: manifest.corpusHash,
    configHash: manifest.configHash,
    promptHash: manifest.promptHash,
    annotation: manifest.annotation,
    limitations: manifest.limitations,
    semanticQuality: "pending independent blind review",
    confidenceIntervals: "not computed: one developer author cluster",
    latencyScope:
      "warm engine generation; excludes audio, endpoint, IPC, native preview, validation and insertion",
    rssScope:
      "process_high_water: cumulative across requests, separate worker per checkpoint",
    summaries,
  };
  await fs.writeFile(
    path.join(out, "summary.json"),
    JSON.stringify(summary, null, 2),
    { flag: "wx" },
  );
  console.log(JSON.stringify(summary, null, 2));
}

if (process.argv[2] === "run") await run();
else if (process.argv[2] === "report") await report();
else if (process.argv[1] === fileURLToPath(import.meta.url))
  throw new Error("Use run or report");
