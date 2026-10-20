"""Run all36 identical formatter requests on an explicitly observed Metal provider."""
import pathlib,sys,json,argparse,hashlib
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
ROOT=engine.ROOT
def rowhash(row):return hashlib.sha256(json.dumps(row,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();f=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/f['modelPath'];engine.verify(f)
 refs=[json.loads(x) for x in (ROOT/'benchmarks/runs/rewrite-formatter-v3/run/results.jsonl').read_text().splitlines()];out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);m=dict(status='running',phase='dev',freezeSha256=engine.sha(HERE/'freeze.json'),model=f['model'],runtimeFiles=f['runtimeFiles'],workers=[],sourceCases=36,pipeline='metal-regression-v1',reusedActualSelections=36);engine.write(out/'manifest.json',m);worker=None;count=0
 try:
  with (out/'results.jsonl').open('x') as stream:
   for scene in ['work','email','prompt','chat']:
    worker=Session(out,'metal-'+scene,18810,gpu_layers=99,verbosity=5);m['workers'].append(dict(scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for ref in [r for r in refs if r['scene']==scene]:
     response,elapsed,rss=worker.generate(ref['request']);body=response['choices'][0]['message']['content'];row=dict(ref,variant='metal-full36',pipeline='metal-regression-v1',response=response,renderedBody=body,generationMs=elapsed,textStageMs=elapsed,preparationMs=0,renderMs=0,residentRssBytes=rss,sourceControl=dict(run='rewrite-formatter-v3/run',variant='formatter-v3',id=ref['id'],rowSha256=rowhash(ref)),scope='Same36 exposed development formatter requests and actual v2 selector outputs, complete provider comparison. No new selector generations; no full two-call/new-audio/independent/min16GB/concurrent native acceptance.')
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(ref['id'],round(elapsed),repr(body),flush=True)
    worker.close();worker=None
  m.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  m.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',m);raise
if __name__=='__main__':main()
