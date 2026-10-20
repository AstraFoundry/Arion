"""Versioned executor with explicit prompt ownership; v2 remains immutable evidence."""
import pathlib,argparse,json,subprocess,time,hashlib
import gguf_rewrite as engine
from gguf_session_v2 import Session
from gguf_template_identity import identity
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def execute(here,pipeline,plans,metadata):
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);p.add_argument('--cleanup-worker',type=pathlib.Path,required=True);a=p.parse_args();f=engine.read(here/'freeze.json');engine.MODEL=engine.ROOT/f['modelPath'];engine.verify(f)
 if engine.sha(a.cleanup_worker)!=f['cleanupWorkerSha256']:raise ValueError('Changed cleanup executable')
 out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);m=dict(metadata,status='running',freezeSha256=engine.sha(here/'freeze.json'),model=f['model'],runtimeFiles=f['runtimeFiles'],cleanupWorkerSha256=f['cleanupWorkerSha256'],workers=[]);engine.write(out/'manifest.json',m);worker=None;cleaner=None;count=0;selectors=0;formatters=0;attempted=0;errors=0;log=(out/'cleanup-worker.log').open('x')
 try:
  cleaner=subprocess.Popen([str(a.cleanup_worker.resolve())],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,text=True,bufsize=1)
  with (out/'results.jsonl').open('x') as stream,(out/'case-outcomes.jsonl').open('x') as outcomes:
   for arm,items in plans:
    for scene in ['work','email','prompt','chat']:
     subset=[item for item in items if item['sample']['scene']==scene]
     if not subset:continue
     name=metadata['pipeline']+'-'+arm+'-'+scene;worker=Session(out,name,18822,gpu_layers=99,verbosity=5);m['workers'].append(dict(arm=arm,scene=scene,name=name,command=worker.command,startupMs=worker.startup_ms))
     for item in subset:
      sample=item['sample'];start=time.monotonic();cleaner.stdin.write(json.dumps({k:sample[k] for k in ['id','source']},ensure_ascii=False)+'\n');cleaner.stdin.flush();line=cleaner.stdout.readline()
      if not line:raise RuntimeError('Actual Go cleanup worker ended')
      observed=json.loads(line);cleaned=time.monotonic();cleanup=observed['cleanup']
      if observed['id']!=sample['id'] or observed['source']!=sample['source'] or cleanup['version']!='speech-spans-v2':raise ValueError('Wrong actual source cleanup')
      request,prepared=pipeline.selection_payload(arm,sample,cleanup);called=bool(prepared['hypotheses']);source_control=dict(path=item['sourcePath'],rowSha256=rowhash(sample));common=dict(sample,arm=arm,pipeline=metadata['pipeline'],evaluationSet=sample['cohort'],modelSha256=f['model']['sha256'],sourceControl=source_control,cleanup=cleanup,cleanupObservation=observed,cleanupIpcMs=(cleaned-start)*1000,prepared=prepared);choice={};edited=cleanup['text'];edits=[];selection_hash=None;outcome=dict(id=sample['id'],arm=arm,sourceRowSha256=rowhash(sample),modelSelectorCalled=called)
      if called:
       template=identity(engine.ROOT,f,pipeline,arm,'selection',scene,request);response,elapsed,rss=worker.generate(request);raw=response['choices'][0]['message']['content'];error=None
       try:edited,edits=pipeline.candidates_for(arm).apply(raw,prepared);choice=json.loads(raw)
       except (ValueError,KeyError,TypeError) as issue:error=type(issue).__name__+': '+str(issue)
       row=dict(common,variant=arm+'-selection',stage='selection',effectiveModelScene='spelling-selection',**template,request=request,actualContextUsed=sample['context'],response=response,selection=choice,editedSource=edited,edits=edits,renderedBody=None,renderError=error,generationMs=elapsed,residentRssBytes=rss)
       stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;selectors+=1;selection_hash=rowhash(row);print(arm,'selection',sample['id'],repr(raw),error,flush=True)
       if error:
        attempted+=1;errors+=1;outcome.update(status='selection-invalid',selectionRowSha256=selection_hash,finalRowSha256=None,error=error);outcomes.write(json.dumps(outcome,ensure_ascii=False)+'\n');outcomes.flush();continue
      request,encoding=pipeline.format_payload(sample,edited,prepared,choice,edits);template=identity(engine.ROOT,f,pipeline,arm,'format',scene,request);response,elapsed,rss=worker.generate(request);body=response['choices'][0]['message']['content'];finished=time.monotonic();row=dict(common,variant=arm+'-format',stage='format',effectiveModelScene=scene,**template,request=request,actualContextUsed={},encoding=encoding,response=response,editedSource=edited,edits=edits,selectionRowSha256=selection_hash,sourcePreparation=dict(kind='actual-go-cleanup-conditional-selector',modelSelectorCalled=called,skipReason=None if called else 'zero-produced-hypotheses'),renderedBody=body,renderError=None,generationMs=elapsed,pipelineMs=(finished-start)*1000,residentRssBytes=rss)
      stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;formatters+=1;attempted+=1;outcome.update(status='generated',selectionRowSha256=selection_hash,finalRowSha256=rowhash(row));outcomes.write(json.dumps(outcome,ensure_ascii=False)+'\n');outcomes.flush();print(arm,'format',sample['id'],round(elapsed),repr(body),flush=True)
     worker.close();worker=None
  cleaner.stdin.close();cleaner.wait(timeout=10)
  if cleaner.returncode:raise RuntimeError('Actual cleanup worker failed')
  log.close();m.update(status='complete',generations=count,selectorGenerations=selectors,formatterGenerations=formatters,attemptedSources=attempted,stageProtocolFailures=errors,resultsSha256=engine.sha(out/'results.jsonl'),outcomesSha256=engine.sha(out/'case-outcomes.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  if cleaner and cleaner.poll() is None:cleaner.kill();cleaner.wait()
  log.close();m.update(status='failed',completedGenerations=count,attemptedSources=attempted,error=str(error));engine.write(out/'manifest.json',m);raise
