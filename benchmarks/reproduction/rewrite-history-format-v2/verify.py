"""Verify the archived formatter prompt comparison without native inference."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 d=json.loads((ROOT/'benchmarks/runs/rewrite-history-format-v2/archive.json').read_text())
 for path,h in d['files'].items():
  if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=h:raise ValueError('Changed formatter archive: '+path)
 print('Verified',len(d['files']),'formatter comparison evidence files')
if __name__=='__main__':main()
