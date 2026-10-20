"""Verify the archived research evidence without inference."""
import hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
    archive=json.loads((ROOT/'benchmarks/runs/rewrite-entity-evidence-v1/archive.json').read_text())
    for name,digest in archive['files'].items():
        if hashlib.sha256((ROOT/name).read_bytes()).hexdigest()!=digest:raise ValueError('Changed archive: '+name)
    print(f"Verified {len(archive['files'])} evidence files")
if __name__=='__main__':main()
