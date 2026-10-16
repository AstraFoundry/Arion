"""Reuse immutable v2 identity checks and retrieval for a paired comparison."""
import hashlib
import importlib.util
import json
from pathlib import Path
import sys

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
V2 = HERE.parent / "user-modify-asr-v2"
sys.path.insert(0, str(V2))
import policy as v2_policy

spec = importlib.util.spec_from_file_location("memory_v2_runner", V2/"run.py")
v2_runner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(v2_runner)


def sha(p):
    return hashlib.sha256(Path(p).read_bytes()).hexdigest()


def write(p, value):
    Path(p).write_text(json.dumps(value, ensure_ascii=False, indent=2)+"\n")


def read(p):
    return json.loads(Path(p).read_text())


def source_paths():
    return [HERE/p for p in ["run.py", "common.py", "retrieval.py"]] + [
        V2/"run.py", V2/"policy.py", V2.parent/"user-modify-asr-v1/run.py",
        ROOT/"apps/desktop/cmd/arion-memory-loop/main.go",
        ROOT/"apps/desktop/infrastructure/vocabulary/vocabulary.go",
        ROOT/"packages/core/domain/learning.go", ROOT/"packages/core/domain/memory.go",
    ]


def source_hashes():
    return {str(p.relative_to(ROOT)): sha(p) for p in source_paths()}


def verify_freeze(fixtures):
    frozen = read(fixtures/"freeze.json")
    if frozen["status"] != "frozen_before_any_v3_inference":
        raise ValueError("Initial freeze missing")
    for name, value in frozen["files"].items():
        p=fixtures/name
        if sha(p) != value["sha256"] or p.stat().st_size != value["sizeBytes"]:
            raise ValueError("Frozen fixture changed: "+name)


def verify_separation(fixtures):
    inventories={}
    for split in ["dev", "test"]:
        teaching=read(fixtures/split/"teaching.json")["events"]
        future=read(fixtures/split/"recognition.json")["samples"]
        labels=read(fixtures/split/"evaluation.json")
        inventories[split]=dict(
            entities={x["term"].casefold() for x in labels["teaching"]},
            audio={x["audio"]["sha256"] for x in teaching+future},
            speakers={x["audio"]["speakerId"] for x in teaching+future},
            sessions={x["sessionId"] for x in teaching+future},
        )
        if {x["audio"]["sha256"] for x in teaching} & {x["audio"]["sha256"] for x in future}:
            raise ValueError("Teaching audio reused for recurrence")
        if {x["sessionId"] for x in teaching} & {x["sessionId"] for x in future}:
            raise ValueError("Teaching sessions reused for recurrence")
    for field in ["entities", "audio", "speakers", "sessions"]:
        if inventories["dev"][field] & inventories["test"][field]:
            raise ValueError("Cross-split overlap: "+field)
    return {split: {field:len(values) for field,values in data.items()} for split,data in inventories.items()}
