// Portable entry point for the local recognition-stage memory study.
// Actual inference requires the separately provisioned Apple Silicon runtime.
import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const operation = process.argv[2];
const option = (key) => process.env[key] || undefined;
const required = (...keys) => {
  for (const key of keys) if (!option(key)) throw new Error(`${key} required`);
};
const source = "benchmarks/reproduction/user-modify-asr-v2";
let command;
let args = process.argv.slice(3);
if (operation === "run") {
  if (process.platform !== "darwin") throw new Error("This study requires Apple Silicon and MLX");
  command = option("EVAL_PYTHON") ?? path.join(root, ".workspace/user-modify-asr-venv/bin/python");
  if (!args.length) {
    required("CORPUS", "MODELS", "OUT");
    args = ["--fixtures", option("CORPUS"), "--model", option("MODELS"), "--out", option("OUT")];
    if (option("MEMORY_SELECTION")) args.push("--selection", option("MEMORY_SELECTION"));
  }
  args = [`${source}/run.py`, ...args];
} else if (operation === "report") {
  command = process.execPath;
  if (!args.length) {
    required("CORPUS", "RESULTS", "OUT");
    args = [option("CORPUS"), option("RESULTS"), option("OUT")];
  }
  args = [`${source}/report.mjs`, ...args];
} else {
  throw new Error("usage: asr-memory-bench-command.mjs run|report [arguments]");
}
const result = spawnSync(command, args, {
  cwd: root,
  stdio: "inherit",
  shell: false,
  env: { ...process.env, ...(operation === "run" ? { HF_HUB_OFFLINE: "1" } : {}) },
});
if (result.error) throw result.error;
process.exit(result.status ?? 1);
