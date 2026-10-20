"""Verify native cache study archive bytes without inference."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 d=json.loads((ROOT/'benchmarks/runs/rewrite-metal-prefix-v1/archive.json').read_text())
 for path,h in d['files'].items():
  if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=h:raise ValueError('Changed cache archive: '+path)
 print('Verified',len(d['files']),'prefix evidence files')
if __name__=='__main__':main()
