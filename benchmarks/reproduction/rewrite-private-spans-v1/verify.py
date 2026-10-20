"""Verify frozen experiment evidence without model acquisition or inference."""
import hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
    data=json.loads((ROOT/"benchmarks/runs/rewrite-private-spans-v1/archive.json").read_text())
    for path,digest in data["files"].items():
        if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=digest:raise ValueError("Changed archived evidence: "+path)
    print("Verified",len(data["files"]),"evidence files")
if __name__=="__main__":main()
