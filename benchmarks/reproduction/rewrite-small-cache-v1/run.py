"""Measure recurrent shared-prefix caching separately from prompt changes."""
import argparse,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
ROOT=engine.ROOT
TEMPLATES=HERE.parent/'rewrite-small-models-v1/templates'
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();frozen=engine.read(HERE/'freeze.json');engine.verify(frozen);out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);worker=None;count=0
 manifest=dict(status='running',phase='cache-diagnostic',freezeSha256=engine.sha(HERE/'freeze.json'),model=frozen['model'],runtimeFiles=frozen['runtimeFiles'],workers=[]);engine.write(out/'manifest.json',manifest)
 try:
  samples=engine.read(HERE/'cases.json');warmups={s['scene']:s for s in engine.read(HERE/'warmups.json')}
  with (out/'results.jsonl').open('x') as stream:
   for order in ['forward','reverse']:
    for cached in [False,True]:
     for scene in ['work','email']:
      variant=order+'-'+('cached' if cached else 'uncached');template_path=TEMPLATES/f'{scene}.json';template=engine.read(template_path);worker=engine.Session(out,variant+'-'+scene,18769);manifest['workers'].append(dict(variant=variant,scene=scene,command=worker.command,startupMs=worker.startup_ms));subset=[s for s in samples if s['scene']==scene];subset=subset if order=='forward' else list(reversed(subset))
      for sample in [warmups[scene]]+subset:
       generated=worker.generate(sample,template,cached);row=dict(**sample,variant=variant,order=order,cacheEnabled=cached,role='warmup' if sample['id'].startswith('cache-warmup-') else 'measured',effectiveModelScene=scene,templateSha256=engine.sha(template_path),modelSha256=frozen['model']['sha256'],**generated);stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(variant,sample['id'],round(row['generationMs']),row['response']['timings']['cache_n'],flush=True)
      worker.close();worker=None
  manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
 except Exception as error:
  if worker:worker.abort()
  manifest.update(status='failed',error=str(error),completedGenerations=count);engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
