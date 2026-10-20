"""Prime every declared script/scene prefix and retain actual startup responses."""
import pathlib,importlib.util,json,sys,argparse,subprocess,time
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
prior=module('primed_continuous_runner',HERE.parent/'rewrite-metal-chain-v1/run.py');payloads=module('primed_runner_payloads',HERE/'pipeline.py')

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--cleanup-worker',type=pathlib.Path,required=True);parser.add_argument('--order',choices=['forward','reverse'],required=True);a=parser.parse_args();warmups=json.loads((HERE/'warmups.json').read_text());native=prior.Session;read=prior.engine.read;warm_rows=[];workers=[]
 if a.out.exists():raise FileExistsError('Existing run output')
 def ordered_read(path):
  result=read(path)
  if pathlib.Path(path).resolve()==HERE/'inputs.json' and a.order=='reverse':return list(reversed(result))
  return result
 class PrimedSession(native):
  def __init__(self,out,name,port,**kwargs):
   super().__init__(out,name,port,**kwargs);load_ms=self.startup_ms;start=time.monotonic();scene=name.removeprefix('chain-')
   try:
    samples=[s for s in warmups if s['workerScene']==scene]
    command=[str(a.cleanup_worker.resolve())];data=''.join(json.dumps({k:s[k] for k in ['id','source']},ensure_ascii=False)+'\n' for s in samples)
    completed=subprocess.run(command,input=data,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True,check=True);observations=[json.loads(x) for x in completed.stdout.splitlines()]
    (out/(name+'-prime-cleanup.log')).write_text(completed.stderr)
    if len(observations)!=4:raise ValueError('Incomplete actual warmup cleanup')
    for sample,observed in zip(samples,observations):
     if observed['id']!=sample['id'] or observed['source']!=sample['source'] or observed['cleanup']['version']!='speech-spans-v2':raise ValueError('Wrong actual warmup cleanup identity')
     request,prepared,cleanup=payloads.selection_payload(sample,observed['cleanup'])
     if sample['stage']=='format':request,encoding=payloads.format_payload(sample,cleanup['text'],prepared,{},[])
     response,elapsed,rss=self.generate(request);raw=response['choices'][0]['message']['content']
     row=dict(sample,variant='priming-'+sample['stage'],role='warmup',order=a.order,pipeline='metal-primed-v1',effectiveModelScene='spelling-selection' if sample['stage']=='selection' else sample['scene'],modelSha256=read(HERE/'freeze.json')['model']['sha256'],templateVersion='bounded-edit-v2-selection' if sample['stage']=='selection' else 'language-profile-v2-'+sample['scene'],request=request,prepared=prepared,cleanup=cleanup,cleanupObservation=observed,actualContextUsed=sample['context'] if sample['stage']=='selection' else {},response=response,renderedBody=None if sample['stage']=='selection' else raw,renderError=None,generationMs=elapsed,residentRssBytes=rss)
     if sample['stage']=='selection':
      edited,edits=payloads.candidates.apply(raw,prepared);row.update(selection=json.loads(raw),editedSource=edited,edits=edits)
     else:row.update(editedSource=cleanup['text'],edits=[],encoding=encoding)
     with (out/'warmups.jsonl').open('a') as stream:stream.write(json.dumps(row,ensure_ascii=False)+'\n')
     warm_rows.append(row);print('warmup',a.order,sample['id'],round(elapsed),repr(raw),flush=True)
    prime_ms=(time.monotonic()-start)*1000;self.startup_ms=load_ms+prime_ms;workers.append(dict(scene=scene,modelStartupMs=load_ms,primingMs=prime_ms,readyMs=self.startup_ms,warmupResponses=4))
   except Exception:self.abort();raise
 prior.HERE=HERE;prior.pipeline=payloads;prior.Session=PrimedSession;prior.engine.read=ordered_read;old_argv=sys.argv;sys.argv=[old_argv[0],'--out',str(a.out),'--cleanup-worker',str(a.cleanup_worker)]
 try:
  prior.main();m=read(a.out/'manifest.json');readout=dict(status='complete',order=a.order,freezeSha256=prior.engine.sha(HERE/'freeze.json'),measuredGenerations=m['generations'],warmupGenerations=len(warm_rows),actualGenerations=m['generations']+len(warm_rows),warmupResultsSha256=prior.engine.sha(a.out/'warmups.jsonl'),measuredResultsSha256=m['resultsSha256'],workers=workers,scope='All prefix priming and cleanup-worker startup costs recorded before measured cases; no native/ASR/concurrent/min16 acceptance.')
  prior.engine.write(a.out/'study.json',readout)
 except Exception as error:
  if a.out.exists():prior.engine.write(a.out/'study.json',dict(status='failed',order=a.order,warmupGenerations=len(warm_rows),error=str(error)))
  raise
 finally:sys.argv=old_argv;prior.engine.read=read
if __name__=='__main__':main()
