"""Verify full-history archive and offline inference evidence without model weights."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 d=json.loads((ROOT/'benchmarks/runs/rewrite-history-chain-v1/archive.json').read_text())
 for path,h in d['files'].items():
  if hashlib.sha256((ROOT/path).read_bytes()).hexdigest()!=h:raise ValueError('Changed history archive: '+path)
 print('Verified',len(d['files']),'historical evidence files')
if __name__=='__main__':main()
