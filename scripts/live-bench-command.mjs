// Entry points for the live rewrite pipeline benchmarks. Binaries are built
// into .workspace/bin; fixtures, weights and rows stay in .workspace.
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const operation = process.argv[2];
// Make exports unset optional variables as empty strings.
const option = (key) => (process.env[key] === "" ? undefined : process.env[key]);
const required = (...keys) => {
  for (const key of keys) if (!option(key)) throw new Error(`${key} required`);
};
const invoke = (command, args, env = {}, cwd = root) => {
  const result = spawnSync(command, args, { cwd, stdio: "inherit", shell: false, env: { ...process.env, ...env } });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
};
const bin = path.join(root, ".workspace/bin");
const build = (pkg, name, cgo) =>
  invoke("go", ["build", "-o", path.join(bin, name), pkg], { CGO_ENABLED: cgo ? "1" : "0" }, path.join(root, "apps/desktop"));
const runtimeDir = () => option("LLM_RUNTIME") ?? path.join(root, "apps/desktop/resources/darwin-arm64/llm");
const score = (script, ...args) => invoke(process.execPath, [`tools/evaluation/live/${script}`, ...args]);

if (process.platform !== "darwin") throw new Error("The live benchmarks currently run on macOS only");
fs.mkdirSync(bin, { recursive: true });
if (operation === "fixtures") {
  required("CORPUS", "OUT");
  invoke("python3", ["-I", "tools/evaluation/live/build-live-audio.py", option("CORPUS"), option("OUT")]);
} else if (operation === "text") {
  required("CORPUS", "LLM_MODEL", "OUT");
  build("./cmd/arion-llm", "arion-llm", false);
  build("./cmd/arion-rewrite-bench", "arion-rewrite-bench", false);
  invoke(path.join(bin, "arion-rewrite-bench"), ["-corpus", option("CORPUS"), "-worker", path.join(bin, "arion-llm"), "-runtime", runtimeDir(),
    "-model", option("LLM_MODEL"), "-variant", option("VARIANT") ?? path.basename(option("LLM_MODEL")), "-scene", option("SCENE") ?? "corpus",
    ...(option("CORRECT") === "1" ? ["-correct"] : []), ...(option("CONTEXT") ? ["-context", option("CONTEXT")] : []),
    ...(option("SCREEN") ? ["-screen", option("SCREEN")] : []), ...(option("OCR") ? ["-ocr", option("OCR")] : []),
    ...(option("CONTEXT_AS") ? ["-context-as", option("CONTEXT_AS")] : []), "-out", option("OUT")]);
  score("score-text.mjs", option("CORPUS"), option("OUT"));
} else if (operation === "audio") {
  required("MANIFEST", "CORPUS", "MODELS", "BUNDLE", "OUT");
  build("./cmd/arion-llm", "arion-llm", false);
  build("./cmd/arion-live-bench", "arion-live-bench", true);
  const args = ["-manifest", option("MANIFEST"), "-corpus", option("CORPUS"), "-models", option("MODELS"), "-bundle", option("BUNDLE"),
    "-variant", option("VARIANT") ?? option("BUNDLE"), "-out", option("OUT")];
  if (option("SEGMENT") === "1") args.push("-segment-finalize");
  if (option("LLM_MODEL")) args.push("-llm-worker", path.join(bin, "arion-llm"), "-llm-runtime", runtimeDir(), "-llm-model", option("LLM_MODEL"));
  // caffeinate keeps the host awake so idle sleep cannot stretch latencies.
  invoke("caffeinate", ["-i", path.join(bin, "arion-live-bench"), ...args]);
  score("score-audio.mjs", option("MANIFEST"), option("CORPUS"), option("OUT"));
} else if (operation === "ocr") {
  required("CORPUS", "OUT");
  build("./cmd/arion-ocr", "arion-ocr", true);
  invoke(path.join(bin, "arion-ocr"), ["-corpus", option("CORPUS"), "-out", option("OUT")]);
  score("score-ocr.mjs", option("CORPUS"), option("OUT"));
} else if (operation === "scene") {
  invoke("go", ["run", "./cmd/arion-scene-bench", "-misses", "-contexts", path.resolve(root, option("CONTEXTS") ?? "benchmarks/configs/scene-contexts.json")],
    { CGO_ENABLED: "0" }, path.join(root, "apps/desktop"));
} else if (operation === "report") {
  required("RUNS", "OUT");
  score("report.mjs", option("RUNS"), option("OUT"));
} else {
  throw new Error("usage: live-bench-command.mjs fixtures|text|audio|report|scene|ocr");
}
