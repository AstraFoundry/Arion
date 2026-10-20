"""Reuse the verified production-engine experiment worker; never change old evidence."""
import argparse
import hashlib
import importlib.util
import json
import pathlib
import random
import time

HERE=pathlib.Path(__file__).resolve().parent
ROOT=next(p for p in HERE.parents if (p/'models/catalog.json').exists())
spec=importlib.util.spec_from_file_location('alignment_v1',ROOT/'benchmarks/reproduction/scene-rewrite-alignment-v1/study.py')
engine=importlib.util.module_from_spec(spec);spec.loader.exec_module(engine)
def check_freeze():
 for name,h in engine.read(HERE/'freeze.json')['files'].items():
  if engine.sha(HERE/name)!=h:raise ValueError('Frozen input changed: '+name)

def run():
 p=argparse.ArgumentParser();p.add_argument('--phase',choices=['dev','evaluation'],required=True);p.add_argument('--out',type=pathlib.Path,required=True);p.add_argument('--selection',type=pathlib.Path);a=p.parse_args()
 check_freeze();engine.check_freeze()
 if (a.phase=='evaluation')!=bool(a.selection):raise ValueError('Evaluation requires frozen development selection')
 out=a.out.resolve();out.mkdir(parents=True,exist_ok=False)
 manifest=dict(status='running',phase=a.phase,provenance=engine.provenance(),freezeSha256=engine.sha(HERE/'freeze.json'),selectionSha256=engine.sha(a.selection) if a.selection else None)
 engine.write(out/'manifest.json',manifest)
 if a.phase=='dev':samples=engine.read(HERE/'development.json');variants=['focused','examples']
 else:
  selection=engine.read(a.selection)
  if selection['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('Selection freeze mismatch')
  samples=engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json');variants=['production',selection['selected']]
 worker=None;count=0
 try:
  with (out/'results.jsonl').open('x') as f:
   for variant in variants:
    # Keep each scene/language template resident for its group; loads are measured separately.
    groups=[('all','all',samples)] if variant=='production' else [(scene,lang,[s for s in samples if s['scene']==scene and s['language']==lang]) for scene in ['work','prompt','email','chat'] for lang in ['zh','en']]
    for scene,lang,jobs in groups:
     template=None if variant=='production' else HERE/f'templates/{variant}-{scene}-{lang}.json'
     worker=engine.Worker(out,f'{variant}-{scene}-{lang}',template)
     jobs=list(jobs);random.Random(1729).shuffle(jobs)
     # Explicit context stratum avoids repeated plain/context prefix thrashing in the baseline.
     jobs.sort(key=lambda s:bool(s['context']))
     for sample in jobs:
      modelscene=sample['scene']
      if variant=='production':modelscene='email' if modelscene=='email' else 'chat'
      start=time.monotonic()
      response=worker.ask('rewrite',session=sample['id'],scene=modelscene,text=sample['source'],context=sample['context'])
      ms=(time.monotonic()-start)*1000
      strict=engine.strict_check(sample['source'],response,sample['context'])
      envelope=engine.envelope(response,strict)
      row=dict(id=sample['id'],variant=variant,source=sample['source'],scene=sample['scene'],language=sample['language'],cohort=sample['cohort'],context=sample['context'],effectiveModelScene=modelscene,templateSha256=engine.sha(template) if template else None,response=response,generationMs=ms,strict=strict,envelope=envelope)
      f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush();count+=1
      print(a.phase,variant,sample['id'],strict,envelope,round(ms),flush=True)
     worker.close();worker=None
  manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
 except Exception as e:
  if worker:worker.p.kill();worker.p.wait()
  manifest.update(status='failed',error=type(e).__name__+': '+str(e));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':run()
