"""Verify the frozen archive and regenerate both reports without model/Lua/SDK."""
import json
from pathlib import Path
import study

root = study.ROOT
manifest = study.read(root / "benchmarks/results/rime-llm-reranker-v1/archive.json")
if manifest["status"] != "completed" or not manifest["files"]:
    raise ValueError("Incomplete archive")
seen = set()
for entry in manifest["files"]:
    file = root / entry["path"]
    if file.is_symlink() or not file.resolve().is_relative_to(root) or str(file) in seen:
        raise ValueError("Unsafe or duplicate archive path")
    seen.add(str(file))
    if file.stat().st_size != entry["bytes"] or study.digest(file) != entry["sha256"]:
        raise ValueError(f"Archive bytes changed: {file}")
run = root / "benchmarks/runs/rime-llm-reranker-v1"
results = root / "benchmarks/results/rime-llm-reranker-v1"
for directory, summary in [("final-stock", "summary.json"), ("typing-stock", "typing-summary.json")]:
    report = study.evaluate(run / directory)
    if report != study.read(results / summary):
        raise ValueError("Report replay mismatch")
    meta = study.read(run / directory / "metadata.json")
    if meta["candidateMetadataSha256"] != study.digest(run / "candidates/metadata.json"):
        raise ValueError("Candidate producer provenance mismatch")
    for entry in study.read(run / directory / "observations.json"):
        native = study.read(run / "candidates/raw" / (entry["id"] + ".json"))
        request = study.read(run / directory / "requests" / (entry["id"] + ".json"))
        primary = [s for s in request["steps"] if s["id"].startswith("key-")]
        expected = [r["snapshot"] for r in native["steps"]] if directory == "typing-stock" else [native["beforeConfirmation"]]
        if len(primary) != len(expected) or any(a["candidates"] != b["candidates"] or a["preedit"] != b["preedit"] for a, b in zip(primary, expected)):
            raise ValueError("Candidates differ from actual recorded native producer")
print(json.dumps({"status": "passed", "verifiedFiles": len(seen), "replayedReports": 2,
                  "nativeCandidateProvenance": "matched", "modelsRequired": False}, indent=2))
