"""Verify complete selector-screen archive identities without inference."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 d=json.loads((ROOT/'benchmarks/runs/rewrite-selector-screen-v1/archive.json').read_text())
 for path,h in d['files'].items():
  if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=h:raise ValueError('Changed selector-screen archive: '+path)
 print('Verified',len(d['files']),'selector-screen evidence files')
if __name__=='__main__':main()
