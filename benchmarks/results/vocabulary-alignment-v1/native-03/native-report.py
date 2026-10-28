"""Replay controlled native vocabulary acceptance from immutable wire evidence."""
import hashlib
import json
import pathlib
import sys


def read(path):
    return json.loads(path.read_text())


def evaluate(run):
    manifest = read(run / "manifest.json")
    for name, expected in manifest["files"].items():
        path = pathlib.PurePosixPath(name)
        if path.is_absolute() or ".." in path.parts:
            raise ValueError("invalid archive path")
        if hashlib.sha256((run / name).read_bytes()).hexdigest() != expected:
            raise ValueError(f"hash mismatch: {name}")
    if read(run / "status.json")["status"] != "completed":
        raise ValueError("incomplete native run")
    observations = [json.loads(line) for line in (run / "observations.jsonl").read_text().splitlines()]
    if observations[-1]["kind"] != "complete":
        raise ValueError("missing completion event")
    labels = read(pathlib.Path(__file__).with_name("native-expected.json"))
    rows = [row for row in observations if row["kind"] == "observation"]
    if len(rows) != 2 * len(labels) or {(r["id"], r["phase"]) for r in rows} != {(key, phase) for key in labels for phase in ("preview", "final")}:
        raise ValueError("missing or duplicate observations")
    cases = []
    for row in rows:
        label = labels[row["id"]]
        request = row["request"]
        selectors = [w["request"] for w in row["wire"] if w["request"].get("response_format")]
        formats = [w["request"] for w in row["wire"] if not w["request"].get("response_format")]
        context_ok = request["Terms"] == label["terms"]
        for call in selectors:
            payload = json.loads(call["messages"][-1]["content"])
            context_ok &= payload["context"].get("Terms") == "\n".join(label["terms"])
        format_ok = len(formats) == 1 and formats[0]["messages"][-1]["content"] == label["formatterSource"]
        passed = (not row["error"] and not row["result"].get("truncated", False)
                  and row["deliveryVerdict"] == "accepted" and context_ok and format_ok
                  and row["result"]["text"] in label["outputs"]
                  and len(selectors) == label["selectorCalls"])
        cases.append({"id": row["id"], "phase": row["phase"], "text": row["result"]["text"],
                      "passed": passed, "canonicalContext": context_ok, "formatterIsolation": format_ok,
                      "selectorCalls": len(selectors), "elapsedMs": row["elapsedMs"]})
    return {"schema": "vocabulary-native-acceptance-v1", "passed": all(r["passed"] for r in cases),
            "observations": len(cases), "cases": cases,
            "limitations": "Constructed development text through actual pinned GGUF; no microphone, ASR, natural-speech accuracy, independent human rating or Windows execution. Preview/final are harness phases, not two independent samples. Warm cache affects final timing."}


if __name__ == "__main__":
    result = evaluate(pathlib.Path(sys.argv[1]))
    print(json.dumps(result, ensure_ascii=False, indent=2))
    if not result["passed"]:
        raise SystemExit(1)
