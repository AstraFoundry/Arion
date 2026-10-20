"""Verify archived staged research bytes without weights or inference."""
import hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 data=json.loads((ROOT/'benchmarks/runs/rewrite-bounded-edit-v1/archive.json').read_text())
 for name,digest in data['files'].items():
  if hashlib.sha256((ROOT/name).read_bytes()).hexdigest()!=digest:raise ValueError('Changed archived evidence: '+name)
 print('Verified',len(data['files']),'evidence files')
if __name__=='__main__':main()
