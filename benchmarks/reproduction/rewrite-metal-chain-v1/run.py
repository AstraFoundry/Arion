"""Execute cleanup IPC, conditional native spelling selection and native formatting."""
import pathlib,sys,json,time,subprocess,argparse,hashlib,os
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
import pipeline
ROOT=engine.ROOT
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--cleanup-worker',type=pathlib.Path,required=True);a=parser.parse_args()
 frozen=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/frozen['modelPath'];engine.verify(frozen)
 if engine.sha(a.cleanup_worker)!=frozen['cleanupWorkerSha256']:raise ValueError('Changed cleanup executable')
 out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);samples=engine.read(HERE/'inputs.json');development={s['id'] for s in engine.read(ROOT/'benchmarks/reproduction/rewrite-bounded-edit-v2/development.json')}
 m=dict(status='running',pipeline='metal-chain-v1',phase='diagnostic',freezeSha256=engine.sha(HERE/'freeze.json'),model=frozen['model'],runtimeFiles=frozen['runtimeFiles'],cleanupWorkerSha256=frozen['cleanupWorkerSha256'],sourceCases=len(samples),newAsrGenerations=0,workers=[],clock='time.monotonic; continuous cleanup IPC to final model response; excludes model startup and target insertion',selectorSkipRule='Skip iff deterministic producer has zero hypotheses; never fabricate an empty LM response')
 engine.write(out/'manifest.json',m);worker=None;cleaner=None;count=0;selections=0;cleanup_log=(out/'cleanup-worker.log').open('x')
 try:
  cleaner=subprocess.Popen([str(a.cleanup_worker.resolve())],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=cleanup_log,text=True,bufsize=1)
  with (out/'results.jsonl').open('x') as stream:
   for scene in ['work','email','prompt','chat']:
    worker=Session(out,'chain-'+scene,18814,gpu_layers=99,verbosity=5);m['workers'].append(dict(scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for sample in [s for s in samples if s['scene']==scene]:
     start=time.monotonic();cleaner.stdin.write(json.dumps({k:sample[k] for k in ['id','source']},ensure_ascii=False)+'\n');cleaner.stdin.flush();reply=cleaner.stdout.readline()
     if not reply:raise RuntimeError('Cleanup worker ended without result')
     observed=json.loads(reply);cleaned_at=time.monotonic()
     if observed['id']!=sample['id'] or observed['source']!=sample['source']:raise ValueError('Wrong cleanup source identity')
     cleanup=observed['cleanup'];request,prepared,reused=pipeline.selection_payload(sample,cleanup)
     if reused!=cleanup or cleanup['version']!='speech-spans-v2':raise ValueError('Changed actual cleanup result')
     common=dict(sample,evaluationSet='development' if sample['id'] in development else 'natural',pipeline='metal-chain-v1',modelSha256=frozen['model']['sha256'],cleanup=cleanup,cleanupObservation=observed,cleanupIpcMs=(cleaned_at-start)*1000,sourceControl=dict(path='benchmarks/reproduction/rewrite-metal-chain-v1/inputs.json',rowSha256=rowhash(sample)))
     selection_hash=None;choices={};edits=[];edited=cleanup['text'];selection_called=bool(prepared['hypotheses'])
     if selection_called:
      ready=time.monotonic();response,elapsed,rss=worker.generate(request);raw=response['choices'][0]['message']['content'];error=None
      try:edited,edits=pipeline.candidates.apply(raw,prepared);choices=json.loads(raw)
      except (ValueError,TypeError,KeyError) as issue:error=type(issue).__name__+': '+str(issue)
      rendered=time.monotonic()
      row=dict(common,variant='metal-chain-selection',stage='selection',effectiveModelScene='spelling-selection',templateVersion='bounded-edit-v2-selection',templateSha256=engine.sha(HERE.parent/'rewrite-bounded-edit-v2/prompts.json'),request=request,prepared=prepared,actualContextUsed=sample['context'],response=response,selection=choices,editedSource=edited,edits=edits,renderedBody=None,renderError=error,preparationMs=(ready-start)*1000,generationMs=elapsed,textStageMs=(rendered-start)*1000,residentRssBytes=rss,sourcePreparation=dict(modelSelectorCalled=True))
      stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;selections+=1;print('selection',sample['id'],round(elapsed),repr(raw),error,flush=True)
      if error:raise ValueError('Invalid actual selection recorded; no invented formatter result: '+sample['id'])
      selection_hash=rowhash(row)
     format_start=time.monotonic();request,encoding=pipeline.format_payload(sample,edited,prepared,choices,edits);ready=time.monotonic();response,elapsed,rss=worker.generate(request);body=response['choices'][0]['message']['content'];finished=time.monotonic()
     row=dict(common,variant='metal-chain-format',stage='format',effectiveModelScene=scene,templateVersion='language-profile-v2-'+scene,templateSha256=engine.sha(HERE.parent/'rewrite-language-profile-v2/prompts.json'),request=request,prepared=prepared,actualContextUsed={},response=response,encoding=encoding,editedSource=edited,edits=edits,renderedBody=body,renderError=None,preparationMs=(ready-format_start)*1000,generationMs=elapsed,textStageMs=(finished-format_start)*1000,pipelineMs=(finished-start)*1000,residentRssBytes=rss,coordinatorRssBytes=int(subprocess.check_output(['ps','-o','rss=','-p',str(os.getpid())],text=True).strip())*1024,selectionRowSha256=selection_hash,sourcePreparation=dict(kind='actual-go-cleanup-conditional-selector',modelSelectorCalled=selection_called,hypotheses=prepared['hypotheses'],skipReason=None if selection_called else 'zero-produced-hypotheses'))
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print('format',count,sample['id'],round(elapsed),round(row['pipelineMs']),repr(body),flush=True)
    worker.close();worker=None
  cleaner.stdin.close();cleaner.wait(timeout=10)
  if cleaner.returncode:raise RuntimeError('Cleanup worker failed')
  cleanup_log.close();m.update(status='complete',generations=count,newSelectorGenerations=selections,formatterGenerations=len(samples),resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  if cleaner and cleaner.poll() is None:cleaner.kill();cleaner.wait()
  cleanup_log.close();m.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',m);raise
if __name__=='__main__':main()
