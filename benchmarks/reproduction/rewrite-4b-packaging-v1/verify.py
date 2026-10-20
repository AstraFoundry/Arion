"""Verify frozen research files without loading a model."""
import hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
    archive=json.loads((ROOT/"benchmarks/runs/rewrite-4b-packaging-v1/archive.json").read_text())
    for name,digest in archive["files"].items():
        if hashlib.sha256((ROOT/name).read_bytes()).hexdigest()!=digest:raise ValueError("Changed archive: "+name)
    print('Verified',len(archive['files']),'evidence files')
if __name__=="__main__":main()
