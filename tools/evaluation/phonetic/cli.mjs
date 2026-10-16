import fs from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { fileURLToPath } from "node:url";
import { execFileSync } from "node:child_process";
import { parseArgs } from "node:util";
import { Worker } from "../diagnostic/worker.mjs";
import { hashFile as originalHashFile, loadCorpus as originalLoadCorpus } from "../src/manifest.mjs";
import { openArchive } from "../archive/manifest.mjs";
import { quantile, distribution } from "../src/metrics.mjs";
import { shuffle, pairedBootstrap } from "../src/statistics.mjs";
import { VARIANTS, promptFor, scoreRow, summarizeRows, requireCoverage } from "./design.mjs";
import { validatePreflight, outputContractFailure } from "./preflight.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const own = path.dirname(fileURLToPath(import.meta.url));
const { positionals: [command], values: flags } = parseArgs({
  allowPositionals: true,
  options: Object.fromEntries(["config", "corpus", "python", "models", "phone-models", "runtime-artifacts", "llm-artifacts", "evidence", "preflight", "out", "results", "archive"].map(key => [key, { type: "string" }])),
});
if (flags.archive && command !== "report") throw new Error("--archive is only supported for report replay");
const archive = flags.archive ? await openArchive(path.resolve(flags.archive), root) : null;
const readBytes = archive ? archive.read : name => fs.readFile(name);
const hashFile = archive ? archive.hash : originalHashFile;
const loadCorpus = async name => {
  if (!archive) return originalLoadCorpus(name);
  await archive.read(name);
  return originalLoadCorpus(archive.resolve(name));
};
const read = async name => JSON.parse(await readBytes(name));
const json = value => JSON.stringify(value, null, 2) + "\n";
const lines = async name => (await readBytes(name)).toString("utf8").trim().split("\n").filter(Boolean).map(line => JSON.parse(line));
const absolute = key => path.resolve(required(key));
function required(key) {
  if (!flags[key]) throw new Error(`--${key} required`);
  return flags[key];
}
async function artifactsVerified(artifacts) {
  if (!artifacts.length) throw new Error("Nonempty artifact manifest required");
  for (const artifact of artifacts)
    if (await hashFile(artifact.path) !== artifact.sha256) throw new Error(`Artifact mismatch: ${artifact.path}`);
  return artifacts;
}
function validateConfig(config) {
  if (config.kind !== "arion-phonetic-ablation-v1" || config.split !== "test" ||
      !config.models.length || new Set(config.models).size !== config.models.length ||
      !config.variants.length || new Set(config.variants).size !== config.variants.length ||
      config.variants.some(variant => !VARIANTS.includes(variant)) ||
      !Number.isInteger(config.threads) || config.threads < 1 ||
      !(config.scoreQuantile > 0 && config.scoreQuantile < 1)) throw new Error("Invalid phonetic experiment");
  return config;
}
async function metadata(stage, config, corpusFile) {
  return {
    kind: "arion-phonetic-ablation-v1", stage, status: "running", config,
    configHash: await hashFile(absolute("config")), corpusFile, corpusHash: await hashFile(corpusFile),
    host: { os: process.platform, architecture: process.arch, cpu: os.cpus()[0].model,
      ramBytes: os.totalmem(), osRelease: os.release(),
      powerMode: execFileSync("pmset", ["-g"], { encoding: "utf8" }).trim() },
    gitRevision: execFileSync("git", ["rev-parse", "HEAD"], { cwd: root, encoding: "utf8" }).trim(),
    gitDirty: Boolean(execFileSync("git", ["status", "--porcelain"], { cwd: root, encoding: "utf8" }).trim()),
    timer: "monotonic time.perf_counter / performance.now", codeArtifacts: await Promise.all([
      fileURLToPath(import.meta.url), path.join(own, "design.mjs"), path.join(own, "preflight.mjs"), path.join(own, "acoustic_worker.py"),
      path.join(own, "../diagnostic/qwen_worker.py"), path.join(own, "../diagnostic/worker.mjs"),
    ].map(async name => ({ path: name, sha256: await hashFile(name) }))),
  };
}
async function extract() {
  const config = validateConfig(await read(absolute("config")));
  const corpusFile = absolute("corpus"), corpus = await loadCorpus(corpusFile);
  const out = absolute("out");
  await fs.mkdir(out, { mode: 0o700 });
  const meta = await metadata("acoustic-extraction", config, corpusFile);
  let worker, diagnostics;
  try {
    const catalog = await read(path.join(root, "models/catalog.json"));
    const model = catalog.artifacts.find(item => item.id === config.asrModel);
    if (!model || model.kind !== "online-transducer") throw new Error("Pinned transducer required");
    const asrFiles = Object.fromEntries(Object.entries(model.files).map(([role, item]) => [role, path.join(absolute("models"), model.id, path.basename(item.member))]));
    meta.artifacts = await artifactsVerified([
      ...Object.entries(model.files).map(([role, item]) => ({ path: asrFiles[role], sha256: item.sha256, license: model.license })),
      ...await read(absolute("runtime-artifacts")),
    ]);
    await fs.writeFile(path.join(out, "run.json"), json(meta));
    diagnostics = await fs.open(path.join(out, "stderr.log"), "wx");
    worker = new Worker(absolute("python"), diagnostics.fd, path.join(own, "acoustic_worker.py"));
    meta.runtime = await worker.request({ op: "load", config, artifacts: meta.artifacts, asrFiles, phoneModels: absolute("phone-models") });
    const warmups = corpus.samples.filter(sample => sample.split === "train").slice(0, config.warmupCount);
    if (warmups.length !== config.warmupCount) throw new Error("Separate acoustic warmups required");
    const measured = corpus.samples.filter(sample => ["dev", "test"].includes(sample.split));
    meta.sampleIds = measured.map(sample => sample.id);
    for (const [index, sample] of [...warmups, ...measured].entries()) {
      const response = await worker.request({ op: "infer", audio: { ...sample.audio, path: path.resolve(path.dirname(corpusFile), sample.audio.path) } });
      const filename = sample.split === "train" ? "warmups.jsonl" : "results.jsonl";
      await fs.appendFile(path.join(out, filename), JSON.stringify({ caseId: sample.id, response }) + "\n", { mode: 0o600 });
      if (index % 10 === 0) console.log(`Acoustic evidence ${index + 1}/${warmups.length + measured.length}`);
    }
    meta.status = "complete";
    meta.resultsHash = await hashFile(path.join(out, "results.jsonl"));
    meta.warmupsHash = await hashFile(path.join(out, "warmups.jsonl"));
  } catch (error) {
    meta.status = "failed"; meta.failure = error.message; throw error;
  } finally {
    if (worker) { try { await worker.close(); } finally { worker.child.kill(); } }
    await diagnostics?.close();
    await fs.writeFile(path.join(out, "run.json"), json(meta));
  }
}
async function evidenceLoaded(directory, corpusHash) {
  const meta = await read(path.join(directory, "run.json"));
  if (meta.status !== "complete" || meta.stage !== "acoustic-extraction" || meta.corpusHash !== corpusHash ||
      meta.resultsHash !== await hashFile(path.join(directory, "results.jsonl"))) throw new Error("Incomplete or mismatched acoustic evidence");
  const rows = await lines(path.join(directory, "results.jsonl"));
  const ids = rows.map(row => row.caseId);
  if (new Set(ids).size !== ids.length || ids.length !== meta.sampleIds.length || meta.sampleIds.some(id => !ids.includes(id))) throw new Error("Acoustic coverage mismatch");
  return { meta, index: new Map(rows.map(row => [row.caseId, row.response])) };
}
async function run() {
  const config = validateConfig(await read(absolute("config")));
  const corpusFile = absolute("corpus"), corpus = await loadCorpus(corpusFile);
  const corpusHash = await hashFile(corpusFile), evidence = await evidenceLoaded(absolute("evidence"), corpusHash);
  if (JSON.stringify(evidence.meta.config) !== JSON.stringify(config)) throw new Error("Extraction and correction configurations differ");
  const dev = corpus.samples.filter(sample => sample.split === "dev");
  const test = corpus.samples.filter(sample => sample.split === config.split);
  if (dev.length < config.warmupCount || !test.length) throw new Error("Development warmups and test samples required");
  const scores = dev.flatMap(sample => evidence.index.get(sample.id).segments.flatMap(segment => segment.logScores));
  const threshold = quantile(scores, config.scoreQuantile);
  if (!Number.isFinite(threshold)) throw new Error("Development acoustic scores unavailable");
  const preflightDirectory = absolute("preflight"), preflight = await read(path.join(preflightDirectory, "run.json"));
  const preflightResultsHash = await hashFile(path.join(preflightDirectory, "results.jsonl"));
  if (preflightResultsHash !== preflight.resultsHash) throw new Error("Changed preflight results");
  const preflightCheck = validatePreflight(preflight, await lines(path.join(preflightDirectory, "results.jsonl")), config,
    await hashFile(path.join(own, "design.mjs")), new Set(dev.map(sample => sample.id)));
  const out = absolute("out");
  await fs.mkdir(out, { mode: 0o700 });
  const meta = await metadata("correction", config, corpusFile);
  Object.assign(meta, { evidenceDirectory: absolute("evidence"), evidenceRunHash: await hashFile(path.join(absolute("evidence"), "run.json")),
    preflightDirectory, preflightRunHash: await hashFile(path.join(preflightDirectory, "run.json")), preflightResultsHash, preflightCheck,
    threshold, thresholdSource: "development token scores only; no reference optimization", sampleIds: test.map(sample => sample.id),
    runtimes: {}, modelArtifacts: {}, runtimeArtifacts: await read(absolute("runtime-artifacts")) });
  let worker, diagnostic;
  try {
    await artifactsVerified(meta.runtimeArtifacts);
    const allArtifacts = await read(absolute("llm-artifacts"));
    await fs.writeFile(path.join(out, "run.json"), json(meta));
    for (const sample of test) {
      const response = { text: evidence.index.get(sample.id).asrText, generationMs: 0, truncated: false };
      await fs.appendFile(path.join(out, "results.jsonl"), JSON.stringify({ caseId: sample.id, model: "raw", variant: "raw", response }) + "\n", { mode: 0o600 });
    }
    for (const model of config.models) {
      const original = path.join(absolute("models"), model), staged = path.join(out, model);
      await fs.mkdir(staged);
      const artifacts = await artifactsVerified(allArtifacts.filter(item => path.dirname(item.path) === original));
      const filenames = await fs.readdir(original);
      if (filenames.some(name => !artifacts.some(item => item.path === path.join(original, name)))) throw new Error("Unpinned LLM directory member");
      for (const name of filenames) if (name !== "genai_config.json") await fs.symlink(path.join(original, name), path.join(staged, name));
      const runtime = await read(path.join(original, "genai_config.json"));
      runtime.model.decoder.session_options.intra_op_num_threads = config.threads;
      runtime.model.decoder.session_options.inter_op_num_threads = 1;
      await fs.writeFile(path.join(staged, "genai_config.json"), json(runtime));
      artifacts.push({ path: path.join(staged, "genai_config.json"), sha256: await hashFile(path.join(staged, "genai_config.json")) });
      meta.modelArtifacts[model] = artifacts;
      diagnostic = await fs.open(path.join(out, `${model}-stderr.log`), "wx");
      worker = new Worker(absolute("python"), diagnostic.fd, path.join(own, "../diagnostic/qwen_worker.py"));
      meta.runtimes[model] = await worker.request({ op: "load", modelPath: staged, artifacts });
      await fs.writeFile(path.join(out, "run.json"), json(meta));
      const generate = async (sample, variant) => {
        const prompt = promptFor(evidence.index.get(sample.id), variant, threshold);
        const start = performance.now();
        const response = await worker.request({ op: "generate", prompt, maxInputTokens: config.maxInputTokens, maxNewTokens: config.maxNewTokens });
        return { caseId: sample.id, model, variant, prompt, response, parentWallMs: performance.now() - start };
      };
      for (const sample of dev.slice(0, config.warmupCount)) for (const variant of config.variants)
        await fs.appendFile(path.join(out, "warmups.jsonl"), JSON.stringify(await generate(sample, variant)) + "\n", { mode: 0o600 });
      const jobs = shuffle(test.flatMap(sample => config.variants.map(variant => ({ sample, variant }))), config.orderSeed);
      for (const [index, job] of jobs.entries()) {
        const row = await generate(job.sample, job.variant);
        await fs.appendFile(path.join(out, "results.jsonl"), JSON.stringify(row) + "\n", { mode: 0o600 });
        if (index % 20 === 0) console.log(`${model} ${index + 1}/${jobs.length}`);
      }
      await worker.close(); worker = undefined;
      await diagnostic.close(); diagnostic = undefined;
    }
    requireCoverage(await lines(path.join(out, "results.jsonl")), test, config.models, config.variants);
    meta.resultsHash = await hashFile(path.join(out, "results.jsonl"));
    meta.warmupsHash = await hashFile(path.join(out, "warmups.jsonl"));
    meta.status = "complete";
  } catch (error) {
    meta.status = "failed"; meta.failure = error.message; throw error;
  } finally {
    worker?.child.kill(); await diagnostic?.close();
    await fs.writeFile(path.join(out, "run.json"), json(meta));
  }
}
async function report() {
  const directory = absolute("results"), meta = await read(path.join(directory, "run.json"));
  if (meta.status !== "complete" || meta.stage !== "correction" || meta.resultsHash !== await hashFile(path.join(directory, "results.jsonl")) ||
      meta.warmupsHash !== await hashFile(path.join(directory, "warmups.jsonl")) || meta.corpusHash !== await hashFile(meta.corpusFile)) throw new Error("Incomplete run or changed results/corpus");
  const corpus = await loadCorpus(meta.corpusFile), config = validateConfig(meta.config);
  if (meta.preflightRunHash !== await hashFile(path.join(meta.preflightDirectory, "run.json")) ||
      meta.preflightResultsHash !== await hashFile(path.join(meta.preflightDirectory, "results.jsonl"))) throw new Error("Changed preflight evidence");
  if (meta.evidenceRunHash !== await hashFile(path.join(meta.evidenceDirectory, "run.json"))) throw new Error("Changed extraction provenance");
  const evidence = await evidenceLoaded(meta.evidenceDirectory, meta.corpusHash);
  const samples = corpus.samples.filter(sample => sample.split === config.split), index = new Map(samples.map(sample => [sample.id, sample]));
  const rows = await lines(path.join(directory, "results.jsonl"));
  requireCoverage(rows, samples, config.models, config.variants);
  for (const row of rows) {
    const sample = index.get(row.caseId);
    Object.assign(row, { clusterId: sample.speakerId, language: sample.language, unit: sample.unit, tags: sample.tags,
      seed: 11, cacheState: "warm", metrics: scoreRow(sample, evidence.index.get(row.caseId), row.response) });
    if (row.model !== "raw" && row.prompt !== promptFor(evidence.index.get(row.caseId), row.variant, meta.threshold)) throw new Error("Treatment prompt mismatch");
  }
  const groups = {};
  for (const row of rows) (groups[`${row.model}/${row.variant}`] ??= []).push(row);
  const summarize = values => ({ ...summarizeRows(values),
    outputContractFailures: values.filter(row => row.model !== "raw" && outputContractFailure(row.response)).length });
  const summaries = Object.fromEntries(Object.entries(groups).map(([key, values]) => [key, {
    overall: summarize(values),
    byLanguage: Object.fromEntries(["zh", "en", "mixed"].map(language => [language, summarize(values.filter(row => row.language === language))])),
    byTag: Object.fromEntries([...new Set(values.flatMap(row => row.tags))].map(tag => [tag, summarize(values.filter(row => row.tags.includes(tag)))])),
  }]));
  const comparisons = [];
  for (const [treatment, values] of Object.entries(groups)) {
    if (treatment === "raw/raw") continue;
    const model = treatment.split("/")[0];
    for (const baseline of ["raw/raw", `${model}/text`]) {
      if (baseline === treatment) continue;
      for (const unit of ["cer", "wer", "mer"]) {
        const select = group => group.filter(row => row.unit === unit && !row.metrics.silence);
        comparisons.push({ baseline, treatment, unit, ...pairedBootstrap(select(groups[baseline]), select(values), group => {
          const units = group.reduce((sum, row) => sum + row.metrics.corrected.referenceUnits, 0);
          return units ? group.reduce((sum, row) => sum + row.metrics.corrected.errors, 0) / units : null;
        }, { iterations: config.bootstrapIterations, seed: config.orderSeed }) });
      }
    }
  }
  const acoustic = [...evidence.index.values()];
  const summary = { kind: meta.kind, status: "complete", corpusHash: meta.corpusHash, runHash: await hashFile(path.join(directory, "run.json")), resultsHash: meta.resultsHash,
    config, host: meta.host, threshold: meta.threshold, testSamples: samples.length,
    speakers: new Set(samples.map(sample => sample.speakerId)).size,
    acousticTimings: Object.fromEntries(["asrMs", "phoneMs", "pinyinMs"].map(key => [key, distribution(acoustic.map(row => row[key]))])),
    summaries, comparisons,
    limitations: ["Exploratory paired speaker bootstrap; multiple comparisons unadjusted", "Accelerated acoustic replay; warm CPU generation, no UI/insertion or stop-to-review latency", "Natural read speech and synthetic mixed-language stress fixtures; no spontaneous speech or independent semantic ratings", "Audio IPA comes from unfine-tuned Allosaurus uni2005; single phone path, no tone-aware pinyin acoustic model", "No real N-best exposed by selected ASR; no candidate reranking claim", "No protected-span annotations in this corpus; error-rate and corruption proxies do not prove intent preservation"] };
  await fs.writeFile(absolute("out"), json(summary), { flag: "wx" });
  console.log(`Report: ${absolute("out")}`);
}

if (command === "extract") await extract();
else if (command === "run") await run();
else if (command === "report") await report();
else throw new Error("Expected extract, run or report");
