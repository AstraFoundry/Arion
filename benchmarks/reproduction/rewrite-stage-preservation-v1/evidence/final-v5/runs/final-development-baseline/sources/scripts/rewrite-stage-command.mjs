// Portable, explicit entry point for native rewrite-stage experiments.
// No downloads, installed app changes, or fabricated inference are performed.
import { createHash } from "node:crypto";
import { spawn, spawnSync } from "node:child_process";
import { createReadStream, createWriteStream } from "node:fs";
import * as fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { finished } from "node:stream/promises";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const study = path.join(root, "benchmarks/reproduction/rewrite-stage-preservation-v1");
const python = process.env.PYTHON || "python3";
const operation = process.argv[2];
const arguments_ = process.argv.slice(3);
const usage = `rewrite-stage-command.mjs run --arm baseline|candidate|no-prior|no-screen --cases PATH --out NEWDIR --runtime PATH --model PATH
rewrite-stage-command.mjs report [report.py arguments]
rewrite-stage-command.mjs verify REPORT_DIRECTORY
rewrite-stage-command.mjs test`;

function execute(command, args) {
  const result = spawnSync(command, args, { cwd: root, stdio: "inherit", shell: false });
  if (result.error) throw result.error;
  process.exitCode = result.status ?? 1;
}
function probe(command, args) {
  const result = spawnSync(command, args, { cwd: root, encoding: "utf8", shell: false });
  return { status: result.status, stdout: result.stdout ?? "", stderr: result.stderr ?? "", error: result.error?.message ?? "" };
}
async function sha256(file) {
  const hash = createHash("sha256");
  for await (const chunk of createReadStream(file)) hash.update(chunk);
  return hash.digest("hex");
}
async function writeJSON(file, value) {
  await fs.writeFile(file, `${JSON.stringify(value, null, 2)}\n`, { flag: "wx" });
}
async function filesUnder(directory) {
  const result = [];
  for (const entry of await fs.readdir(directory, { withFileTypes: true })) {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) result.push(...await filesUnder(file));
    else if (entry.isFile() || entry.isSymbolicLink()) result.push(file);
  }
  return result.sort();
}
async function existing(file) {
  try { await fs.access(file); return true; } catch (error) {
    if (error.code === "ENOENT") return false;
    throw error;
  }
}
function options(args) {
  const allowed = new Set(["arm", "cases", "out", "runtime", "model"]);
  const values = {};
  for (let i = 0; i < args.length; i += 2) {
    const name = args[i].replace(/^--/, "");
    if (!args[i].startsWith("--") || !allowed.has(name) || !args[i + 1] || args[i + 1].startsWith("--") || name in values) {
      throw new Error(`Invalid or duplicate option ${args[i]}\n${usage}`);
    }
    values[name] = args[i + 1];
  }
  for (const name of allowed) if (!values[name]) throw new Error(`--${name} is required`);
  if (!["baseline", "candidate", "no-prior", "no-screen"].includes(values.arm)) throw new Error("Unsupported arm");
  for (const name of ["cases", "out", "runtime", "model"]) values[name] = path.resolve(root, values[name]);
  return values;
}

async function run() {
  const config = options(arguments_);
  // Claim the run directory before validation so failed attempts remain visible.
  await fs.mkdir(config.out, { recursive: false });
  const started = new Date().toISOString();
  await writeJSON(path.join(config.out, "attempt.json"), { status: "started", started, operation, config });
  let status;
  try {
    const cases = JSON.parse(await fs.readFile(config.cases, "utf8"));
    if (!Array.isArray(cases) || !cases.length || new Set(cases.map(c => c.id)).size !== cases.length ||
        cases.some(c => typeof c.id !== "string" || !c.id || ["scene", "preview", "final", "screen"].some(k => typeof c[k] !== "string"))) {
      throw new Error("Cases require unique IDs and explicit scene/preview/final/screen strings");
    }
    const catalogPath = path.join(root, "models/catalog.json");
    const catalog = JSON.parse(await fs.readFile(catalogPath, "utf8"));
    const product = catalog.rewriters.find(item => item.id === "qwen3.5-2b-q4-k-m-metal");
    const modelName = "Qwen3.5-2B-Q4_K_M.gguf";
    const expected = product?.files?.[modelName];
    if (!expected?.sha256) throw new Error("Selected product GGUF digest is missing from catalog");
    const modelFile = path.join(config.model, modelName);
    const model = { path: modelFile, sha256: await sha256(modelFile), sizeBytes: (await fs.stat(modelFile)).size,
      expectedSha256: expected.sha256, catalogID: product.id, runtimeContract: product.runtime };
    await writeJSON(path.join(config.out, "model-identity.json"), model);
    if (model.sha256 !== expected.sha256 || model.sizeBytes !== expected.sizeBytes) throw new Error("Local GGUF does not match pinned product catalog digest/size");
    const runtimeFiles = [];
    if (!await existing(path.join(config.runtime, "llama-server"))) throw new Error("Supplied runtime has no llama-server");
    for (const file of await filesUnder(config.runtime)) {
      const stat = await fs.stat(file);
      if (stat.isFile()) runtimeFiles.push({ path: path.relative(config.runtime, file), sha256: await sha256(file), sizeBytes: stat.size });
    }
    await writeJSON(path.join(config.out, "runtime-identity.json"), {
      path: config.runtime, identityPolicy: "Observed hashes; app signing/staging may change executable hashes. No claim of upstream binary identity.", files: runtimeFiles,
    });

    const sourceFiles = new Set(["go.work", "apps/desktop/go.mod", "apps/desktop/go.sum", "packages/core/go.mod", "models/go.mod", "models/catalog.json", "scripts/rewrite-stage-command.mjs"]);
    for (const directory of ["apps/desktop/infrastructure/llm/gguf", "apps/desktop/infrastructure/llm/prompt", "apps/desktop/infrastructure/llm/spelling", "apps/desktop/infrastructure/llm/engine", "packages/core/domain", "packages/core/application"]) {
      for (const file of await filesUnder(path.join(root, directory))) {
        if (/\.(go|json|dict)$/.test(file)) sourceFiles.add(path.relative(root, file));
      }
    }
    // Include other changed source dependencies while excluding execution output.
    const changes = probe("git", ["diff", "--name-only", "-z", "HEAD", "--", "apps", "packages/core", "models", "scripts"]);
    if (changes.status !== 0) throw new Error(`Cannot snapshot Git source changes: ${changes.stderr || changes.error}`);
    for (const relative of changes.stdout.split("\0").filter(Boolean)) {
      if (/\.(go|json|mjs|mod|sum)$/.test(relative) && await existing(path.join(root, relative))) sourceFiles.add(relative);
    }
    const manifest = {};
    const sources = path.join(config.out, "sources");
    for (const relative of [...sourceFiles].sort()) {
      const from = path.join(root, relative);
      if (!await existing(from)) continue;
      const to = path.join(sources, relative);
      await fs.mkdir(path.dirname(to), { recursive: true });
      await fs.copyFile(from, to, 1);
      manifest[relative] = await sha256(to);
    }
    await fs.copyFile(config.cases, path.join(config.out, "cases.json"), 1);
    // Compile exactly the archived source snapshot, even if another task edits
    // the checkout after this point. Baseline overrides only its frozen files.
    const overlay = Object.fromEntries(Object.keys(manifest).map(relative =>
      [path.join(root, relative), path.join(sources, relative)]));
    if (config.arm === "baseline") {
      // Candidate-only white-box tests reference helpers absent from the
      // historical engine. Keep the native stage harness, but do not compile
      // those tests against a different implementation revision.
      const candidateOnlyTest = path.join(root, "apps/desktop/infrastructure/llm/gguf/ambiguity_test.go");
      if (await existing(candidateOnlyTest)) {
        const omittedTest = path.join(config.out, "baseline-candidate-test.go");
        await fs.writeFile(omittedTest, "package gguf\n", { flag: "wx" });
        overlay[candidateOnlyTest] = omittedTest;
      }
      for (const [relative, short] of [["apps/desktop/infrastructure/llm/gguf/engine.go", "engine.go"], ["apps/desktop/infrastructure/llm/prompt/selected.go", "selected.go"]]) {
        const frozenTree = path.join(study, "source/baseline", relative);
        const frozenFlat = path.join(study, "source/baseline", short);
        const frozen = await existing(frozenTree) ? frozenTree : frozenFlat;
        if (!await existing(frozen)) throw new Error(`Missing frozen baseline source: ${relative}`);
        const to = path.join(config.out, "baseline-overlay", relative);
        await fs.mkdir(path.dirname(to), { recursive: true });
        await fs.copyFile(frozen, to, 1);
        overlay[path.join(root, relative)] = to;
      }
    }
    await writeJSON(path.join(config.out, "overlay.json"), { Replace: overlay });
    await writeJSON(path.join(config.out, "source-manifest.json"), { files: manifest,
      overlays: Object.fromEntries(await Promise.all(Object.entries(overlay).map(async ([target, file]) => [target, { path: file, sha256: await sha256(file) }]))),
      cases: { original: config.cases, archived: "cases.json", sha256: await sha256(path.join(config.out, "cases.json")) } });
    const goArgs = ["test", `-overlay=${path.join(config.out, "overlay.json")}`];
    goArgs.push("./apps/desktop/infrastructure/llm/gguf", "-run", "^TestNativeStagePreservation$", "-count=1", "-timeout=20m");
    const environment = { ...process.env };
    // Prevent inherited diagnostic/filter flags from silently changing an arm.
    for (const key of Object.keys(environment)) if (key.startsWith("ARION_STAGE_")) delete environment[key];
    Object.assign(environment, { ARION_GGUF_RUNTIME: config.runtime, ARION_GGUF_MODEL: config.model,
      ARION_STAGE_CASES: path.join(config.out, "cases.json"), ARION_STAGE_OUT: path.join(config.out, "observations.jsonl") });
    if (config.arm === "no-prior") environment.ARION_STAGE_NO_PREVIOUS = "1";
    if (config.arm === "no-screen") environment.ARION_STAGE_NO_SCREEN = "1";
    const effectiveEnvironment = Object.fromEntries(Object.entries(environment).filter(([key]) => key.startsWith("ARION_STAGE_") || key.startsWith("ARION_GGUF_")));
    await writeJSON(path.join(config.out, "provenance.json"), { started, arm: config.arm, command: ["go", ...goArgs],
      effectiveEnvironment, revision: probe("git", ["rev-parse", "HEAD"]), dirtyState: probe("git", ["status", "--porcelain=v1"]),
      sourceDiff: probe("git", ["diff", "HEAD", "--", ...[...sourceFiles].filter(f => /\.(go|json|mjs|mod|sum)$/.test(f))]),
      host: { platform: process.platform, arch: process.arch, release: os.release(), cpus: os.cpus(), totalMemoryBytes: os.totalmem() },
      powerMode: process.platform === "darwin" ? probe("pmset", ["-g", "custom"]) : { unavailable: "Power-mode probe is implemented for macOS only" },
      go: probe("go", ["version"]), node: process.version,
      settings: { threads: 4, contextTokens: 4096, maxNewTokens: 256, observations: "Actual pinned GGUF inference; constructed/native input origins remain fixture-specific" },
      baseline: config.arm === "baseline" ? "Frozen pre-change engine.go and selected.go via Go overlay; current legacy Prepare remains available" : null });
    console.log(`Running ${config.arm}; logs and observations: ${config.out}`);
    const stdout = createWriteStream(path.join(config.out, "stdout.log"), { flags: "wx" });
    const stderr = createWriteStream(path.join(config.out, "stderr.log"), { flags: "wx" });
    const result = await new Promise((resolve) => {
      const child = spawn("go", goArgs, { cwd: root, env: environment, shell: false, stdio: ["ignore", "pipe", "pipe"] });
      child.stdout.pipe(stdout);
      child.stderr.pipe(stderr);
      child.on("error", (error) => resolve({ code: null, signal: null, error: error.message }));
      child.on("close", (code, signal) => resolve({ code, signal }));
    });
    await Promise.all([finished(stdout), finished(stderr)]);
    let completed = false;
    const observations = path.join(config.out, "observations.jsonl");
    if (await existing(observations)) {
      const lines = (await fs.readFile(observations, "utf8")).trim().split("\n");
      completed = JSON.parse(lines.at(-1)).kind === "complete";
    }
    status = { status: result.code === 0 && completed ? "completed" : "failed", ...result,
      evidenceComplete: completed, quality: "No automatic quality pass; review exact observations and paired report", finished: new Date().toISOString() };
  } catch (error) {
    status = { status: "failed", error: error.stack || error.message, finished: new Date().toISOString() };
  }
  const archived = {};
  for (const file of await filesUnder(config.out)) archived[path.relative(config.out, file)] = await sha256(file);
  await writeJSON(path.join(config.out, "status.json"), status);
  archived["status.json"] = await sha256(path.join(config.out, "status.json"));
  await writeJSON(path.join(config.out, "manifest.json"), { schema: "rewrite-stage-run-v1", files: archived });
  console.log(`${status.status}: ${path.join(config.out, "status.json")}`);
  if (status.status !== "completed") {
    console.error(status.error || "Native run failed or did not write a complete event; retain this attempt.");
    process.exitCode = 1;
  }
}

try {
  if (operation === "run") await run();
  else if (operation === "report") execute(python, [path.join(study, "report.py"), ...arguments_]);
  else if (operation === "verify") {
    if (arguments_.length !== 1) throw new Error(`verify requires a directory\n${usage}`);
    execute(python, [path.join(study, "report.py"), "--verify", path.resolve(root, arguments_[0])]);
  } else if (operation === "test") {
    if (arguments_.length) throw new Error("test does not accept arguments");
    execute(python, ["-m", "unittest", "discover", "-s", study, "-p", "test_report.py", "-v"]);
  } else if (operation === "--help" || operation === "help") console.log(usage);
  else throw new Error(usage);
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
