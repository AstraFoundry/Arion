"""Verify the frozen OCR-alignment evidence archive without loading a model."""
import hashlib
import json
import pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
    archive=json.loads((ROOT/'benchmarks/runs/ocr-alignment-v2/archive.json').read_text())
    for name,digest in archive['files'].items():
        if hashlib.sha256((ROOT/name).read_bytes()).hexdigest()!=digest:raise ValueError('Archive identity mismatch: '+name)
    print(f"Verified {len(archive['files'])} evidence files")
if __name__=='__main__':main()
