"""Real-inference cache controls using only recorded producer candidates."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import study

parser = argparse.ArgumentParser()
parser.add_argument("--workspace", required=True, type=Path)
parser.add_argument("--candidates", required=True, type=Path)
parser.add_argument("--out", required=True, type=Path)
parser.add_argument("--endpoint", default="http://127.0.0.1:18782/v1/chat/completions")
args = parser.parse_args()
workspace, out = args.workspace.resolve(), args.out.resolve()
out.mkdir(parents=True, exist_ok=False)
home = out / "isolated-home"
(home / "Library/Rime").mkdir(parents=True)
env = {**os.environ, "HOME": str(home), "LUA_PATH": str(workspace / "lua/share/lua/5.4/?.lua") + ";;",
       "LUA_CPATH": str(workspace / "lua/lib/lua/5.4/?.so") + ";;"}
cases = {c["id"]: c for c in study.read(study.CORPUS)["cases"]}
results = []
for name in ["context-023", "context-034", "context-048", "common-007"]:
    native = study.read(args.candidates / "raw" / (name + ".json"))["beforeConfirmation"]
    source = native["candidates"]
    # Rotate the same first five candidates; membership stays identical.
    rotated = source[1:5] + source[:1] + source[5:]
    request = {"id": name, "config": study.settings(args.endpoint, .5), "history": cases[name]["context"],
               "steps": [{"id": "original", "preedit": native["preedit"], "candidates": source},
                         {"id": "rotated-same-members", "preedit": native["preedit"], "candidates": rotated}]}
    study.save(out / (name + ".request.json"), request)
    result = subprocess.run([str(workspace / "sources/lua-5.4.8/src/lua"), str(study.STUDY / "filter-driver.lua"),
                             str(study.STUDY / "upstream/rime.lua")], input=json.dumps(request, ensure_ascii=False) + "\n",
                            env=env, text=True, capture_output=True, check=True, timeout=30)
    response = json.loads(result.stdout)
    study.save(out / (name + ".response.json"), response)
    first, second = response["steps"]
    for row in response["steps"]:
        study.validate_row(row)
    results.append({"id": name, "originalSource": source, "rotatedSource": rotated,
                    "originalFirst": first["ranked"][0], "afterRotationFirst": second["ranked"][0],
                    "firstRequests": len(first["requests"]), "secondRequests": len(second["requests"]),
                    "stalePositionChangesChosenText": first["ranked"][0] != second["ranked"][0]})
study.save(out / "summary.json", {"status": "completed", "kind": "Real model query followed by candidate-order perturbation; no new linguistic quality samples", "cases": results})
print(json.dumps(results, ensure_ascii=False, indent=2))
