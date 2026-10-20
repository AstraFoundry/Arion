"""Execute frozen formatter jobs in arm-scoped native sessions without output filtering."""
import pathlib,argparse,json
import gguf_rewrite as engine
from gguf_session_v2 import Session
def execute(here,jobs,metadata):
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();f=engine.read(here/'freeze.json');engine.MODEL=engine.ROOT/f['modelPath'];engine.verify(f);out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);m=dict(metadata,status='running',freezeSha256=engine.sha(here/'freeze.json'),model=f['model'],runtimeFiles=f['runtimeFiles'],workers=[]);engine.write(out/'manifest.json',m);worker=None;count=0
 try:
  with (out/'results.jsonl').open('x') as stream:
   for arm in dict.fromkeys(job['variant'] for job in jobs):
    name=metadata['pipeline']+'-'+arm;worker=Session(out,name,18820,gpu_layers=99,verbosity=5);m['workers'].append(dict(arm=arm,name=name,command=worker.command,startupMs=worker.startup_ms))
    for job in (j for j in jobs if j['variant']==arm):
     response,elapsed,rss=worker.generate(job['request']);body=response['choices'][0]['message']['content'];row=dict(job,response=response,renderedBody=body,renderError=None,generationMs=elapsed,residentRssBytes=rss)
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(arm,row['id'],round(elapsed),repr(body),flush=True)
    worker.close();worker=None
  m.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  m.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',m);raise
