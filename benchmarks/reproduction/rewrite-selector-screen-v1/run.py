"""Run complete source preparation and all four actual selector-only treatments."""
import pathlib,sys,json,subprocess,time,argparse,hashlib
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
from gguf_template_identity import identity
import pipeline
ROOT=engine.ROOT
SOURCES=['benchmarks/corpora/rewrite-history-v1/inputs.json','benchmarks/corpora/rewrite-tone-controls-v1/inputs.json','benchmarks/corpora/rewrite-semantic-roles-v1/inputs.json']
def rowhash(x):return hashlib.sha256(json.dumps(x,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def inputs():return [dict(sample=s,path=p) for p in SOURCES for s in engine.read(ROOT/p)]
def execute():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--cleanup-worker',type=pathlib.Path,required=True);a=parser.parse_args();f=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/f['modelPath'];engine.verify(f)
 if engine.sha(a.cleanup_worker)!=f['cleanupWorkerSha256']:raise ValueError('Changed cleanup executable')
 out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);m=dict(pipeline='selector-screen-v1',status='running',model=f['model'],runtimeFiles=f['runtimeFiles'],cleanupWorkerSha256=f['cleanupWorkerSha256'],freezeSha256=engine.sha(HERE/'freeze.json'),arms=pipeline.ARMS,plannedSources=457,plannedSourceTreatments=1828,newAsrGenerations=0,newOcrGenerations=0,formatterGenerations=0,workers=[],scope=engine.read(HERE/'protocol.json')['limits']);engine.write(out/'manifest.json',m);preps=[];worker=None;cleaner=None;count=0;skipped=0;attempted=0;errors=0;log=(out/'cleanup-worker.log').open('x')
 try:
  cleaner=subprocess.Popen([str(a.cleanup_worker.resolve())],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,text=True,bufsize=1)
  with (out/'preparations.jsonl').open('x') as stream:
   for item in inputs():
    s=item['sample'];start=time.monotonic();cleaner.stdin.write(json.dumps({k:s[k] for k in ['id','source']},ensure_ascii=False)+'\n');cleaner.stdin.flush();line=cleaner.stdout.readline()
    if not line:raise RuntimeError('Actual Go cleanup worker ended')
    observed=json.loads(line);elapsed=(time.monotonic()-start)*1000
    if observed['id']!=s['id'] or observed['source']!=s['source'] or observed['cleanup']['version']!='speech-spans-v2':raise ValueError('Wrong actual cleanup source')
    _,prepared=pipeline.selection_payload('baseline',s,observed['cleanup']);p=dict(sample=s,sourceControl=dict(path=item['path'],rowSha256=rowhash(s)),cleanupObservation=observed,cleanupIpcMs=elapsed,prepared=prepared);preps.append(p);stream.write(json.dumps(p,ensure_ascii=False)+'\n')
  cleaner.stdin.close();cleaner.wait(timeout=10)
  if cleaner.returncode:raise RuntimeError('Actual cleanup failed')
  m.update(sourcePreparations=len(preps),preparationsSha256=engine.sha(out/'preparations.jsonl'));engine.write(out/'manifest.json',m)
  with (out/'results.jsonl').open('x') as stream,(out/'case-outcomes.jsonl').open('x') as outcomes:
   for arm in pipeline.ARMS:
    for scene in ['work','email','prompt','chat']:
     subset=[p for p in preps if p['sample']['scene']==scene]
     if any(p['prepared']['hypotheses'] for p in subset):
      name='selector-screen-v1-'+arm+'-'+scene;worker=Session(out,name,18822,gpu_layers=99,verbosity=5);m['workers'].append(dict(arm=arm,scene=scene,name=name,command=worker.command,startupMs=worker.startup_ms))
     for p in subset:
      sample=p['sample'];called=bool(p['prepared']['hypotheses']);outcome=dict(id=sample['id'],arm=arm,sourceRowSha256=rowhash(sample),preparationRowSha256=rowhash(p),modelSelectorCalled=called,formatterExecuted=False);attempted+=1
      if not called:
       skipped+=1;outcome.update(status='no-hypotheses',resultRowSha256=None);outcomes.write(json.dumps(outcome,ensure_ascii=False)+'\n');continue
      request,prepared=pipeline.selection_payload(arm,sample,p['cleanupObservation']['cleanup'])
      if prepared!=p['prepared']:raise ValueError('Changed shared producer within screen')
      template=identity(ROOT,f,pipeline,arm,'selection',scene,request);response,elapsed,rss=worker.generate(request);raw=response['choices'][0]['message']['content'];error=None;edited=None;edits=[];choice=None
      try:edited,edits=pipeline.candidates_for(arm).apply(raw,prepared);choice=json.loads(raw)
      except (ValueError,KeyError,TypeError) as issue:error=type(issue).__name__+': '+str(issue);errors+=1
      row=dict(sample,arm=arm,variant=arm+'-selection',pipeline='selector-screen-v1',stage='selection',effectiveModelScene='spelling-selection',evaluationSet=sample['cohort'],sourceControl=p['sourceControl'],preparationRowSha256=rowhash(p),cleanup=p['cleanupObservation']['cleanup'],cleanupObservation=p['cleanupObservation'],cleanupIpcMs=p['cleanupIpcMs'],sourcePreparation=dict(kind='actual-go-cleanup-reused-within-selector-screen',modelSelectorCalled=True,formatterExecuted=False),prepared=prepared,**template,request=request,response=response,actualContextUsed=sample['context'],selection=choice,editedSource=edited,edits=edits,renderedBody=None,renderError=error,modelSha256=f['model']['sha256'],generationMs=elapsed,residentRssBytes=rss)
      stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;outcome.update(status='selection-invalid' if error else 'selected',resultRowSha256=rowhash(row));outcomes.write(json.dumps(outcome,ensure_ascii=False)+'\n');outcomes.flush();print(arm,sample['id'],repr(raw),repr(edited),error,flush=True)
     if worker:worker.close();worker=None
  log.close();m.update(status='complete',generations=count,selectorGenerations=count,sourcePreparations=len(preps),attemptedSources=attempted,skippedSourceTreatments=skipped,stageProtocolFailures=errors,resultsSha256=engine.sha(out/'results.jsonl'),outcomesSha256=engine.sha(out/'case-outcomes.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  if cleaner and cleaner.poll() is None:cleaner.kill();cleaner.wait()
  log.close();m.update(status='failed',completedGenerations=count,sourcePreparations=len(preps),attemptedSources=attempted,error=str(error));engine.write(out/'manifest.json',m);raise
if __name__=='__main__':execute()
