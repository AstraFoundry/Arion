"""Replay the explicit failed-before-generation microbatch experiment."""
import argparse,pathlib,json,hashlib
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-small-cache-parity-v1'
def read(path):return json.loads(path.read_text())
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def checked():
 frozen=read(HERE/'freeze.json');manifest=read(RUN/'failed/manifest.json');environment=read(RUN/'run-environment.json');failure=read(RUN/'failure.json')
 for path,digest in frozen['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen failed input')
 if manifest['status']!='failed' or manifest['completedGenerations']!=0 or (RUN/'failed/results.jsonl').read_bytes()!=b'':raise ValueError('Wrong failed-generation coverage')
 if manifest['freezeSha256']!=sha(HERE/'freeze.json') or environment['freezeSha256']!=sha(HERE/'freeze.json') or environment['environment']!={'LLAMA_ARG_BATCH':'1','LLAMA_ARG_UBATCH':'1'}:raise ValueError('Wrong failed treatment')
 if sha(RUN/'runtime-common.cpp')!=failure['source']['sha256'] or 'failed to add token 0 at index 1' not in (RUN/'failed/forward-uncached-work-server.log').read_text():raise ValueError('Changed failure diagnosis')
 return dict(status='failed-before-generation',generations=0,reviewedOutputs=0,productionActivated=False,failure=failure)
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();data=checked();out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(data,indent=2)+'\n');print(json.dumps(data,indent=2))
if __name__=='__main__':main()
