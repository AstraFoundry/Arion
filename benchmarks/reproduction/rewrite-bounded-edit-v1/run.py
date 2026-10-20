"""Run every source through actual bounded selection and natural formatting inference."""
import pathlib,sys,json,time,subprocess,argparse,hashlib
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
import pipeline
ROOT=engine.ROOT
def rowhash(row):return hashlib.sha256(json.dumps(row,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def rss(pid):return int(subprocess.check_output(['ps','-o','rss=','-p',str(pid)],text=True).strip())*1024
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();frozen=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/frozen['modelPath'];engine.verify(frozen)
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);manifest=dict(status='running',phase='dev',freezeSha256=engine.sha(HERE/'freeze.json'),model=frozen['model'],runtimeFiles=frozen['runtimeFiles'],workers=[],pipeline='bounded-edit-v1');engine.write(out/'manifest.json',manifest);worker=None;count=0
 try:
  with (out/'results.jsonl').open('x') as stream:
   for scene in ['work','email','prompt','chat']:
    worker=engine.Session(out,'staged-'+scene,18769);manifest['workers'].append(dict(scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for sample in [s for s in engine.read(HERE/'development.json') if s['scene']==scene]:
     case_started=time.monotonic();request,prepared,cleanup=pipeline.selection_payload(sample);ready=time.monotonic();response=engine.base.request(18769,'/v1/chat/completions',request,timeout=90);finished=time.monotonic();raw=response['choices'][0]['message']['content'];stage_error=None;edited=None;edits=None;selection=None
     try:edited,edits=pipeline.candidates.apply(raw,prepared);selection=json.loads(raw)
     except (ValueError,KeyError,TypeError) as error:stage_error=type(error).__name__+': '+str(error)
     rendered=time.monotonic()
     row=dict(**sample,variant='bounded-q4-select',stage='selection',pipeline='bounded-edit-v1',effectiveModelScene='spelling-selection',templateVersion='bounded-edit-v1-selection',templateSha256=engine.sha(HERE/'prompts.json'),modelSha256=frozen['model']['sha256'],request=request,prepared=prepared,cleanup=cleanup,actualContextUsed=sample['context'],response=response,selection=selection,editedSource=edited,edits=edits,renderedBody=None,renderError=stage_error,preparationMs=(ready-case_started)*1000,generationMs=(finished-ready)*1000,renderMs=(rendered-finished)*1000,textStageMs=(rendered-case_started)*1000,residentRssBytes=rss(worker.process.pid),coordinatorRssBytes=rss(__import__('os').getpid()))
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print('selection',sample['id'],round(row['generationMs']),repr(raw),'edits',edits,stage_error,flush=True)
     if stage_error:raise ValueError('Recorded invalid selection; no invented formatter result: '+sample['id'])
     selection_hash=rowhash(row);start=time.monotonic();request,encoding=pipeline.format_payload(sample,edited,prepared,selection,edits);ready=time.monotonic();response=engine.base.request(18769,'/v1/chat/completions',request,timeout=90);finished=time.monotonic();raw=response['choices'][0]['message']['content'];body=pipeline.candidates.restore(raw,encoding);rendered=time.monotonic()
     row=dict(**sample,variant='bounded-q4-format',stage='format',pipeline='bounded-edit-v1',effectiveModelScene=scene,templateVersion='bounded-edit-v1-'+scene,templateSha256=engine.sha(HERE/'prompts.json'),modelSha256=frozen['model']['sha256'],request=request,actualContextUsed={},selectionRowSha256=selection_hash,cleanup=cleanup,editedSource=edited,edits=edits,encoding=encoding,response=response,renderedBody=body,renderError=None,preparationMs=(ready-start)*1000,generationMs=(finished-ready)*1000,renderMs=(rendered-finished)*1000,textStageMs=(rendered-start)*1000,pipelineMs=(rendered-case_started)*1000,residentRssBytes=rss(worker.process.pid),coordinatorRssBytes=rss(__import__('os').getpid()))
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print('format',sample['id'],round(row['generationMs']),repr(body),flush=True)
    worker.close();worker=None
  manifest.update(status='complete',generations=count,sourceCases=36,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
 except Exception as error:
  if worker:worker.abort()
  manifest.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
