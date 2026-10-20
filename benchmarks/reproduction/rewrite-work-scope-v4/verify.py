"""Verify archived work formatter evidence without inference."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 d=json.loads((ROOT/'benchmarks/runs/rewrite-work-scope-v4/archive.json').read_text())
 for path,h in d['files'].items():
  if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=h:raise ValueError('Changed work formatter archive: '+path)
 print('Verified',len(d['files']),'work formatter evidence files')
if __name__=='__main__':main()
