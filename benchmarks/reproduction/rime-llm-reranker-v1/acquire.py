"""Acquire the pinned public MLX checkpoint; weights stay in the private workspace."""
import concurrent.futures
import hashlib
import json
from pathlib import Path
import subprocess
import sys

workspace = Path(sys.argv[1]).resolve()
metadata = json.loads((workspace / "model-blobs.json").read_text())
revision = metadata["sha"]
assert revision == "3067585164dbcc505eec73d554349de6a27571a4"
destination = workspace / "models" / "Qwen3.5-0.8B-MLX-bf16"
destination.mkdir(parents=True, exist_ok=True)


def fetch(item):
    name = item["rfilename"]
    if name == ".gitattributes":
        return None
    target = destination / name
    if target.exists():
        raise RuntimeError(f"Refuse existing artifact: {target}")
    url = f"https://huggingface.co/{metadata['id']}/resolve/{revision}/{name}"
    partial = target.with_suffix(target.suffix + ".partial")
    subprocess.run(["curl", "--fail", "--location", "--retry", "3", "--output", str(partial), url], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    digest = hashlib.file_digest(partial.open("rb"), "sha256").hexdigest()
    if item.get("lfs") and digest != item["lfs"]["sha256"]:
        raise RuntimeError(f"Upstream digest mismatch: {name}")
    if partial.stat().st_size != item["size"]:
        raise RuntimeError(f"Upstream size mismatch: {name}")
    partial.rename(target)
    print(f"Verified {name}: {target.stat().st_size} bytes", flush=True)
    return {"path": name, "sha256": digest, "bytes": target.stat().st_size, "url": url}


with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
    files = [row for row in pool.map(fetch, metadata["siblings"]) if row]
(workspace / "model-manifest.json").write_text(json.dumps({"repository": metadata["id"], "revision": revision, "license": "Apache-2.0", "directory": str(destination), "files": files}, indent=2) + "\n")
