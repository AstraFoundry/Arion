"""Generate frozen prompt variants; score all delivery policies on identical raw proposals."""
import argparse
import importlib.util
import json
import pathlib
import time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('prompt_v2',HERE.parent/'scene-rewrite-prompt-v2/run.py')
v2=importlib.util.module_from_spec(spec);spec.loader.exec_module(v2)
engine=v2.engine

def check_freeze():
    frozen=engine.read(HERE/'freeze.json')
    for name,digest in frozen['files'].items():
        if engine.sha(HERE/name)!=digest:raise ValueError('Frozen input changed: '+name)
    for name,digest in frozen['dependencies'].items():
        if engine.sha(ROOT/name)!=digest:raise ValueError('Frozen dependency changed: '+name)
    v2.check_freeze();engine.check_freeze()

def template_path(variant,scene):
    if variant=='previous':
        return HERE.parent/f'context-repair-v1/{scene}-zh.json' if scene in ['work','email'] else HERE.parent/f'scene-rewrite-prompt-v2/templates/examples-{scene}-zh.json'
    return HERE/f'templates/{variant}-{scene}.json'

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','evaluation'],required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--selection',type=pathlib.Path);args=parser.parse_args()
    check_freeze()
    if (args.phase=='evaluation')!=bool(args.selection):raise ValueError('Evaluation requires frozen selection')
    if args.phase=='dev':
        samples=engine.read(HERE/'development.json');variants=['minimal','anchored']
    else:
        selection=engine.read(args.selection)
        if selection['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('Selection freeze mismatch')
        for name in ['results.jsonl','review.json']:
            if selection['dev'+('Results' if name=='results.jsonl' else 'Review')+'Sha256']!=engine.sha(args.selection.parent/name):raise ValueError('Selection evidence changed')
        samples=engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json');variants=['previous',selection['selected']]
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),selectionSha256=engine.sha(args.selection) if args.selection else None,provenance=engine.provenance())
    engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for variant in variants:
                for scene in ['work','email','prompt','chat']:
                    template=template_path(variant,scene)
                    worker=engine.Worker(out,f'{variant}-{scene}',template)
                    jobs=sorted([s for s in samples if s['scene']==scene],key=lambda s:bool(s['context']))
                    for sample in jobs:
                        start=time.monotonic()
                        response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context=sample['context'])
                        ms=(time.monotonic()-start)*1000;start=time.monotonic()
                        strict=engine.strict_check(sample['source'],response,sample['context'])
                        validation_ms=(time.monotonic()-start)*1000
                        row=dict(**sample,variant=variant,effectiveModelScene=scene,templateSha256=engine.sha(template),response=response,generationMs=ms,strictCheckMs=validation_ms,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1
                        print(args.phase,variant,sample['id'],strict,row['envelope'],round(ms),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
