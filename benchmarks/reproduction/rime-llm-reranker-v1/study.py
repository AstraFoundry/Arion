"""Native-candidate collection, real Lua/HTTP execution, and independent scoring."""
import argparse
from collections import Counter, defaultdict
import hashlib
import json
import os
from pathlib import Path
import platform
import random
import shutil
import subprocess
import sys
import time

ROOT = Path(__file__).resolve().parents[3]
STUDY = Path(__file__).resolve().parent
CORPUS = ROOT / "benchmarks/corpora/rime-llm-reranker-v1/corpus.json"
MODEL = "Qwen3.5-0.8B-MLX-bf16"


def read(file):
    return json.loads(Path(file).read_text())


def save(file, value):
    Path(file).write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n")


def digest(file):
    result = hashlib.sha256()
    with Path(file).open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            result.update(block)
    return result.hexdigest()


def command(args):
    result = subprocess.run(args, text=True, capture_output=True)
    return {"args": args, "exitCode": result.returncode, "stdout": result.stdout, "stderr": result.stderr}


def metadata(workspace):
    return {"platform": platform.platform(), "machine": platform.machine(), "timer": "CLOCK_MONOTONIC; Python perf_counter",
            "hardware": command(["sysctl", "-n", "machdep.cpu.brand_string", "hw.memsize", "hw.ncpu"]),
            "power": command(["pmset", "-g", "batt"]), "powerSettings": command(["pmset", "-g", "custom"]),
            "thermal": command(["pmset", "-g", "therm"]), "git": command(["git", "rev-parse", "HEAD"]),
            "dirty": command(["git", "status", "--short"]), "loadAverage": list(os.getloadavg()),
            "model": read(workspace / "model-manifest.json") if (workspace / "model-manifest.json").exists() else None,
            "omlxRevision": command(["git", "-C", str(workspace / "omlx"), "rev-parse", "HEAD"]),
            "packages": command(["uv", "pip", "freeze", "--python", str(workspace / "venv/bin/python")]),
            "cpuIsolated": False, "concurrentASR": False}


def collect(workspace, out):
    out.mkdir(parents=True, exist_ok=False)
    for name in ["requests", "raw", "profiles"]:
        (out / name).mkdir()
    save(out / "status.json", {"status": "running"})
    binary = workspace / "native-build/driver"
    runtime = ROOT / "apps/desktop/resources/darwin/keyboard"
    meta = metadata(workspace)
    meta.update({"corpusSha256": digest(CORPUS), "driverSha256": digest(binary),
                 "runtimeManifestSha256": digest(runtime / "runtime.json"),
                 "librimeSha256": digest(runtime / "lib/librime.1.dylib"),
                 "scope": "Fresh production Rime engine/profile per fixture; reference-free input; no OS event dispatch"})
    save(out / "metadata.json", meta)
    try:
        for case in read(CORPUS)["cases"]:
            request = {"id": case["id"], "input": case["input"], "warmups": ["xuetang", "heishan"]}
            source, target = out / "requests" / (case["id"] + ".json"), out / "raw" / (case["id"] + ".json")
            save(source, request)
            profile = out / "profiles" / case["id"]
            result = command([str(binary), str(runtime / "data"), str(profile), str(source), str(target)])
            save(out / "raw" / (case["id"] + ".process.json"), result)
            if result["exitCode"] or read(target)["status"] != "completed":
                raise RuntimeError(f"Native candidate collection failed: {case['id']}")
            shutil.rmtree(profile)
        save(out / "status.json", {"status": "completed", "cases": len(read(CORPUS)["cases"])})
    except Exception as error:
        save(out / "status.json", {"status": "failed", "error": str(error)})
        raise


def settings(endpoint, timeout):
    # Match the supplied upstream YAML, including its exact system prompt.
    return {"enabled": True, "endpoint": endpoint, "model": MODEL, "max_candidates": 5,
            "timeout": timeout, "min_syllables": 1, "max_rerank_pos": 5, "max_tokens": 20,
            "min_context_len": 0, "max_preedit_len": 10, "debug_log": True,
            "fallback_on_error": True, "collect_training_data": True,
            "collect_user_pairs": True, "collect_llm_pairs": False,
            "system_prompt": "你是输入法候选词排序助手。根据对话历史和当前输入，从候选词列表中选出最符合语境的词。按优先级从高到低列出最多5个候选词，用顿号分隔，不要拼音、数字序号或任何解释。"}


def execute(workspace, candidates, out, endpoint, timeout, mode):
    if read(candidates / "status.json")["status"] != "completed":
        raise ValueError("Incomplete candidate collection")
    out.mkdir(parents=True, exist_ok=False)
    (out / "requests").mkdir()
    (out / "raw").mkdir()
    home = out / "isolated-home"
    (home / "Library/Rime").mkdir(parents=True)
    env = {**os.environ, "HOME": str(home),
           "LUA_PATH": str(workspace / "lua/share/lua/5.4/?.lua") + ";;",
           "LUA_CPATH": str(workspace / "lua/lib/lua/5.4/?.so") + ";;"}
    args = [str(workspace / "sources/lua-5.4.8/src/lua"), str(STUDY / "filter-driver.lua"), str(STUDY / "upstream/rime.lua")]
    cases = read(CORPUS)["cases"]
    if mode == "typing":
        cases = [c for c in cases if c["group"] == "context"][:16] + [c for c in cases if c["group"] in {"abbreviation", "partial", "long-input"}][:24:4]
    random.Random(12701).shuffle(cases)
    meta = metadata(workspace)
    meta.update({"corpusSha256": digest(CORPUS), "candidateMetadataSha256": digest(candidates / "metadata.json"),
                 "upstreamSha256": digest(STUDY / "upstream/rime.lua"), "driverSha256": digest(STUDY / "filter-driver.lua"),
                 "config": settings(endpoint, timeout), "mode": mode, "args": args,
                 "warmPolicy": "Server resident after separately retained warmup; Lua state resets per fixture, repeated snapshot measures Lua cache",
                 "boundary": "Original Lua filter + real local HTTP/model; recorded native candidate snapshots, not installed Rime/IMK dispatch"})
    save(out / "metadata.json", meta)
    save(out / "status.json", {"status": "running"})
    index = []
    try:
        for case in cases:
            native = read(candidates / "raw" / (case["id"] + ".json"))
            snapshots = [row["snapshot"] for row in native["steps"]] if mode == "typing" else [native["beforeConfirmation"]]
            steps = [{"id": f"key-{i + 1}", "preedit": s["preedit"], "candidates": s["candidates"]} for i, s in enumerate(snapshots)]
            last = steps[-1]
            steps.append({**last, "id": "identical-repeat"})
            if mode == "final" and case["id"].startswith("context-") and int(case["id"].split("-")[1]) <= 8:
                steps.append({**last, "id": "changed-candidate-order", "candidates": list(reversed(last["candidates"]))})
            request = {"id": case["id"], "config": settings(endpoint, timeout), "history": case["context"], "steps": steps}
            # Gold answers and even their field names stay outside the Lua driver.
            assert set(request) == {"id", "config", "history", "steps"}
            save(out / "requests" / (case["id"] + ".json"), request)
            start = time.perf_counter()
            result = subprocess.run(args, input=json.dumps(request, ensure_ascii=False) + "\n", text=True,
                                    capture_output=True, env=env, timeout=max(60, timeout * len(steps) + 15))
            raw = out / "raw" / (case["id"] + ".json")
            if result.returncode:
                save(raw, {"status": "failed", "stdout": result.stdout, "stderr": result.stderr})
                raise RuntimeError(f"Lua failed: {case['id']}: {result.stderr}")
            response = json.loads(result.stdout)
            if response["id"] != case["id"] or any(row["status"] != "completed" for row in response["steps"]):
                raise RuntimeError("Incomplete filter response")
            response["subprocessWallMs"] = (time.perf_counter() - start) * 1000
            response["stderr"] = result.stderr
            save(raw, response)
            index.append({"id": case["id"], "requestSha256": digest(out / "requests" / (case["id"] + ".json")), "responseSha256": digest(raw)})
            save(out / "observations.json", index)
            if len(index) % 10 == 0:
                print(f"Completed {len(index)}/{len(cases)} {mode} fixtures", flush=True)
        save(out / "status.json", {"status": "completed", "cases": len(cases), "mode": mode})
    except Exception as error:
        save(out / "status.json", {"status": "failed", "error": str(error), "completed": len(index)})
        raise


def warmup(workspace, candidates, out, endpoint):
    """Retain unrelated cancelled producer warmups, including cold model load."""
    out.mkdir(parents=True, exist_ok=False)
    home = out / "isolated-home"
    (home / "Library/Rime").mkdir(parents=True)
    native = read(candidates / "raw/context-001.json")
    steps = [{"id": f"warmup-{i + 1}", "preedit": item["steps"][-1]["snapshot"]["preedit"],
              "candidates": item["steps"][-1]["snapshot"]["candidates"]} for i, item in enumerate(native["warmups"])]
    request = {"id": "unrelated-warmups", "history": "", "config": settings(endpoint, 60), "steps": steps}
    save(out / "request.json", request)
    env = {**os.environ, "HOME": str(home), "LUA_PATH": str(workspace / "lua/share/lua/5.4/?.lua") + ";;",
           "LUA_CPATH": str(workspace / "lua/lib/lua/5.4/?.so") + ";;"}
    args = [str(workspace / "sources/lua-5.4.8/src/lua"), str(STUDY / "filter-driver.lua"), str(STUDY / "upstream/rime.lua")]
    result = subprocess.run(args, input=json.dumps(request, ensure_ascii=False) + "\n", text=True, capture_output=True, env=env, timeout=150)
    save(out / "process.json", {"exitCode": result.returncode, "stdout": result.stdout, "stderr": result.stderr})
    if result.returncode:
        raise RuntimeError("Warmup failed; process result retained")
    save(out / "response.json", json.loads(result.stdout))


def percentile(values, p):
    if not values:
        return None
    values = sorted(values)
    index = (len(values) - 1) * p
    low = int(index)
    return values[low] + (values[min(low + 1, len(values) - 1)] - values[low]) * (index - low)


def timings(values):
    return {"n": len(values), "p50": percentile(values, .5), "p95": percentile(values, .95),
            "p99": percentile(values, .99), "max": max(values) if values else None}


def validate_row(row):
    source, identities = row["source"], row["identities"]
    if sorted(identities) != list(range(1, len(source) + 1)):
        raise ValueError("Rank output is not an exact candidate-ID permutation")
    if row["ranked"] != [source[i - 1] for i in identities]:
        raise ValueError("Rank output altered candidate text")


def evaluate(run):
    status = read(run / "status.json")
    if status["status"] != "completed":
        raise ValueError("Refuse incomplete run")
    corpus = read(CORPUS)
    meta = read(run / "metadata.json")
    if meta["corpusSha256"] != digest(CORPUS):
        raise ValueError("Corpus hash mismatch")
    labels = {c["id"]: c for c in corpus["cases"]}
    index = read(run / "observations.json")
    expected = len(corpus["cases"]) if meta["mode"] == "final" else status["cases"]
    if len(index) != expected or len({c["id"] for c in index}) != expected:
        raise ValueError("Missing or duplicate observations")
    counts, latency, cache_latency, query_latency, cpu, transport = Counter(), [], [], [], [], Counter()
    groups, cases, changes, cache_controls = defaultdict(Counter), [], [], []
    query_records = []
    for entry in index:
        name = entry["id"] + ".json"
        request_file, response_file = run / "requests" / name, run / "raw" / name
        if digest(request_file) != entry["requestSha256"] or digest(response_file) != entry["responseSha256"]:
            raise ValueError("Observation hash mismatch")
        request = read(request_file)
        if set(request) != {"id", "config", "history", "steps"} or any(set(s) - {"id", "preedit", "candidates", "history"} for s in request["steps"]):
            raise ValueError("Unexpected driver fields / evaluator label leak")
        if request["id"] != entry["id"] or request["history"] != labels[entry["id"]].get("context", ""):
            raise ValueError("Fixture identity/context mismatch")
        if "config" in meta and request["config"] != meta["config"]:
            raise ValueError("Treatment changed during run")
        rows = read(response_file)["steps"]
        if len(rows) != len(request["steps"]):
            raise ValueError("Missing filter steps")
        for original, row in zip(request["steps"], rows):
            if row["source"] != original["candidates"] or row["preedit"] != original["preedit"] or row["status"] != "completed":
                raise ValueError("Candidate/input mismatch")
            validate_row(row)
            if row["id"] == "identical-repeat":
                cache_latency.append(row["wallMs"])
                counts["repeatHttpRequests"] += len(row["requests"])
                continue
            if row["id"] == "changed-candidate-order":
                cache_controls.append({"id": entry["id"], "httpRequests": len(row["requests"]), "source": row["source"], "ranked": row["ranked"], "identities": row["identities"]})
                continue
            latency.append(row["wallMs"])
            cpu.append(row["processCpuMs"])
            counts["filterCalls"] += 1
            counts["httpCalls"] += len(row["requests"])
            if not row["requests"]:
                counts["noHttpCalls"] += 1
            for http in row["requests"]:
                query_latency.append(http["networkWallMs"])
                transport[str(http["code"])] += 1
                parsed = None
                if http["response"]:
                    try:
                        parsed = json.loads(http["response"])
                    except json.JSONDecodeError:
                        counts["invalidResponseJson"] += 1
                if parsed and parsed.get("choices"):
                    choice = parsed["choices"][0]
                    counts["emptyContent"] += not bool(choice["message"].get("content"))
                    counts["tokenLimitResponses"] += choice.get("finish_reason") == "length"
                query_records.append({"id": entry["id"], "step": row["id"], "wallMs": http["networkWallMs"], "response": parsed})
        measured = [r for r in rows if r["id"].startswith("key-")][-1]
        acceptable = labels[entry["id"]]["acceptable"]
        source, ranked = measured["source"], measured["ranked"]
        base = bool(source and source[0] in acceptable)
        treatment = bool(ranked and ranked[0] in acceptable)
        rank_before = next((i + 1 for i, t in enumerate(source) if t in acceptable), None)
        rank_after = next((i + 1 for i, t in enumerate(ranked) if t in acceptable), None)
        result = {"id": entry["id"], "group": labels[entry["id"]]["group"], "acceptable": acceptable,
                  "source": source, "ranked": ranked, "preedit": measured["preedit"],
                  "baselineCorrect": base, "rerankedCorrect": treatment, "rankBefore": rank_before,
                  "rankAfter": rank_after, "wallMs": measured["wallMs"], "httpCalls": len(measured["requests"])}
        cases.append(result)
        values = {"n": 1, "baselineTop1": int(base), "rerankedTop1": int(treatment), "oracle9": int(rank_before is not None),
                  "oracle5": int(rank_before is not None and rank_before <= 5), "rescues": int(not base and treatment),
                  "breakages": int(base and not treatment), "changed": int(source != ranked),
                  "baselineRR": 1 / rank_before if rank_before else 0, "rerankedRR": 1 / rank_after if rank_after else 0}
        counts.update(values)
        groups[result["group"]].update(values)
        if source != ranked:
            changes.append(result)
    n = counts["n"]
    return {"status": "completed", "mode": meta["mode"], "counts": dict(counts), "groups": dict(groups),
            "baselineTop1": counts["baselineTop1"] / n, "rerankedTop1": counts["rerankedTop1"] / n,
            "baselineMRR9": counts["baselineRR"] / n, "rerankedMRR9": counts["rerankedRR"] / n,
            "latencyMs": {"filter": timings(latency), "http": timings(query_latency), "processCpu": timings(cpu), "identicalLuaCache": timings(cache_latency)},
            "httpStatus": dict(transport), "cacheControls": cache_controls, "changes": changes,
            "cases": sorted(cases, key=lambda c: c["id"]), "queryRecords": query_records,
            "limitations": ["Single assistant-authored exposed exploratory corpus, no independent annotation",
                            "Recorded native producer candidates plus original Lua API adapter; no installed IMK/UI dispatch",
                            "Resident server and independently reset Lua state, no OS cache eviction",
                            "No concurrent ASR; host not power/CPU isolated; invalid/timeouts retain upstream order and remain in denominator"]}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("operation", choices=["collect", "run", "report", "warmup"])
    parser.add_argument("--workspace", type=Path)
    parser.add_argument("--candidates", type=Path)
    parser.add_argument("--run", type=Path)
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--endpoint", default="http://127.0.0.1:18782/v1/chat/completions")
    parser.add_argument("--timeout", type=float, default=.5)
    parser.add_argument("--mode", choices=["final", "typing"], default="final")
    args = parser.parse_args()
    if args.operation == "collect":
        collect(args.workspace.resolve(), args.out.resolve())
    elif args.operation == "run":
        execute(args.workspace.resolve(), args.candidates.resolve(), args.out.resolve(), args.endpoint, args.timeout, args.mode)
    elif args.operation == "warmup":
        warmup(args.workspace.resolve(), args.candidates.resolve(), args.out.resolve(), args.endpoint)
    else:
        report = evaluate(args.run.resolve())
        with args.out.open("x") as stream:
            json.dump(report, stream, ensure_ascii=False, indent=2)
            stream.write("\n")
        print(json.dumps({k: v for k, v in report.items() if k not in {"queryRecords", "cases", "changes", "cacheControls"}}, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
