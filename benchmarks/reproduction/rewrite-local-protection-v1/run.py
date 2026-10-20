"""Run latest small-model direct-body and local-encoding packages independently."""
import argparse,json,pathlib,sys,time,subprocess
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
import codec
ROOT=engine.ROOT
ARMS={'qwen-direct':('Qwen3.5-2B-Q4_K_M.gguf',False),'qwen-protected':('Qwen3.5-2B-Q4_K_M.gguf',True),'granite-direct':('granite-4.0-h-1b-Q4_K_M.gguf',False)}
def payload(sample,arm):
 messages,encoding=codec.messages(sample,engine.read(HERE/f"templates/{sample['scene']}.json"),ARMS[arm][1])
 return dict(messages=messages,temperature=0,seed=42,max_tokens=384,cache_prompt=False,stream=False),encoding

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--arm',choices=ARMS,required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--phase',choices=['dev','evaluation'],default='dev');parser.add_argument('--selection',type=pathlib.Path);args=parser.parse_args();frozen=engine.read(HERE/'freeze.json');filename=ARMS[args.arm][0];model=frozen['models'][filename];engine.MODEL=ROOT/frozen['modelPaths'][filename];engine.verify(dict(frozen,model=model))
 if args.phase=='evaluation':
  if args.selection is None:raise ValueError('No reviewed development selection')
  decision=engine.read(args.selection)
  if not decision['qualified'] or decision['selected'] not in ARMS or args.arm not in ['qwen-direct',decision['selected']] or decision['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('No qualified frozen package')
  for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
   if engine.sha(args.selection.parent/name)!=decision[key]:raise ValueError('Changed selected evidence')
 elif args.selection:raise ValueError('Development has no selection')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);worker=None;count=0;manifest=dict(status='running',phase=args.phase,arm=args.arm,freezeSha256=engine.sha(HERE/'freeze.json'),model=model,runtimeFiles=frozen['runtimeFiles'],workers=[]);engine.write(out/'manifest.json',manifest)
 try:
  samples=engine.read(HERE/('development.json' if args.phase=='dev' else 'fresh.json'))
  with (out/'results.jsonl').open('x') as stream:
   for scene in ['work','email','prompt','chat']:
    worker=engine.Session(out,args.arm+'-'+scene,18769);manifest['workers'].append(dict(variant=args.arm,scene=scene,command=worker.command,startupMs=worker.startup_ms))
    for sample in [s for s in samples if s['scene']==scene]:
     start=time.monotonic();request,encoding=payload(sample,args.arm);ready=time.monotonic();response=engine.base.request(18769,'/v1/chat/completions',request,timeout=90);finished=time.monotonic();raw=response['choices'][0]['message']['content'];error=None
     try:body=codec.decode(raw,encoding)
     except ValueError as failure:body=None;error=str(failure)
     rendered=time.monotonic();rss=int(subprocess.check_output(['ps','-o','rss=','-p',str(worker.process.pid)],text=True).strip())*1024
     row=dict(**sample,variant=args.arm,effectiveModelScene=scene,templateSha256=engine.sha(HERE/f'templates/{scene}.json'),modelSha256=model['sha256'],request=request,encoding=encoding,response=response,renderedBody=body,renderError=error,preparationMs=(ready-start)*1000,generationMs=(finished-ready)*1000,renderMs=(rendered-finished)*1000,textStageMs=(rendered-start)*1000,residentRssBytes=rss);stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(args.arm,sample['id'],round(row['generationMs']),repr(body),error,flush=True)
    worker.close();worker=None
  manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
 except Exception as error:
  if worker:worker.abort()
  manifest.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
