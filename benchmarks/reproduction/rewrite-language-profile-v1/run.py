"""Paired source-only editing on full36 regressions plus all120 natural ASR sources."""
import pathlib,sys,json,argparse,hashlib
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
import pipeline
ROOT=engine.ROOT
def rh(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();f=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/f['modelPath'];engine.verify(f);p=engine.read(HERE/'protocol.json')
 sources={kind:[json.loads(x) for x in (ROOT/path).read_text().splitlines()] for kind,path in p['references'].items()};selected={r['id']:r for r in [json.loads(x) for x in (ROOT/'benchmarks/runs/rewrite-bounded-edit-v2/run/results.jsonl').read_text().splitlines()] if r['stage']=='selection'};out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);m=dict(status='running',phase='diagnostic',freezeSha256=engine.sha(HERE/'freeze.json'),model=f['model'],runtimeFiles=f['runtimeFiles'],workers=[],sourceCases=156,newAsrGenerations=0,newSelectorGenerations=0,pipeline='language-profile-v1');engine.write(out/'manifest.json',m);worker=None;count=0
 try:
  with (out/'results.jsonl').open('x') as stream:
   for kind,scene in [('development',s) for s in ['work','email','prompt','chat']]+[('natural','chat')]:
    worker=Session(out,kind+'-'+scene,18812,gpu_layers=99,verbosity=5);m['workers'].append(dict(evaluationSet=kind,scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for ref in [r for r in sources[kind] if r['scene']==scene]:
     if kind=='development':
      selection=selected[ref['id']]
      if selection['editedSource']!=ref['editedSource'] or selection['edits']!=ref['edits']:raise ValueError('Changed actual native source edits')
      prepared,choices=selection['prepared'],selection['selection']
     else:
      prepared=pipeline.candidates.prepare(ref['editedSource'],ref['context'],scene);choices={}
      if prepared['hypotheses'] or ref['sourcePreparation']['modelSelectorCalled'] is not False:raise ValueError('No simulated natural selector response')
     request,encoding=pipeline.format_payload(ref,ref['editedSource'],prepared,choices,ref['edits']);response,elapsed,rss=worker.generate(request);body=response['choices'][0]['message']['content']
     row=dict(ref,variant='language-profile-'+kind,evaluationSet=kind,pipeline='language-profile-v1',templateVersion='language-profile-v1-'+scene,templateSha256=engine.sha(HERE/'prompts.json'),request=request,encoding=encoding,response=response,renderedBody=body,generationMs=elapsed,textStageMs=elapsed,preparationMs=None,renderMs=0,residentRssBytes=rss,sourceControl=dict(path=p['references'][kind],variant=ref['variant'],id=ref['id'],rowSha256=rh(ref)),scope='All36 development and120 historical natural sources, source-derived Han/Latin presence, common English editing policy, same Metal2B/source/model/budget. No new ASR or spelling-selection inference, no reference/old body/model score in messages. No output rejection/source fallback or production acceptance.')
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(kind,count,ref['id'],round(elapsed),repr(body),flush=True)
    worker.close();worker=None
  m.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  m.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',m);raise
if __name__=='__main__':main()
