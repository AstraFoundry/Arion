"""Run every historical formatter with the actual prior spelling choices fixed."""
import pathlib,sys,json,argparse,hashlib
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
import pipeline
ROOT=engine.ROOT;REFERENCE=ROOT/'benchmarks/runs/rewrite-history-chain-v1/run/results.jsonl'
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();f=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/f['modelPath'];engine.verify(f)
 refs=[json.loads(x) for x in REFERENCE.read_text().splitlines()];choices={r['id']:r['selection'] for r in refs if r['stage']=='selection'};finals=[r for r in refs if r['stage']=='format'];out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);m=dict(status='running',pipeline='history-format-v2',freezeSha256=engine.sha(HERE/'freeze.json'),model=f['model'],runtimeFiles=f['runtimeFiles'],sourceCases=401,newSelectorGenerations=0,reusedActualSelections=len(choices),newAsrGenerations=0,workers=[],scope=engine.read(HERE/'protocol.json')['limits']);engine.write(out/'manifest.json',m);worker=None;count=0
 try:
  with (out/'results.jsonl').open('x') as stream:
   for scene in ['work','email','prompt','chat']:
    worker=Session(out,'format-v2-'+scene,18818,gpu_layers=99,verbosity=5);m['workers'].append(dict(scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for ref in [r for r in finals if r['scene']==scene]:
     request,encoding=pipeline.format_payload(ref,ref['editedSource'],ref['prepared'],choices.get(ref['id'],{}),ref['edits']);response,elapsed,rss=worker.generate(request);body=response['choices'][0]['message']['content'];row=dict(ref,pipeline='history-format-v2',variant='history-format-v2',templateVersion='history-format-v2-'+scene,templateSha256=engine.sha(HERE/'prompts.json'),request=request,encoding=encoding,response=response,renderedBody=body,generationMs=elapsed,residentRssBytes=rss,sourceControl=dict(path=str(REFERENCE.relative_to(ROOT)),rowSha256=rowhash(ref)),sourcePreparation=dict(kind='reused-actual-historical-cleanup-and-selection',modelSelectorCalled=False,priorModelSelectorCalled=ref['sourcePreparation']['modelSelectorCalled']),pipelineMs=None,cleanupIpcMs=None,scope='Only formatting generated now; cleanup and conditional selection are actual prior frozen observations. Request latency excludes reused stages; no output rejection or source fallback.')
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(row['id'],round(elapsed),repr(body),flush=True)
    worker.close();worker=None
  m.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  m.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',m);raise
if __name__=='__main__':main()
