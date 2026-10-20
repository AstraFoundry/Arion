"""Verify the complete semantic-v2 archive without model inference."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 d=json.loads((ROOT/'benchmarks/runs/rewrite-selector-semantic-v2/archive.json').read_text())
 for path,h in d['files'].items():
  if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=h:raise ValueError('Changed semantic-v2 archive: '+path)
 print('Verified',len(d['files']),'semantic-v2 evidence files')
if __name__=='__main__':main()
