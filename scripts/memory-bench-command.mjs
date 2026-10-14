// Entry point for the edit-learning and vocabulary-recall benchmark
// (ADR 014). It runs on any OS; the optional replay reads private
// recognition results and the corpus from .workspace.
import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
// Make exports unset optional variables as empty strings.
const option = (key) => (process.env[key] === "" ? undefined : process.env[key]);
if (!option("OUT")) throw new Error("OUT required");
const absolute = (value) => path.resolve(root, value);
const args = ["run", "./cmd/arion-memory-bench", "-out", absolute(option("OUT")),
  "-cases", absolute(option("CASES") ?? "benchmarks/configs/edit-learning-cases.json")];
if (option("CORPUS")) {
  for (const key of ["DEV", "TEST"]) if (!option(key)) throw new Error(`${key} required with CORPUS`);
  args.push("-corpus", absolute(option("CORPUS")), "-dev", absolute(option("DEV")), "-test", absolute(option("TEST")));
}
if (option("REPS")) args.push("-reps", option("REPS"));
const result = spawnSync("go", args, { cwd: path.join(root, "apps/desktop"), stdio: "inherit", shell: false,
  env: { ...process.env, CGO_ENABLED: "0" } });
if (result.error) throw result.error;
process.exit(result.status ?? 1);
