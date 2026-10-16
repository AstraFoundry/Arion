// Portable orchestration for closed-loop memory and context research.
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import crypto from "node:crypto";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const operation = process.argv[2];
const argument = (key) => {
  if (!process.env[key]) throw new Error(`${key} required`);
  return process.env[key];
};
const invoke = (command, args, cwd = root, env = process.env) => {
  const result = spawnSync(command, args, { cwd, env, stdio: "inherit", shell: false });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
};
if (operation === "verify") {
  const manifests = process.argv.slice(3);
  if (!manifests.length) manifests.push("benchmarks/results/user-modify-asr-v3/archive.json", "benchmarks/results/context-memory-v1/archive.json");
  let count = 0;
  for (const manifest of manifests) {
    const archive = JSON.parse(fs.readFileSync(path.resolve(root, manifest), "utf8"));
    if (archive.status !== "complete" || archive.version !== 1) throw new Error("Complete version-one archive required");
    const seen = new Set();
    for (const file of archive.files) {
      const filename = path.resolve(root, file.path);
      if (!filename.startsWith(root + path.sep) || seen.has(file.path)) throw new Error("Invalid or duplicate archive path");
      seen.add(file.path);
      const bytes = fs.readFileSync(filename);
      const digest = crypto.createHash("sha256").update(bytes).digest("hex");
      if (digest !== file.sha256 || bytes.length !== file.sizeBytes) throw new Error(`Archive identity mismatch: ${file.path}`);
      count++;
    }
  }
  console.log(`Verified ${count} file identities across ${manifests.length} research archives.`);
} else if (operation === "build") {
  const directory = path.resolve(root, process.env.MEMORY_BIN_DIR || ".workspace/research/user-modify-asr-v3");
  for (const name of ["arion-memory-loop", "arion-context-probe"]) {
    invoke("go", ["build", "-o", path.join(directory, name + (process.platform === "win32" ? ".exe" : "")), `./cmd/${name}`], path.join(root, "apps/desktop"), { ...process.env, CGO_ENABLED: "0" });
  }
} else {
  const settings = {
    run: ["user-modify-asr-v3", "run.py"],
    report: ["user-modify-asr-v3", "report.mjs"],
    context: ["context-memory-v1", "run.py"],
    "context-report": ["context-memory-v1", "report.mjs"],
  }[operation];
  if (!settings) throw new Error("build, run, report, context or context-report required");
  const report = operation.endsWith("report");
  if (!report && (process.platform !== "darwin" || process.arch !== "arm64")) {
    throw new Error("New inference in these studies requires the provisioned macOS arm64 runtime; report replay is portable");
  }
  let args = process.argv.slice(3);
  if (!args.length) {
    args = report ? [argument("CORPUS"), argument("RESULTS"), argument("OUT")] : ["--fixtures", argument("CORPUS"), "--split", argument("MEMORY_SPLIT"), "--model", argument("MODELS"), "--out", argument("OUT")];
    if (!report) {
      if (operation === "context") args.push("--memory-driver", argument("MEMORY_DRIVER"), "--context-driver", argument("CONTEXT_DRIVER"), "--worker-python", argument("WORKER_PYTHON"), "--model-metadata", argument("MODEL_METADATA"));
      else args.push("--driver", argument("MEMORY_DRIVER"));
      if (process.env.MEMORY_SELECTION) args.push("--selection", process.env.MEMORY_SELECTION);
    }
  }
  const source = path.join(root, "benchmarks/reproduction", ...settings);
  const command = report ? process.execPath : process.env.EVAL_PYTHON || path.join(root, ".workspace/user-modify-asr-venv/bin/python");
  invoke(command, [source, ...args], root, { ...process.env, HF_HUB_OFFLINE: "1", PYTHONDONTWRITEBYTECODE: "1" });
}
