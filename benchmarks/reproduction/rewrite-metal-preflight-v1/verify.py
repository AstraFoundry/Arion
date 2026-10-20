"""Verify exact capability-probe archive bytes without model execution."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
def main():
 data=json.loads((ROOT/'benchmarks/runs/rewrite-metal-preflight-v1/archive.json').read_text())
 for name,digest in data['files'].items():
  if hashlib.sha256((ROOT/name).read_bytes()).hexdigest()!=digest:raise ValueError('Changed probe archive: '+name)
 print('Verified',len(data['files']),'probe evidence files')
if __name__=='__main__':main()
