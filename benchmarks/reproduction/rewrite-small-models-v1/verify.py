"""Verify archived small-model evidence without loading weights."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
    data=json.loads((ROOT/"benchmarks/runs/rewrite-small-models-v1/archive.json").read_text())
    for path,digest in data["files"].items():
        if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=digest:raise ValueError("Changed archive: "+path)
    print("Verified",len(data["files"]),"evidence files")
if __name__=="__main__":main()
