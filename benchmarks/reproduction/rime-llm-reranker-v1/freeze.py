"""Create a one-time checksum inventory after evidence and prose are complete."""
from pathlib import Path
import study

result = study.ROOT / "benchmarks/results/rime-llm-reranker-v1"
manifest = result / "archive.json"
files = []
for relative in ["benchmarks/corpora/rime-llm-reranker-v1", "benchmarks/reproduction/rime-llm-reranker-v1",
                 "benchmarks/runs/rime-llm-reranker-v1", "benchmarks/results/rime-llm-reranker-v1"]:
    for file in sorted((study.ROOT / relative).rglob("*")):
        if file == manifest or not file.is_file() or "__pycache__" in file.parts or file.name == ".DS_Store":
            continue
        if file.is_symlink():
            raise ValueError("Refuse symlink in archive")
        files.append({"path": str(file.relative_to(study.ROOT)), "sha256": study.digest(file), "bytes": file.stat().st_size})
with manifest.open("x") as stream:
    import json
    json.dump({"version": 1, "status": "completed", "files": files}, stream, indent=2)
    stream.write("\n")
print(f"Froze {len(files)} files")
