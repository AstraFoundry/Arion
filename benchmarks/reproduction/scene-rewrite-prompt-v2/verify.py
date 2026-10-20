"""Verify archived prompt study evidence without running inference."""
import hashlib
import json
import pathlib

ROOT=pathlib.Path(__file__).resolve().parents[3]
MANIFEST=ROOT/'benchmarks/runs/scene-rewrite-prompt-v2/archive.json'

def main():
    archive=json.loads(MANIFEST.read_text())
    for relative,expected in archive['files'].items():
        path=ROOT/relative
        if hashlib.sha256(path.read_bytes()).hexdigest()!=expected:
            raise ValueError('Archive identity mismatch: '+relative)
    print(f"Verified {len(archive['files'])} archived files")

if __name__=='__main__':main()
