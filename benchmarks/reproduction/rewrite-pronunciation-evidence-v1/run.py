"""Compare explicit pronunciation metadata with a matched guidance-only control."""
import argparse,json,pathlib,sys,time,subprocess
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
import pronunciation_messages as presentation
ROOT=engine.ROOT
engine.MODEL=ROOT/'.workspace/rewrite-small-models-v1/Qwen3.5-2B-Q4_K_M.gguf'
VARIANTS=['baseline','evidence']
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','evaluation'],required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--selection',type=pathlib.Path);args=parser.parse_args();frozen=engine.read(HERE/'freeze.json');engine.verify(frozen)
 if (args.phase=='evaluation')!=bool(args.selection):raise ValueError('Evaluation requires reviewed selection')
 if args.selection:
  decision=engine.read(args.selection)
  if decision['selected'] not in VARIANTS or decision['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('No qualified frozen package')
  for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
   if engine.sha(args.selection.parent/name)!=decision[key]:raise ValueError('Changed selected development evidence')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);worker=None;count=0;manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),model=frozen['model'],runtimeFiles=frozen['runtimeFiles'],workers=[],selectionSha256=engine.sha(args.selection) if args.selection else None);engine.write(out/'manifest.json',manifest)
 try:
  samples=engine.read(HERE/('development.json' if args.phase=='dev' else 'fresh.json'))
  with (out/'results.jsonl').open('x') as stream:
   for variant in VARIANTS:
    for scene in ['work','email','prompt','chat']:
     template_path=HERE/f'templates/{variant}-{scene}.json';template=engine.read(template_path);worker=engine.Session(out,variant+'-'+scene,18769);manifest['workers'].append(dict(variant=variant,scene=scene,command=worker.command,startupMs=worker.startup_ms))
     for sample in [s for s in samples if s['scene']==scene]:
      begin=time.monotonic();messages,evidence=presentation.messages(sample,template);payload=dict(messages=messages,temperature=0,seed=42,max_tokens=384,cache_prompt=False,stream=False);prepared=time.monotonic();response=engine.base.request(18769,'/v1/chat/completions',payload,timeout=60);finished=time.monotonic();rss=int(subprocess.check_output(['ps','-o','rss=','-p',str(worker.process.pid)],text=True).strip())*1024
      row=dict(**sample,variant=variant,effectiveModelScene=scene,templateSha256=engine.sha(template_path),modelSha256=frozen['model']['sha256'],request=payload,pronunciationEvidence=evidence,response=response,generationMs=(finished-prepared)*1000,preparationMs=(prepared-begin)*1000,textStageMs=(finished-begin)*1000,residentRssBytes=rss);stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(variant,sample['id'],round(row['generationMs']),[(e['sourceSpan'],e['status']) for e in evidence],flush=True)
     worker.close();worker=None
  manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
 except Exception as error:
  if worker:worker.abort()
  manifest.update(status='failed',error=str(error),completedGenerations=count);engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
