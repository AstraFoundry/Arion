"""Exercise every historical tuple and preserve failures without invented bodies."""
import pathlib,sys,json,time,subprocess,argparse,hashlib,os
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
import pipeline
ROOT=engine.ROOT
SOURCE='benchmarks/corpora/rewrite-history-v1/inputs.json'
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--cleanup-worker',type=pathlib.Path,required=True);a=parser.parse_args();f=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/f['modelPath'];engine.verify(f)
 if engine.sha(a.cleanup_worker)!=f['cleanupWorkerSha256']:raise ValueError('Changed actual cleanup executable')
 out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);samples=engine.read(ROOT/SOURCE);m=dict(status='running',pipeline='history-chain-v1',freezeSha256=engine.sha(HERE/'freeze.json'),model=f['model'],runtimeFiles=f['runtimeFiles'],cleanupWorkerSha256=f['cleanupWorkerSha256'],sourceCases=len(samples),newAsrGenerations=0,workers=[],scope='Every401 frozen tuple attempted; stage protocol errors retain actual raw output and explicit missing final body, never fabricated fallback. Actual model calls and delivered-body coverage separate.',timing='Monotonic actual cleanup IPC to final body when available; no priming/warmups, model startup separate. Quality breadth experiment, not latency-SLO/cold-warm/native/concurrent/min16 acceptance.');engine.write(out/'manifest.json',m);worker=None;cleaner=None;count=0;selection_count=0;format_count=0;failures=0;attempted=0;log=(out/'cleanup-worker.log').open('x')
 try:
  cleaner=subprocess.Popen([str(a.cleanup_worker.resolve())],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,text=True,bufsize=1)
  with (out/'results.jsonl').open('x') as stream,(out/'case-outcomes.jsonl').open('x') as outcomes:
   for scene in ['work','email','prompt','chat']:
    worker=Session(out,'history-'+scene,18816,gpu_layers=99,verbosity=5);m['workers'].append(dict(scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for sample in [s for s in samples if s['scene']==scene]:
     start=time.monotonic();cleaner.stdin.write(json.dumps({k:sample[k] for k in ['id','source']},ensure_ascii=False)+'\n');cleaner.stdin.flush();line=cleaner.stdout.readline()
     if not line:raise RuntimeError('Actual Go cleanup worker ended')
     observed=json.loads(line);cleaned_at=time.monotonic()
     if observed['id']!=sample['id'] or observed['source']!=sample['source'] or observed['cleanup']['version']!='speech-spans-v2':raise ValueError('Wrong actual source cleanup')
     cleanup=observed['cleanup'];request,prepared,_=pipeline.selection_payload(sample,cleanup);common=dict(sample,pipeline='history-chain-v1',evaluationSet=sample['cohort'],modelSha256=f['model']['sha256'],sourceControl=dict(path=SOURCE,rowSha256=rowhash(sample)),cleanup=cleanup,cleanupObservation=observed,cleanupIpcMs=(cleaned_at-start)*1000,prepared=prepared);choices={};edited=cleanup['text'];edits=[];selection_hash=None;called=bool(prepared['hypotheses']);outcome=dict(id=sample['id'],sourceRowSha256=rowhash(sample),modelSelectorCalled=called)
     if called:
      response,elapsed,rss=worker.generate(request);raw=response['choices'][0]['message']['content'];error=None
      try:edited,edits=pipeline.candidates.apply(raw,prepared);choices=json.loads(raw)
      except (ValueError,TypeError,KeyError) as issue:error=type(issue).__name__+': '+str(issue)
      row=dict(common,variant='history-selection',stage='selection',effectiveModelScene='spelling-selection',templateVersion='bounded-edit-v2-selection',templateSha256=engine.sha(HERE.parent/'rewrite-bounded-edit-v2/prompts.json'),request=request,actualContextUsed=sample['context'],response=response,selection=choices,editedSource=edited,edits=edits,renderedBody=None,renderError=error,generationMs=elapsed,textStageMs=(time.monotonic()-start)*1000,residentRssBytes=rss)
      stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;selection_count+=1;selection_hash=rowhash(row);print('selection',sample['id'],round(elapsed),repr(raw),error,flush=True)
      if error:
       failures+=1;attempted+=1;outcome.update(status='selection-invalid',selectionRowSha256=selection_hash,finalRowSha256=None,error=error);outcomes.write(json.dumps(outcome,ensure_ascii=False)+'\n');outcomes.flush();continue
     request,encoding=pipeline.format_payload(sample,edited,prepared,choices,edits);response,elapsed,rss=worker.generate(request);finished=time.monotonic();body=response['choices'][0]['message']['content'];row=dict(common,variant='history-format',stage='format',effectiveModelScene=scene,templateVersion='language-profile-v2-'+scene,templateSha256=engine.sha(HERE.parent/'rewrite-language-profile-v2/prompts.json'),request=request,actualContextUsed={},encoding=encoding,response=response,editedSource=edited,edits=edits,selectionRowSha256=selection_hash,sourcePreparation=dict(kind='actual-go-cleanup-conditional-selector',modelSelectorCalled=called,skipReason=None if called else 'zero-produced-hypotheses'),renderedBody=body,renderError=None,generationMs=elapsed,pipelineMs=(finished-start)*1000,residentRssBytes=rss)
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;format_count+=1;attempted+=1;outcome.update(status='generated',selectionRowSha256=selection_hash,finalRowSha256=rowhash(row));outcomes.write(json.dumps(outcome,ensure_ascii=False)+'\n');outcomes.flush();print('format',count,sample['id'],round(elapsed),repr(body),flush=True)
    worker.close();worker=None
  cleaner.stdin.close();cleaner.wait(timeout=10)
  if cleaner.returncode:raise RuntimeError('Go cleanup failed')
  log.close();m.update(status='complete',generations=count,selectorGenerations=selection_count,formatterGenerations=format_count,attemptedSources=attempted,stageProtocolFailures=failures,resultsSha256=engine.sha(out/'results.jsonl'),outcomesSha256=engine.sha(out/'case-outcomes.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  if cleaner and cleaner.poll() is None:cleaner.kill();cleaner.wait()
  log.close();m.update(status='failed',completedGenerations=count,attemptedSources=attempted,error=str(error));engine.write(out/'manifest.json',m);raise
if __name__=='__main__':main()
