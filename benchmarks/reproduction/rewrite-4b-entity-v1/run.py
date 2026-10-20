"""Compare entity-policy wording with fixed JSON packaging and model export."""
import argparse,importlib.util,json,pathlib,time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('larger_packaging',HERE.parent/'rewrite-4b-packaging-v1/run.py')
packaging=importlib.util.module_from_spec(spec);spec.loader.exec_module(packaging)
engine=packaging.engine
VARIANTS=['baseline','exact']
def check_freeze():
    frozen=engine.read(HERE/'freeze.json')
    for name,digest in frozen['files'].items():
        if engine.sha(ROOT/name)!=digest:raise ValueError('Changed frozen input: '+name)
    original=packaging.check_freeze()
    if frozen['modelFiles']!=original['modelFiles']:raise ValueError('Changed model')
    return frozen

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','evaluation'],required=True);parser.add_argument('--selection',type=pathlib.Path);parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();frozen=check_freeze()
    if (args.phase=='evaluation')!=bool(args.selection):raise ValueError('Evaluation requires passing reviewed selection')
    samples=engine.read(HERE/'cases.json')
    if args.phase=='evaluation':
        decision=engine.read(args.selection)
        if not decision['qualifiedForFresh'] or decision['selected']!='exact':raise ValueError('No qualified candidate')
        if decision['freezeSha256']!=engine.sha(HERE/'freeze.json') or decision['devResultsSha256']!=engine.sha(args.selection.parent/'results.jsonl') or decision['devReviewSha256']!=engine.sha(args.selection.parent/'review.json'):raise ValueError('Changed frozen selection')
        samples=engine.read(HERE/'fresh.json')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    provenance=engine.provenance();provenance['productionReference']=provenance.pop('model');provenance.update(model=engine.read(HERE/'model-provenance.json')['challenger'],modelFiles=frozen['modelFiles'],packagingBuild=frozen['build'],packagingWorkerSha256=engine.sha(packaging.screen.packaging.BUILD/'worker'))
    manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),selectionSha256=engine.sha(args.selection) if args.selection else None,provenance=provenance);engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for variant in VARIANTS:
                for scene in ['work','email']:
                    template=HERE/f'templates/{variant}-{scene}.json';worker=packaging.Worker(out,f'{variant}-{scene}',template,'json',pathlib.Path(frozen['modelPath']))
                    for sample in sorted([s for s in samples if s['scene']==scene],key=lambda s:bool(s['context'])):
                        start=time.monotonic();response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context=sample['context']);elapsed=(time.monotonic()-start)*1000
                        strict=engine.strict_check(sample['source'],response,sample['context']);row=dict(**sample,variant=variant,effectiveModelScene=scene,templateSha256=engine.sha(template),modelFiles=frozen['modelFiles'],response=response,generationMs=elapsed,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(args.phase,variant,sample['id'],strict,round(elapsed),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error),completedGenerations=count);engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
