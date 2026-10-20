"""Measure formatter intervention using pinned actual v2 selector responses."""
import pathlib,sys,json,argparse,time,subprocess
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
import pipeline
ROOT=engine.ROOT
def rowhash(row):return __import__('hashlib').sha256(json.dumps(row,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();freeze=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/freeze['modelPath'];engine.verify(freeze)
 protocol=engine.read(HERE/'protocol.json');reference=ROOT/protocol['selectionReference'];previous=[json.loads(x) for x in reference.read_text().splitlines() if json.loads(x)['stage']=='selection'];lookup={r['id']:r for r in previous}
 samples=engine.read(HERE.parent/'rewrite-bounded-edit-v2/development.json');out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);manifest=dict(status='running',phase='dev',freezeSha256=engine.sha(HERE/'freeze.json'),model=freeze['model'],runtimeFiles=freeze['runtimeFiles'],workers=[],pipeline='formatter-v3',newGenerations=36,reusedSelections=36,selectionReference=protocol['selectionReference'],selectionReferenceSha256=engine.sha(reference));engine.write(out/'manifest.json',manifest);worker=None;count=0
 try:
  with (out/'results.jsonl').open('x') as stream:
   for scene in ['work','email','prompt','chat']:
    worker=engine.Session(out,'format-'+scene,18789);manifest['workers'].append(dict(scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for s in [s for s in samples if s['scene']==scene]:
     selected=lookup[s['id']]
     if any(selected[k]!=v for k,v in s.items()):raise ValueError('Changed selector source identity')
     start=time.monotonic();request,encoding=pipeline.format_payload(s,selected['editedSource'],selected['prepared'],selected['selection'],selected['edits']);ready=time.monotonic();response=engine.base.request(18789,'/v1/chat/completions',request,timeout=90);finished=time.monotonic();raw=response['choices'][0]['message']['content'];body=pipeline.candidates.restore(raw,encoding);end=time.monotonic()
     row=dict(**s,variant='formatter-v3',stage='format',pipeline='formatter-v3',effectiveModelScene=scene,templateVersion='formatter-v3-'+scene,templateSha256=engine.sha(HERE/'prompts.json'),modelSha256=freeze['model']['sha256'],request=request,actualContextUsed={},selectionReference=protocol['selectionReference'],selectionRowSha256=rowhash(selected),cleanup=selected['cleanup'],editedSource=selected['editedSource'],edits=selected['edits'],encoding=encoding,response=response,renderedBody=body,renderError=None,preparationMs=(ready-start)*1000,generationMs=(finished-ready)*1000,renderMs=(end-finished)*1000,textStageMs=(end-start)*1000,residentRssBytes=int(subprocess.check_output(['ps','-o','rss=','-p',str(worker.process.pid)],text=True).strip())*1024)
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(s['id'],round(row['generationMs']),repr(body),flush=True)
    worker.close();worker=None
  manifest.update(status='complete',generations=count,sourceCases=36,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
 except Exception as error:
  if worker:worker.abort()
  manifest.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
