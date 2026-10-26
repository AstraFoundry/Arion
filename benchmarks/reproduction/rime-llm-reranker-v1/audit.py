"""Diagnose original parsing against frozen live responses; no new model calls."""
import argparse
from collections import Counter
import json
import os
from pathlib import Path
import re
import subprocess
import study

parser = argparse.ArgumentParser()
parser.add_argument("--workspace", required=True, type=Path)
parser.add_argument("--run", required=True, type=Path)
parser.add_argument("--out", required=True, type=Path)
args = parser.parse_args()
report = study.evaluate(args.run)
requests, examples, counts = [], [], Counter()
for entry in study.read(args.run / "observations.json"):
    row = study.read(args.run / "raw" / (entry["id"] + ".json"))["steps"][0]
    if not row["requests"]:
        continue
    response = row["requests"][0]["response"]
    if not response:
        continue
    data = json.loads(response)
    content = data["choices"][0]["message"].get("content", "")
    first = re.split(r"[、,，;；\n]", content)[0].strip()
    first = re.sub(r"^[\d①②③.、。\s]+", "", first).strip()
    source = row["source"]
    valid = first in source[:5]
    requests.append({"id": entry["id"], "response": response, "candidates": source[:5]})
    counts["responses"] += 1
    counts["firstChoiceIsCandidate"] += valid
    counts["firstChoiceOutsideCandidateSet"] += not valid
    if valid and first != row["ranked"][0]:
        examples.append({"id": entry["id"], "modelFirstChoice": first, "actualFirstChoice": row["ranked"][0], "content": content})

env = {**os.environ, "HOME": str(args.run / "isolated-home"),
       "LUA_PATH": str(args.workspace.resolve() / "lua/share/lua/5.4/?.lua") + ";;",
       "LUA_CPATH": str(args.workspace.resolve() / "lua/lib/lua/5.4/?.so") + ";;"}
result = subprocess.run([str(args.workspace.resolve() / "sources/lua-5.4.8/src/lua"),
                         str(study.STUDY / "parser-replay.lua"), str(study.STUDY / "upstream/rime.lua")],
                        input="\n".join(json.dumps(r, ensure_ascii=False) for r in requests) + "\n",
                        text=True, capture_output=True, env=env, check=True)
replayed = [json.loads(line) for line in result.stdout.splitlines()]
counts["originalParserNoMatch"] = sum(r["best"] is False for r in replayed)
for response, request in zip(replayed, requests):
    actual = next(c for c in report["cases"] if c["id"] == response["id"])
    expected = request["candidates"][response["best"] - 1] if response["best"] else actual["source"][0]
    assert actual["ranked"][0] == expected, "Offline original-parser replay disagrees with actual filter"

counts["validFirstChoiceNotApplied"] = len(examples)
output = {"status": "completed", "kind": "offline diagnostic over actual responses, not a new model treatment",
          "counts": dict(counts), "ignoredValidChoices": examples, "originalParserReplay": replayed,
          "cause": "Lua patterns operate on UTF-8 bytes: multibyte punctuation inside bracket character classes also splits/strips bytes belonging to Chinese characters",
          "sourceSha256": study.digest(study.STUDY / "upstream/rime.lua")}
with args.out.open("x") as stream:
    json.dump(output, stream, ensure_ascii=False, indent=2)
    stream.write("\n")
print(json.dumps(counts, indent=2))
