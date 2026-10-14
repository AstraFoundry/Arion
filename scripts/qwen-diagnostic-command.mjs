import { spawnSync } from "node:child_process";
import path from "node:path";
import fs from "node:fs";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const operation = process.argv[2];
// Make exports unset optional variables as empty strings.
const option = (key) =>
  process.env[key] === "" ? undefined : process.env[key];
const python =
  option("EVAL_PYTHON") ??
  path.join(
    root,
    ".workspace/ablation-venv",
    process.platform === "win32" ? "Scripts/python.exe" : "bin/python",
  );
const models =
  option("MODELS") ?? path.join(root, ".workspace/ablation-models");
const invoke = (command, args) => {
  const result = spawnSync(command, args, {
    cwd: root,
    stdio: "inherit",
    shell: false,
  });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
};
if (operation === "env") {
  if (fs.existsSync(path.join(root, ".workspace/ablation-venv")))
    throw new Error(
      "Diagnostic environment already exists; do not overwrite an active runtime",
    );
  invoke("uv", [
    "venv",
    "--python",
    "3.12",
    path.join(root, ".workspace/ablation-venv"),
  ]);
  invoke("uv", [
    "pip",
    "install",
    "--python",
    python,
    "-r",
    "tools/evaluation/diagnostic/requirements.lock",
  ]);
} else if (operation === "provision") {
  if (!process.env.MODEL_IDS)
    throw new Error("MODEL_IDS required; for example qwen3-0.6b,qwen3-1.7b");
  invoke(python, [
    "-I",
    "tools/evaluation/diagnostic/provision_qwen.py",
    "--models",
    models,
    "--ids",
    process.env.MODEL_IDS,
    ...(process.env.QWEN_ASR === "1" ? ["--asr"] : []),
  ]);
} else if (operation === "prefix") {
  for (const key of ["CORPUS", "AUDIO_CORPUS", "DRIVER", "OUT"])
    if (!process.env[key]) throw new Error(`${key} required`);
  invoke(process.execPath, [
    "tools/evaluation/diagnostic/prefix-run.mjs",
    "--config",
    option("PREFIX_CONFIG") ?? "benchmarks/configs/qwen-prefix-diagnostic.json",
    "--corpus",
    process.env.CORPUS,
    "--audio-corpus",
    process.env.AUDIO_CORPUS,
    "--driver",
    process.env.DRIVER,
    "--models",
    models,
    "--python",
    python,
    "--prompts",
    option("QWEN_PROMPTS") ??
      "tools/evaluation/diagnostic/prompts-intent-v2.json",
    "--out",
    process.env.OUT,
  ]);
} else if (operation === "asr") {
  for (const key of ["CORPUS", "DRIVER", "OUT"])
    if (!process.env[key]) throw new Error(`${key} required`);
  invoke(process.execPath, [
    "tools/evaluation/diagnostic/asr-run.mjs",
    "--corpus",
    process.env.CORPUS,
    "--driver",
    process.env.DRIVER,
    "--out",
    process.env.OUT,
  ]);
} else if (["run", "report"].includes(operation)) {
  const required =
    operation === "run" ? ["CORPUS", "OUT"] : ["CORPUS", "RESULTS"];
  for (const key of required)
    if (!process.env[key]) throw new Error(`${key} required`);
  const args = [
    "tools/evaluation/qwen-diagnostic.mjs",
    operation,
    "--corpus",
    process.env.CORPUS,
  ];
  if (operation === "run")
    args.push(
      "--config",
      option("QWEN_CONFIG") ?? "benchmarks/configs/qwen-diagnostic.json",
      "--models",
      models,
      "--python",
      python,
      "--out",
      process.env.OUT,
    );
  else args.push("--results", process.env.RESULTS);
  if (process.env.QWEN_PROMPTS)
    args.push("--prompts", process.env.QWEN_PROMPTS);
  invoke(process.execPath, args);
} else throw new Error("Unknown Qwen diagnostic operation");
