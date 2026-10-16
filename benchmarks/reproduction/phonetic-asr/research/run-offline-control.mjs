import fs from "node:fs/promises";
import path from "node:path";
import { Worker } from "../../../tools/evaluation/diagnostic/worker.mjs";
import { loadCorpus, hashFile } from "../../../tools/evaluation/src/manifest.mjs";
import { errorCounts, distribution } from "../../../tools/evaluation/src/metrics.mjs";

const root = process.cwd(), out = path.resolve(".workspace/benchmarks/phonetic-asr-v2/offline-control");
const corpusFile = path.resolve(".workspace/corpus/v1/corpus.json");
const corpus = await loadCorpus(corpusFile);
const complete = JSON.parse(await fs.readFile(".workspace/benchmarks/phonetic-asr-v2/completed-17b-view/run.json", "utf8"));
if (complete.status !== "complete" || complete.derivedAnalysisScope?.sourceStatus !== "cancelled_by_user") throw new Error("Run offline control after LLM inference to avoid added contention");
const catalog = JSON.parse(await fs.readFile("models/catalog.json", "utf8"));
const model = catalog.artifacts.find(model => model.id === "x-asr-zh-en-offline-punct-int8-2026-06-03");
const files = Object.fromEntries(Object.entries(model.files).map(([key, file]) => [key, path.resolve(".workspace/models/store", model.id, path.basename(file.member))]));
const artifacts = Object.entries(model.files).map(([key, file]) => ({ path: files[key], sha256: file.sha256 }));
const runtimeArtifacts = JSON.parse(await fs.readFile(".workspace/phonetic-models/acoustic-artifacts.json", "utf8"));
artifacts.push(...runtimeArtifacts.filter(artifact => artifact.path.includes("/sherpa_onnx/") || artifact.path.includes("/numpy/") || artifact.path.includes("/bin/python")));
const api = path.resolve(".workspace/phonetic-venv/lib/python3.12/site-packages/sherpa_onnx/offline_recognizer.py");
artifacts.push({ path: api, sha256: await hashFile(api) });
const workerFile = path.resolve(".workspace/research/phonetic-asr/offline_control.py");
artifacts.push({ path: workerFile, sha256: await hashFile(workerFile) });
for (const artifact of artifacts) if (await hashFile(artifact.path) !== artifact.sha256) throw new Error("Offline asset mismatch");
await fs.mkdir(out, { mode: 0o700 });
const metadata = { status: "running", corpusHash: await hashFile(corpusFile), model: model.id, artifacts,
  controllerHash: await hashFile(new URL(import.meta.url)), host: complete.host, threads: 4,
  note: "Whole-file offline acoustic second pass control; no LLM and no realtime preview. Inputs contain audio only." };
await fs.writeFile(path.join(out, "run.json"), JSON.stringify(metadata, null, 2));
const diagnostics = await fs.open(path.join(out, "stderr.log"), "wx");
const worker = new Worker(path.resolve(".workspace/phonetic-venv/bin/python"), diagnostics.fd, workerFile);
try {
  metadata.runtime = await worker.request({ op: "load", files, artifacts, modelingUnit: model.modelingUnit });
  const warmups = corpus.samples.filter(sample => sample.split === "dev").slice(0, 2);
  const samples = corpus.samples.filter(sample => sample.split === "test");
  const rows = [];
  for (const [index, sample] of [...warmups, ...samples].entries()) {
    if (sample.audio.durationMs > 25000) throw new Error("Whole-file control requires production-compatible segment bound");
    const response = await worker.request({ op: "infer", audio: { ...sample.audio, path: path.resolve(path.dirname(corpusFile), sample.audio.path) } });
    const row = { caseId: sample.id, response };
    if (sample.split === "test") rows.push({ ...row, language: sample.language, unit: sample.unit,
      metrics: errorCounts(sample.reference, response.text, sample.unit), silence: sample.reference === "" });
    await fs.appendFile(path.join(out, sample.split === "test" ? "results.jsonl" : "warmups.jsonl"), JSON.stringify(row) + "\n");
    if (index % 10 === 0) console.log(`Offline control ${index + 1}/${warmups.length + samples.length}`);
  }
  if (rows.length !== 154) throw new Error("Offline coverage mismatch");
  const summary = { model: model.id, count: rows.length, rates: {} };
  for (const unit of ["cer", "wer", "mer"]) {
    const subset = rows.filter(row => row.unit === unit && !row.silence);
    const errors = subset.reduce((n, row) => n + row.metrics.errors, 0), referenceUnits = subset.reduce((n, row) => n + row.metrics.referenceUnits, 0);
    summary.rates[unit] = { count: subset.length, errors, referenceUnits, rate: errors / referenceUnits };
  }
  summary.computeMs = distribution(rows.map(row => row.response.computeMs));
  summary.silenceCount = rows.filter(row => row.silence).length;
  summary.falseEmissions = rows.filter(row => row.silence && row.response.text.trim()).length;
  metadata.resultsHash = await hashFile(path.join(out, "results.jsonl"));
  metadata.warmupsHash = await hashFile(path.join(out, "warmups.jsonl"));
  metadata.status = "complete";
  await fs.writeFile(path.join(out, "summary.json"), JSON.stringify(summary, null, 2));
} catch (error) {
  metadata.status = "failed"; metadata.failure = error.message; throw error;
} finally {
  await worker.close(); await diagnostics.close();
  await fs.writeFile(path.join(out, "run.json"), JSON.stringify(metadata, null, 2));
}
