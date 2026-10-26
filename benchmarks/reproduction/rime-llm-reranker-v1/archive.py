"""Publish completed observations and explicit failed attempts, excluding runtimes."""
import argparse
import csv
import json
from pathlib import Path
import shutil
import study

parser = argparse.ArgumentParser()
parser.add_argument("--workspace", required=True, type=Path)
args = parser.parse_args()
workspace = args.workspace.resolve()
run = study.ROOT / "benchmarks/runs/rime-llm-reranker-v1"
results = study.ROOT / "benchmarks/results/rime-llm-reranker-v1"
run.mkdir(parents=True, exist_ok=False)
results.mkdir(parents=True, exist_ok=False)
mappings = []


def copy(source, target):
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, target)
    mappings.append({"original": str(source), "archived": str(target.relative_to(study.ROOT)),
                     "sha256": study.digest(target), "bytes": target.stat().st_size})


for name in ["candidates", "final-stock", "typing-stock", "warmup-stock", "cache-controls"]:
    for source in sorted((workspace / name).rglob("*")):
        if source.is_file() and "profiles" not in source.parts:
            copy(source, run / name / source.relative_to(workspace / name))
for name in ["model-api.json", "model-blobs.json", "model-manifest.json", "acquire.log", "collect.log",
             "final-stock.log", "typing-stock.log", "warmup-stock.log", "cache-controls.log",
             "server-stock.log", "server-no-model-attempt.log", "native-build.log", "report-tests-final.log"]:
    copy(workspace / name, run / "preparation" / name)
for name in ["compile.log", "status.json", "source-identities.json"]:
    copy(workspace / "native-build" / name, run / "preparation/native-initial-failed" / name)
for source in (workspace / "native-build/sources").rglob("*"):
    if source.is_file():
        copy(source, run / "preparation/native-sources" / source.relative_to(workspace / "native-build/sources"))
for name in ["ranking.h", "ranking.m"]:
    source = study.ROOT / "apps/desktop/ime/macos" / name
    copy(source, run / "preparation/native-sources/apps/desktop/ime/macos" / name)
for name in ["settings.json", "model_settings.json"]:
    source = workspace / "server-stock" / name
    if name == "settings.json":
        original = study.read(source)
        allowed = ["version", "server", "model", "memory", "scheduler", "cache", "sampling", "idle_timeout"]
        study.save(run / "preparation/server-settings-public.json", {"originalPrivateSha256": study.digest(source),
                   "omitted": "Authentication secrets and unrelated integration/network settings are excluded",
                   "settings": {k: original[k] for k in allowed}})
    else:
        copy(source, run / "preparation" / name)
study.save(run / "preparation/artifact-identities.json", {
    "luaExecutableSha256": study.digest(workspace / "sources/lua-5.4.8/src/lua"),
    "luaSourceArchiveSha256": study.digest(workspace / "sources/lua.tar.gz"),
    "luaSocketSha256": study.digest(workspace / "lua/lib/lua/5.4/socket/core.so"),
    "monotonicClockSha256": study.digest(workspace / "lua/lib/lua/5.4/benchclock.so"),
    "nativeExecutableSha256": study.digest(workspace / "native-build/driver"),
    "nativeLinkRepair": "Initial historical quality runner omitted ranking.m. Current production ranking.m was compiled with the same -O2/-fobjc-arc/arm64-apple-macos14.0 flags and linked alongside its existing objects. No production source was edited.",
    "runtimeSources": {"lua": "https://www.lua.org/ftp/lua-5.4.8.tar.gz", "luasocket": "95b7efa9da506ef968c1347edf3fc56370f0deed", "dkjson": "e72ba0c9f5d8b8746fc306f6189a819dbb5cd0be", "omlx": "6aba0dd3f28c690230a572a0c96cb43ee4f3d738"},
    "nativeDriverSource": "benchmarks/reproduction/imk-quality-v1/driver.mm",
    "scope": "Binary identities only; compiled executables, model weights, environments, GPU caches and synthetic encrypted profiles excluded"})
copy(workspace / "parser-audit.json", results / "parser-audit.json")
for mode in ["stock", "typing"]:
    name = "final-stock" if mode == "stock" else "typing-stock"
    report = study.evaluate(run / name)
    original = study.read(workspace / ("report-stock.json" if mode == "stock" else "report-typing.json"))
    if report != original:
        raise ValueError("Archived report replay differs")
    study.save(results / ("summary.json" if mode == "stock" else "typing-summary.json"), report)

report = study.read(results / "summary.json")
corpus = {c["id"]: c for c in study.read(study.CORPUS)["cases"]}
with (results / "cases.csv").open("x", newline="") as stream:
    fields = ["id", "group", "context", "input", "acceptable", "baseline", "reranked", "baselineCorrect", "rerankedCorrect", "rankBefore", "rankAfter", "wallMs", "httpCalls"]
    writer = csv.DictWriter(stream, fieldnames=fields)
    writer.writeheader()
    for case in report["cases"]:
        original = corpus[case["id"]]
        writer.writerow({**{k: case[k] for k in ["id", "group", "baselineCorrect", "rerankedCorrect", "rankBefore", "rankAfter", "wallMs", "httpCalls"]},
                         "context": original["context"], "input": original["input"], "acceptable": "/".join(case["acceptable"]),
                         "baseline": case["source"][0], "reranked": case["ranked"][0]})
study.save(run / "path-mapping.json", mappings)
study.save(run / "status.json", {"status": "completed", "nativeFixtures": 100, "finalFixtures": 100,
                                  "typingFixtures": 22, "typingSnapshots": 142, "realModelRequests": 219,
                                  "upstreamSourceUnchanged": True, "productionIntegration": False})
print(f"Archived {len(mappings)} original files; model-free report replay passed")
