// Actual production Go correction/recall observations on controlled fixtures.
// Evaluator labels remain separate and are never passed to the Go driver.
import { createHash } from "node:crypto";
import { spawn, spawnSync } from "node:child_process";
import { createWriteStream } from "node:fs";
import * as fs from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { finished } from "node:stream/promises";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const study = "benchmarks/reproduction/vocabulary-alignment-v1";
const usage = "vocabulary-bench-command.mjs run --out NEWDIR | report --run RUNDIR --out NEWDIR | verify DIRECTORY";
const digest = bytes => createHash("sha256").update(bytes).digest("hex");
const hashFile = async file => digest(await fs.readFile(file));
const json = async file => JSON.parse(await fs.readFile(file, "utf8"));
const writeJSON = async (file, value) => fs.writeFile(file, `${JSON.stringify(value, null, 2)}\n`, { flag: "wx" });
const equal = (a, b) => JSON.stringify(a) === JSON.stringify(b);

function probe(command, args) {
  const result = spawnSync(command, args, { cwd: root, encoding: "utf8", shell: false });
  return { status: result.status, stdout: result.stdout ?? "", stderr: result.stderr ?? "", error: result.error?.message ?? "" };
}
function git(args) {
  const result = probe("git", args);
  if (result.status !== 0) throw new Error(`Git failed: ${result.stderr || result.error}`);
  return result.stdout;
}
function options(args, required) {
  const values = {};
  for (let i = 0; i < args.length; i += 2) {
    const key = args[i].replace(/^--/, "");
    if (!args[i].startsWith("--") || !required.includes(key) || key in values || !args[i + 1]) throw new Error(usage);
    values[key] = path.resolve(root, args[i + 1]);
  }
  if (required.some(k => !values[k])) throw new Error(usage);
  return values;
}
async function filesUnder(directory) {
  const files = [];
  for (const item of await fs.readdir(directory, { withFileTypes: true })) {
    const file = path.join(directory, item.name);
    if (item.isDirectory()) files.push(...await filesUnder(file));
    else if (item.isFile()) files.push(file);
    else throw new Error(`Unsupported archive entry: ${file}`);
  }
  return files.sort();
}
async function seal(directory, schema, extra = {}) {
  const files = {};
  for (const file of await filesUnder(directory)) files[path.relative(directory, file).split(path.sep).join("/")] = await hashFile(file);
  await writeJSON(path.join(directory, "manifest.json"), { schema, ...extra, files });
}
async function verifyInventory(directory) {
  const manifest = await json(path.join(directory, "manifest.json"));
  if (!["vocabulary-recall-run-v1", "vocabulary-recall-report-v1"].includes(manifest.schema)) throw new Error("Unsupported archive schema");
  const actual = (await filesUnder(directory)).map(f => path.relative(directory, f).split(path.sep).join("/")).filter(f => f !== "manifest.json");
  if (!equal(actual.sort(), Object.keys(manifest.files).sort())) throw new Error("Archive inventory differs from manifest");
  for (const [relative, expected] of Object.entries(manifest.files)) {
    if (!/^[a-f0-9]{64}$/.test(expected) || relative.startsWith("/") || relative.split("/").includes("..")) throw new Error("Invalid archive hash/path");
    if (await hashFile(path.join(directory, relative)) !== expected) throw new Error(`SHA-256 mismatch: ${relative}`);
  }
  return manifest;
}
async function readRows(file) {
  const text = await fs.readFile(file, "utf8");
  if (!text.endsWith("\n")) throw new Error(`Truncated JSONL: ${file}`);
  return text.trim().split("\n").filter(Boolean).map(line => JSON.parse(line));
}
function fixturesValid(inputs, expected) {
  if (!Array.isArray(inputs) || !inputs.length || !Array.isArray(expected) || expected.length !== inputs.length) throw new Error("Incomplete fixture/label arrays");
  const ids = new Set();
  for (const c of inputs) {
    if (typeof c.id !== "string" || !c.id || ids.has(c.id) || typeof c.source !== "string" || typeof c.previous !== "string" || !Array.isArray(c.entries) || c.entries.some(e => typeof e.From !== "string" || typeof e.To !== "string")) throw new Error("Invalid input fixture");
    ids.add(c.id);
  }
  const labels = new Set();
  for (const c of expected) {
    if (!ids.has(c.id) || labels.has(c.id) || typeof c.corrected !== "string" || !Array.isArray(c.terms) || c.terms.some(t => typeof t !== "string")) throw new Error("Invalid evaluator label");
    labels.add(c.id);
  }
}
export function percentile(values, p) {
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.max(0, Math.ceil(p * sorted.length) - 1)];
}
const latency = ns => ({ p50: percentile(ns, .5) / 1000, p95: percentile(ns, .95) / 1000, p99: percentile(ns, .99) / 1000 });
const target = term => term.includes(" → ") ? term.split(" → ").at(-1) : term;

export function evaluate(inputs, expected, arms) {
  fixturesValid(inputs, expected);
  const byID = new Map(expected.map(c => [c.id, c]));
  const summary = {};
  const cases = {};
  for (const arm of ["baseline", "candidate"]) {
    const rows = arms[arm];
    if (!Array.isArray(rows) || rows.length !== inputs.length || new Set(rows.map(r => r.id)).size !== rows.length) throw new Error(`${arm}: incomplete or duplicate observations`);
    const totals = { cases: rows.length, exactCorrected: 0, exactTerms: 0, canonicalExactTerms: 0, expectedCanonicalTerms: 0,
      recalledExpectedCanonicalTerms: 0, unexpectedCanonicalTerms: 0, aliasExposingTermSlots: 0, duplicateCanonicalBudgetSlots: 0,
      returnedTermSlots: 0, maxTermSlots: 0, budgetViolations: 0 };
    const times = [];
    cases[arm] = inputs.map((input, i) => {
      const row = rows[i];
      if (row.id !== input.id || row.source !== input.source || typeof row.corrected !== "string" || !(row.terms === null || Array.isArray(row.terms)) || (row.terms ?? []).some(t => typeof t !== "string") || !Array.isArray(row.latencyNs) || row.latencyNs.length !== 100 || row.latencyNs.some(n => !Number.isSafeInteger(n) || n < 0)) throw new Error(`${arm}: invalid or reordered observation ${input.id}`);
      const label = byID.get(row.id);
      const terms = row.terms ?? [];
      const canonical = terms.map(target);
      const uniqueCanonical = [...new Set(canonical)];
      const detail = { id: row.id, corrected: row.corrected, terms, expected: label,
        correctedExact: row.corrected === label.corrected, termsExact: equal(terms, label.terms),
        uniqueCanonical, canonicalTermsExact: equal(uniqueCanonical, label.terms),
        expectedCanonicalRecalled: label.terms.filter(t => uniqueCanonical.includes(t)).length,
        unexpectedCanonical: uniqueCanonical.filter(t => !label.terms.includes(t)),
        aliasExposingTermSlots: terms.filter(t => t.includes(" → ")).length,
        duplicateCanonicalBudgetSlots: canonical.length - uniqueCanonical.length,
        latencyUs: latency(row.latencyNs) };
      totals.exactCorrected += Number(detail.correctedExact);
      totals.exactTerms += Number(detail.termsExact);
      totals.canonicalExactTerms += Number(detail.canonicalTermsExact);
      totals.expectedCanonicalTerms += label.terms.length;
      totals.recalledExpectedCanonicalTerms += detail.expectedCanonicalRecalled;
      totals.unexpectedCanonicalTerms += detail.unexpectedCanonical.length;
      totals.aliasExposingTermSlots += detail.aliasExposingTermSlots;
      totals.duplicateCanonicalBudgetSlots += detail.duplicateCanonicalBudgetSlots;
      totals.returnedTermSlots += terms.length;
      totals.maxTermSlots = Math.max(totals.maxTermSlots, terms.length);
      totals.budgetViolations += Number(terms.length > 8);
      times.push(...row.latencyNs);
      return detail;
    });
    summary[arm] = { ...totals, canonicalRecall: totals.expectedCanonicalTerms ? totals.recalledExpectedCanonicalTerms / totals.expectedCanonicalTerms : null,
      latencyUs: latency(times), largeVocabularyLatencyUs: cases[arm].find(c => c.id === "large-2000")?.latencyUs ?? null };
  }
  return { schema: "vocabulary-recall-evaluation-v1", execution: "Actual production Go deterministic correction/recall; no ASR, OCR or LLM inference",
    population: "Assistant-authored controlled synthetic acceptance fixtures; not held-out speech or model quality",
    timing: "100 warmed repetitions per case of Correct plus TermsForSources; index construction, compilation and process startup excluded; Go monotonic time.Now/time.Since; baseline runs before candidate",
    baseline: "git HEAD vocabulary.go with frozen historical application termsForFinal adapter; policy static Terms empty; raw From → To terms retained",
    candidateAcceptancePassed: summary.candidate.exactCorrected === inputs.length && summary.candidate.exactTerms === inputs.length && summary.candidate.budgetViolations === 0,
    metrics: summary, cases };
}
async function replay(directory) {
  return evaluate(await json(path.join(directory, "inputs.json")), await json(path.join(directory, "expected.json")), {
    baseline: await readRows(path.join(directory, "baseline.jsonl")), candidate: await readRows(path.join(directory, "candidate.jsonl")),
  });
}
function nativeCases(inputs, rows) {
  const selected = new Set(["alias-latin", "alias-second", "alias-cjk", "canonical-already-correct", "unrelated", "previous-alias", "cross-source-dedup", "large-2000"]);
  const sources = new Map(inputs.map(c => [c.id, c]));
  return rows.filter(c => selected.has(c.id)).map(c => ({ id: `vocabulary-${c.id}`, scene: "chat", preview: c.corrected, final: c.corrected,
    screen: "", terms: c.terms ?? [], previousText: sources.get(c.id).previous, pairKind: "controlled-vocabulary-production-recall-observation",
    sourceEvidence: { inputID: c.id, observationFile: "candidate.jsonl", observationSha256: digest(JSON.stringify(c)),
      limitations: "Synthetic text input; preview and final both use observed production correction, terms use actual recall, prior uses original input. No evaluator labels used. No ASR or natural-speech evidence." } }));
}
async function executeGo(args, directory, arm, environment) {
  const stdout = createWriteStream(path.join(directory, `${arm}.jsonl`), { flags: "wx" });
  const stderr = createWriteStream(path.join(directory, `${arm}.stderr.log`), { flags: "wx" });
  const result = await new Promise(resolve => {
    const child = spawn("go", args, { cwd: root, env: environment, shell: false, stdio: ["ignore", "pipe", "pipe"] });
    child.stdout.pipe(stdout);
    child.stderr.pipe(stderr);
    child.on("error", error => resolve({ code: null, error: error.message }));
    child.on("close", (code, signal) => resolve({ code, signal }));
  });
  await Promise.all([finished(stdout), finished(stderr)]);
  return result;
}
async function run(out) {
  await fs.mkdir(path.dirname(out), { recursive: true });
  await fs.mkdir(out);
  await writeJSON(path.join(out, "attempt.json"), { status: "started", started: new Date().toISOString(), operation: "run" });
  let status;
  try {
    const inputs = await json(path.join(root, study, "inputs.json"));
    const expected = await json(path.join(root, study, "expected.json"));
    fixturesValid(inputs, expected);
    await fs.copyFile(path.join(root, study, "inputs.json"), path.join(out, "inputs.json"), 1);
    await fs.copyFile(path.join(root, study, "expected.json"), path.join(out, "expected.json"), 1);
    const productionDomain = (await filesUnder(path.join(root, "packages/core/domain"))).filter(f => f.endsWith(".go") && !f.endsWith("_test.go"));
    const sourceFiles = ["go.work", "packages/core/go.mod", "scripts/vocabulary-bench-command.mjs", `${study}/recall.go`, `${study}/historical_terms_adapter.go.txt`,
      ...productionDomain.map(f => path.relative(root, f))].sort();
    const sourceHashes = {};
    const overlay = {};
    for (const relative of sourceFiles) {
      const archived = path.join(out, "sources", relative);
      await fs.mkdir(path.dirname(archived), { recursive: true });
      await fs.copyFile(path.join(root, relative), archived, 1);
      sourceHashes[relative] = await hashFile(archived);
      overlay[path.join(root, relative)] = archived;
    }
    const oldVocabulary = git(["show", "HEAD:packages/core/domain/vocabulary.go"]);
    const oldApplication = git(["show", "HEAD:packages/core/application/live.go"]);
    const original = oldApplication.match(/func \(p RewritePolicy\) termsForFinal\(source, previous string\) \[\]string \{[\s\S]*?\n\}/)?.[0];
    if (!original) throw new Error("Historical termsForFinal source not found");
    await fs.mkdir(path.join(out, "baseline-source"));
    await fs.writeFile(path.join(out, "baseline-source/vocabulary.go"), oldVocabulary, { flag: "wx" });
    await fs.writeFile(path.join(out, "baseline-source/termsForFinal.go.txt"), `${original}\n`, { flag: "wx" });
    const historicalAdapter = path.join(out, "sources", study, "historical_terms_adapter.go.txt");
    const baselineOverlay = { ...overlay,
      [path.join(root, "packages/core/domain/vocabulary.go")]: path.join(out, "baseline-source/vocabulary.go"),
      [path.join(root, "packages/core/domain/vocabulary_historical_adapter.go")]: historicalAdapter };
    await writeJSON(path.join(out, "candidate-overlay.json"), { Replace: overlay });
    await writeJSON(path.join(out, "baseline-overlay.json"), { Replace: baselineOverlay });
    await writeJSON(path.join(out, "source-manifest.json"), { pathMapping: "sources/<repository-relative path>", files: sourceHashes,
      baseline: { revision: git(["rev-parse", "HEAD"]).trim(), vocabularySha256: digest(oldVocabulary), applicationFunctionSha256: digest(`${original}\n`),
        adapterSha256: await hashFile(historicalAdapter), adaptation: "No static policy Terms; termsForFinal copied with receiver replaced by v and fixed production cap asserted. SameVocabularySource compatibility export retains the historical index lowercase equivalence for unexecuted memory/edit paths. No evaluator labels enter adapter." } });
    const environment = { ...process.env, GOCACHE: path.join(os.tmpdir(), "arion-vocabulary-go-cache"), CGO_ENABLED: "0" };
    const provenance = { revision: git(["rev-parse", "HEAD"]).trim(), dirtyState: git(["status", "--porcelain=v1"]),
      sourceDiff: git(["diff", "HEAD", "--", ...sourceFiles]), sourceHashes,
      host: { platform: process.platform, arch: process.arch, release: os.release(), cpu: os.cpus(), totalMemoryBytes: os.totalmem() },
      power: process.platform === "darwin" ? probe("pmset", ["-g", "custom"]) : process.platform === "win32" ? probe("powercfg", ["/getactivescheme"]) : { unavailable: "No portable power-mode probe for this OS" },
      runtime: { go: probe("go", ["version"]), node: process.version, buildFlags: ["go run", "-overlay=<archived overlay>", "CGO_ENABLED=0"],
        timer: "Go time.Now/time.Since monotonic elapsed nanoseconds", GOCACHE: environment.GOCACHE },
      treatments: { armOrder: ["baseline", "candidate"], repetitions: 100, cap: 8, inference: false, sourceSnapshot: true }, commands: {} };
    let failed = false;
    for (const arm of ["baseline", "candidate"]) {
      const args = ["run", `-overlay=${path.join(out, `${arm}-overlay.json`)}`, path.join(out, "sources", study, "recall.go"), path.join(out, "inputs.json")];
      provenance.commands[arm] = ["go", ...args];
      const result = await executeGo(args, out, arm, environment);
      await writeJSON(path.join(out, `${arm}-status.json`), result);
      if (result.code !== 0) failed = true;
    }
    await writeJSON(path.join(out, "provenance.json"), provenance);
    if (failed) throw new Error("A Go arm failed; partial JSONL/stderr/status retained");
    const report = await replay(out);
    await writeJSON(path.join(out, "report.json"), report);
    const candidate = await readRows(path.join(out, "candidate.jsonl"));
    await writeJSON(path.join(out, "native-cases.json"), nativeCases(inputs, candidate));
    await writeJSON(path.join(out, "native-cases-provenance.json"), { inputSha256: await hashFile(path.join(out, "inputs.json")),
      observationsSha256: await hashFile(path.join(out, "candidate.jsonl")), policy: "Fixtures derived from actual candidate observations and original source.previous only; expected.json is excluded", cases: 8 });
    status = { status: "completed", completed: new Date().toISOString(), candidateAcceptancePassed: report.candidateAcceptancePassed, inference: false };
  } catch (error) {
    status = { status: "failed", failed: new Date().toISOString(), error: error.stack || error.message, inference: false };
  }
  await writeJSON(path.join(out, "status.json"), status);
  await seal(out, "vocabulary-recall-run-v1");
  console.log(JSON.stringify({ out, ...status }, null, 2));
  if (status.status !== "completed") process.exitCode = 1;
}
export async function verify(directory) {
  const manifest = await verifyInventory(directory);
  if (manifest.schema === "vocabulary-recall-report-v1") {
    const runDirectory = path.resolve(directory, manifest.sourceRun);
    await verify(runDirectory);
    if (await hashFile(path.join(runDirectory, "manifest.json")) !== manifest.sourceManifestSha256) throw new Error("Source run manifest changed");
    if (!equal(await replay(runDirectory), await json(path.join(directory, "report.json")))) throw new Error("Replayed report differs");
  } else {
    const status = await json(path.join(directory, "status.json"));
    if (!["completed", "failed"].includes(status.status)) throw new Error("Invalid run status");
    if (status.status === "completed") {
      const report = await replay(directory);
      if (!equal(report, await json(path.join(directory, "report.json")))) throw new Error("Replayed run report differs");
      const expectedCases = nativeCases(await json(path.join(directory, "inputs.json")), await readRows(path.join(directory, "candidate.jsonl")));
      if (!equal(expectedCases, await json(path.join(directory, "native-cases.json")))) throw new Error("Native cases differ from actual observations");
    }
  }
  return manifest;
}
async function report(runDirectory, out) {
  await verify(runDirectory);
  if ((await json(path.join(runDirectory, "status.json"))).status !== "completed") throw new Error("Incomplete run cannot produce a completed report");
  await fs.mkdir(path.dirname(out), { recursive: true });
  await fs.mkdir(out);
  await writeJSON(path.join(out, "report.json"), await replay(runDirectory));
  await seal(out, "vocabulary-recall-report-v1", { sourceRun: path.relative(out, runDirectory).split(path.sep).join("/"), sourceManifestSha256: await hashFile(path.join(runDirectory, "manifest.json")) });
  console.log(`Replayed report: ${out}`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const [operation, ...args] = process.argv.slice(2);
    if (operation === "run") await run(options(args, ["out"]).out);
    else if (operation === "report") { const config = options(args, ["run", "out"]); await report(config.run, config.out); }
    else if (operation === "verify" && args.length === 1) { await verify(path.resolve(root, args[0])); console.log("Archive hashes and report replay verified"); }
    else throw new Error(usage);
  } catch (error) { console.error(error.stack || error.message); process.exitCode = 1; }
}
