"""Run clean-demonstration and instruction-language diagnostics on labeled inputs."""
import argparse
import importlib.util
import json
import pathlib
import time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('packaging',HERE.parent/'rewrite-packaging-v1/run.py')
packaging=importlib.util.module_from_spec(spec);spec.loader.exec_module(packaging)
engine=packaging.engine
ARMS=['baseline','clean-en','clean-zh']

def check_freeze():
    frozen=engine.read(HERE/'freeze.json')
    for name,digest in frozen['files'].items():
        if engine.sha(HERE/name)!=digest:raise ValueError('Changed study input: '+name)
    for name,digest in frozen['dependencies'].items():
        if engine.sha(ROOT/name)!=digest:raise ValueError('Changed dependency: '+name)
    packaging.check_freeze()

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','evaluation'],required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--selection',type=pathlib.Path);args=parser.parse_args();check_freeze()
    if (args.phase=='evaluation')!=bool(args.selection):raise ValueError('Evaluation requires frozen selection')
    samples=engine.read(HERE/'development.json');arms=ARMS
    if args.phase=='evaluation':
        selected=engine.read(args.selection)
        if selected['freezeSha256']!=engine.sha(HERE/'freeze.json') or selected['devResultsSha256']!=engine.sha(args.selection.parent/'results.jsonl') or selected['devReviewSha256']!=engine.sha(args.selection.parent/'review.json'):raise ValueError('Changed selection evidence')
        arms=['baseline',selected['selected']]
        if arms[1] not in ARMS[1:]:raise ValueError('Explicit diagnostic alternate required')
        samples=engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    provenance=engine.provenance();provenance['packagingBuild']=engine.read(packaging.BUILD/'manifest.json');provenance['packagingWorkerSha256']=engine.sha(packaging.BUILD/'worker')
    manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),selectionSha256=engine.sha(args.selection) if args.selection else None,provenance=provenance)
    engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for arm in arms:
                for scene in ['work','email','prompt','chat']:
                    template=HERE/f'templates/{arm}-{scene}.json';worker=packaging.Worker(out,f'{arm}-{scene}',template,'labeled')
                    for sample in sorted([s for s in samples if s['scene']==scene],key=lambda s:bool(s['context'])):
                        start=time.monotonic();response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context=sample['context']);elapsed=(time.monotonic()-start)*1000
                        strict=engine.strict_check(sample['source'],response,sample['context'])
                        row=dict(**sample,variant=arm,effectiveModelScene=scene,templateSha256=engine.sha(template),response=response,generationMs=elapsed,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1
                        print(args.phase,arm,sample['id'],strict,round(elapsed),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
