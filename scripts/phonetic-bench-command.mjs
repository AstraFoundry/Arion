import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const stage = process.argv[2];
const fields = {
  extract: { corpus: "CORPUS", python: "EVAL_PYTHON", models: "MODELS", "phone-models": "PHONE_MODELS", "runtime-artifacts": "RUNTIME_ARTIFACTS", out: "OUT" },
  run: { corpus: "CORPUS", python: "EVAL_PYTHON", models: "MODELS", "runtime-artifacts": "RUNTIME_ARTIFACTS", "llm-artifacts": "LLM_ARTIFACTS", evidence: "EVIDENCE", preflight: "PHONETIC_PREFLIGHT", out: "OUT" },
  report: { results: "RESULTS", out: "OUT" },
}[stage];
if (!fields) throw new Error("Expected extract, run or report");
const args = [fileURLToPath(new URL("../tools/evaluation/phonetic/cli.mjs", import.meta.url)), stage];
if (stage !== "report") args.push("--config", process.env.PHONETIC_CONFIG ?? "benchmarks/configs/phonetic-asr.json");
for (const [flag, name] of Object.entries(fields)) {
  if (!process.env[name]) throw new Error(`${name} is required`);
  args.push(`--${flag}`, process.env[name]);
}
const result = spawnSync(process.execPath, args, { stdio: "inherit" });
if (result.error) throw result.error;
process.exitCode = result.status ?? 1;
