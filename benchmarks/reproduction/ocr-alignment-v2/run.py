"""Run frozen orthographic-repair candidates with the existing verified local worker."""
import argparse
import importlib.util
import json
import pathlib
import time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('ungated_experiment',HERE.parent/'rewrite-no-validator-v1/run.py')
prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
engine=prior.engine

def check_freeze():
    frozen=engine.read(HERE/'freeze.json')
    for group,base in [('files',HERE),('dependencies',ROOT)]:
        for name,digest in frozen[group].items():
            if engine.sha(base/name)!=digest:raise ValueError('Changed frozen input or dependency: '+name)
    prior.check_freeze()

def template_path(variant,scene):
    if variant=='previous':return HERE.parent/f'rewrite-no-validator-v1/templates/anchored-{scene}.json'
    return HERE/f'templates/{variant}-{scene}.json'

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','evaluation'],required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--selection',type=pathlib.Path);args=parser.parse_args()
    check_freeze()
    if (args.phase=='evaluation')!=bool(args.selection):raise ValueError('Evaluation requires frozen development selection')
    variants=['aligned','staged'];samples=engine.read(HERE/'development.json')
    if args.phase=='evaluation':
        selection=engine.read(args.selection)
        if selection['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('Changed selection freeze')
        for name,key in [('results.jsonl','devResultsSha256'),('review.json','devReviewSha256')]:
            if selection[key]!=engine.sha(args.selection.parent/name):raise ValueError('Changed development selection evidence')
        variants=['previous',selection['selected']];samples=engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),selectionSha256=engine.sha(args.selection) if args.selection else None,provenance=engine.provenance())
    engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for variant in variants:
                for scene in ['work','email','prompt','chat']:
                    template=template_path(variant,scene);worker=engine.Worker(out,f'{variant}-{scene}',template)
                    for sample in sorted([s for s in samples if s['scene']==scene],key=lambda s:bool(s['context'])):
                        start=time.monotonic();response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context=sample['context']);generation_ms=(time.monotonic()-start)*1000
                        start=time.monotonic();strict=engine.strict_check(sample['source'],response,sample['context']);strict_ms=(time.monotonic()-start)*1000
                        row=dict(**sample,variant=variant,effectiveModelScene=scene,templateSha256=engine.sha(template),response=response,generationMs=generation_ms,strictCheckMs=strict_ms,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1
                        print(args.phase,variant,sample['id'],strict,row['envelope'],round(generation_ms),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
