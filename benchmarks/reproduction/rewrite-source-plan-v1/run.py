"""Run a matched free-body versus grammar-bound source-plan package diagnostic."""
import argparse,json,pathlib,sys,time,subprocess
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
import plan
ROOT=engine.ROOT
engine.MODEL=ROOT/'.workspace/rewrite-small-models-v1/Qwen3.5-2B-Q4_K_M.gguf'
def payload(sample,variant):
 if variant=='plan':return plan.payload(sample)
 template=engine.read(HERE/f"templates/baseline-{sample['scene']}.json")
 return dict(messages=engine.base.messages(sample,template),temperature=0,seed=42,max_tokens=384,cache_prompt=False,stream=False),None
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();frozen=engine.read(HERE/'freeze.json');engine.verify(frozen);out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);worker=None;count=0
 manifest=dict(status='running',phase='dev',freezeSha256=engine.sha(HERE/'freeze.json'),model=frozen['model'],runtimeFiles=frozen['runtimeFiles'],workers=[]);engine.write(out/'manifest.json',manifest)
 try:
  with (out/'results.jsonl').open('x') as stream:
   for variant in ['baseline','plan']:
    for scene in ['work','email','prompt','chat']:
     worker=engine.Session(out,variant+'-'+scene,18769);manifest['workers'].append(dict(variant=variant,scene=scene,command=worker.command,startupMs=worker.startup_ms))
     for sample in [s for s in engine.read(HERE/'development.json') if s['scene']==scene]:
      start=time.monotonic();request,prepared=payload(sample,variant);ready=time.monotonic();response=engine.base.request(18769,'/v1/chat/completions',request,timeout=90);finished=time.monotonic();raw=response['choices'][0]['message']['content'];render_error=None
      try:body=plan.render(raw,prepared) if variant=='plan' else raw
      except (ValueError,KeyError,TypeError) as error:body=None;render_error=type(error).__name__+': '+str(error)
      rendered=time.monotonic();rss=int(subprocess.check_output(['ps','-o','rss=','-p',str(worker.process.pid)],text=True).strip())*1024
      row=dict(**sample,variant=variant,effectiveModelScene=scene,packageSha256=engine.sha(HERE/('plan.py' if variant=='plan' else f'templates/baseline-{scene}.json')),request=request,preparedPlan=prepared,response=response,renderedBody=body,renderError=render_error,preparationMs=(ready-start)*1000,generationMs=(finished-ready)*1000,renderMs=(rendered-finished)*1000,textStageMs=(rendered-start)*1000,residentRssBytes=rss);stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(variant,sample['id'],round(row['generationMs']),repr(body),render_error,flush=True)
     worker.close();worker=None
  manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
 except Exception as error:
  if worker:worker.abort()
  manifest.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
