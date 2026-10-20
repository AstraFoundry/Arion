"""Verify the combined structure/context diagnostic archive without model assets."""
import hashlib
import json
import pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
    data=json.loads((ROOT/'benchmarks/runs/context-repair-v1/archive.json').read_text())
    for name,digest in data['files'].items():
        if hashlib.sha256((ROOT/name).read_bytes()).hexdigest()!=digest:raise ValueError('Changed archive file: '+name)
    print(f"Verified {len(data['files'])} evidence files")
if __name__=='__main__':main()
