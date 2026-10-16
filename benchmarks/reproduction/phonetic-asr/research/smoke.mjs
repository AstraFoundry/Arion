import fs from "node:fs/promises";
import path from "node:path";
import { Worker } from "../../../tools/evaluation/diagnostic/worker.mjs";
import { promptFor, VARIANTS } from "../../../tools/evaluation/phonetic/design.mjs";
import { loadCorpus, hashFile } from "../../../tools/evaluation/src/manifest.mjs";

const out = path.resolve(process.argv[2]);
await fs.mkdir(out, { mode: 0o700 });
const corpus = await loadCorpus(path.resolve(".workspace/corpus/v1/corpus.json"));
const samples = ["zh", "en", "mixed"].map(language => corpus.samples.find(sample => sample.split === "dev" && sample.language === language));
const evidence = new Map((await fs.readFile(".workspace/benchmarks/phonetic-asr-v1/acoustic/results.jsonl", "utf8")).trim().split("\n").map(JSON.parse).map(row => [row.caseId, row.response]));
const initial = JSON.parse(await fs.readFile(".workspace/benchmarks/phonetic-asr-v1/correction/run.json", "utf8"));
const meta = { kind: "development-output-contract-smoke", status: "running", testCases: 0,
  promptControllerHash: await hashFile("tools/evaluation/phonetic/design.mjs"), models: {}, records: 0 };
try {
  for (const model of ["qwen3-1.7b", "qwen3-4b"]) {
    const diagnostics = await fs.open(path.join(out, model + "-stderr.log"), "wx");
    const worker = new Worker(path.resolve(".workspace/ablation-venv/bin/python"), diagnostics.fd, path.resolve("tools/evaluation/diagnostic/qwen_worker.py"));
    try {
      const staged = path.resolve(".workspace/benchmarks/phonetic-asr-v1/correction", model);
      meta.models[model] = await worker.request({ op: "load", modelPath: staged, artifacts: initial.modelArtifacts[model] });
      for (const sample of samples) for (const variant of VARIANTS) {
        const prompt = promptFor(evidence.get(sample.id), variant, initial.threshold);
        const response = await worker.request({ op: "generate", prompt, maxInputTokens: 4096, maxNewTokens: 256 });
        const contractFailure = response.truncated || /^\s*[\[{]/u.test(response.text) || /<\|im_|<\/?think>/u.test(response.text);
        const row = { caseId: sample.id, split: "dev", language: sample.language, model, variant, prompt, response, contractFailure };
        await fs.appendFile(path.join(out, "results.jsonl"), JSON.stringify(row) + "\n");
        meta.records++;
        console.log(model, sample.language, variant, contractFailure ? "FORMAT-FAIL" : "plain", JSON.stringify(response.text.slice(0, 130)));
      }
    } finally {
      await worker.close(); await diagnostics.close();
    }
  }
  const rows = (await fs.readFile(path.join(out, "results.jsonl"), "utf8")).trim().split("\n").map(JSON.parse);
  meta.contractFailures = rows.filter(row => row.contractFailure).length;
  meta.resultsHash = await hashFile(path.join(out, "results.jsonl"));
  meta.status = "complete";
} catch (error) {
  meta.status = "failed"; meta.failure = error.message; throw error;
} finally {
  await fs.writeFile(path.join(out, "run.json"), JSON.stringify(meta, null, 2));
}
