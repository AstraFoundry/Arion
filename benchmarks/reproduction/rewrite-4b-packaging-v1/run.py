"""Compare serializers on the same pinned larger local model bundle."""
import argparse,importlib.util,json,os,pathlib,subprocess,time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('model_screen',HERE.parent/'rewrite-model-screen-v1/run.py')
screen=importlib.util.module_from_spec(spec);spec.loader.exec_module(screen)
engine=screen.engine
FORMATS=['labeled','json','xml']
class Worker(engine.Worker):
    def __init__(self,out,name,template,format,model):
        self.seq=0;env=dict(os.environ,ORT_LIB_PATH=str(engine.RUNTIME),ARION_EXPERIMENT_TEMPLATE=str(template),ARION_EXPERIMENT_FORMAT=format)
        self.stderr=(out/(name+'-stderr.log')).open('x')
        self.p=subprocess.Popen([str(screen.packaging.BUILD/'worker')],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.stderr,text=True,bufsize=1,env=env)
        try:
            self.load=self.ask('load',timeout=90,runtime=str(engine.RUNTIME),model=str(model),threads=4,maxLength=4096,maxNewTokens=384);engine.write(out/(name+'-load.json'),self.load)
        except Exception:
            self.p.kill();self.p.wait();self.stderr.close();raise

def check_freeze():
    frozen=engine.read(HERE/'freeze.json')
    for name,digest in frozen['files'].items():
        if engine.sha(ROOT/name)!=digest:raise ValueError('Changed frozen input: '+name)
    source=screen.check_freeze()
    if source['modelFiles']['prior-4b-export']!=frozen['modelFiles']:raise ValueError('Changed common model')
    return frozen

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','evaluation'],required=True);parser.add_argument('--selection',type=pathlib.Path);parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();frozen=check_freeze()
    if (args.phase=='evaluation')!=bool(args.selection):raise ValueError('Evaluation requires a reviewed passing development selection')
    formats=FORMATS;samples=engine.read(HERE/'cases.json')
    if args.phase=='evaluation':
        decision=engine.read(args.selection)
        if not decision['qualifiedForFresh'] or decision['selected'] not in FORMATS:raise ValueError('No qualified format')
        if decision['freezeSha256']!=engine.sha(HERE/'freeze.json') or decision['devResultsSha256']!=engine.sha(args.selection.parent/'results.jsonl') or decision['devReviewSha256']!=engine.sha(args.selection.parent/'review.json'):raise ValueError('Changed selection evidence')
        formats=['labeled',decision['selected']]
        if formats[0]==formats[1]:raise ValueError('No improved format selected')
        samples=engine.read(HERE/'fresh.json')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    provenance=engine.provenance();provenance['productionReference']=provenance.pop('model');provenance['modelFiles']=frozen['modelFiles'];provenance['model']=engine.read(HERE/'model-provenance.json')['challenger'];provenance['packagingBuild']=frozen['build'];provenance['packagingWorkerSha256']=engine.sha(screen.packaging.BUILD/'worker')
    manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),selectionSha256=engine.sha(args.selection) if args.selection else None,provenance=provenance);engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for format in formats:
                for scene in ['work','email']:
                    template=HERE/f'templates/{scene}.json';worker=Worker(out,f'{format}-{scene}',template,format,pathlib.Path(frozen['modelPath']))
                    for sample in sorted([s for s in samples if s['scene']==scene],key=lambda s:bool(s['context'])):
                        start=time.monotonic();response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context=sample['context']);elapsed=(time.monotonic()-start)*1000
                        strict=engine.strict_check(sample['source'],response,sample['context']);row=dict(**sample,variant=format,effectiveModelScene=scene,templateSha256=engine.sha(template),modelFiles=frozen['modelFiles'],response=response,generationMs=elapsed,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(args.phase,format,sample['id'],strict,round(elapsed),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error),completedGenerations=count);engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
