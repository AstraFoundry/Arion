"""Verify exact v3 archive bytes without inference."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 d=json.loads((ROOT/'benchmarks/runs/rewrite-formatter-v3/archive.json').read_text())
 for name,digest in d['files'].items():
  if hashlib.sha256((ROOT/name).read_bytes()).hexdigest()!=digest:raise ValueError('Changed archive: '+name)
 print('Verified',len(d['files']),'evidence files')
if __name__=='__main__':main()
