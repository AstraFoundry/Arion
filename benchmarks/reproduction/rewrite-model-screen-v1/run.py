"""Run a bounded comparison of existing local checkpoint/export bundles."""
import argparse,importlib.util,json,os,pathlib,subprocess,time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('packaging',HERE.parent/'rewrite-packaging-v1/run.py')
packaging=importlib.util.module_from_spec(spec);spec.loader.exec_module(packaging)
engine=packaging.engine
VARIANTS=['production-1.7b','prior-4b-export']
class Worker(engine.Worker):
    def __init__(self,out,name,template,model):
        self.seq=0
        env=dict(os.environ,ORT_LIB_PATH=str(engine.RUNTIME),ARION_EXPERIMENT_TEMPLATE=str(template),ARION_EXPERIMENT_FORMAT='labeled')
        self.stderr=(out/(name+'-stderr.log')).open('x')
        self.p=subprocess.Popen([str(packaging.BUILD/'worker')],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.stderr,text=True,bufsize=1,env=env)
        try:
            self.load=self.ask('load',timeout=90,runtime=str(engine.RUNTIME),model=str(model),threads=4,maxLength=4096,maxNewTokens=384)
            engine.write(out/(name+'-load.json'),self.load)
        except Exception:
            self.p.kill();self.p.wait();self.stderr.close();raise

def check_freeze():
    frozen=engine.read(HERE/'freeze.json')
    for name,digest in frozen['files'].items():
        if engine.sha(ROOT/name)!=digest:raise ValueError('Changed frozen input: '+name)
    packaging.check_freeze()
    for name,digest in frozen['runtimeFiles'].items():
        if engine.sha(pathlib.Path(name))!=digest:raise ValueError('Changed runtime/worker: '+name)
    if engine.read(packaging.BUILD/'manifest.json')!=frozen['build']:raise ValueError('Changed shared worker')
    for name,digest in frozen['modelFiles']['prior-4b-export'].items():
        if engine.sha(pathlib.Path(frozen['modelPaths']['prior-4b-export'])/name)!=digest:raise ValueError('Changed challenger file: '+name)
    return frozen

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();frozen=check_freeze()
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    provenance=engine.provenance();reference=provenance.pop('model');actual=provenance.pop('modelFiles')
    if actual!=frozen['modelFiles']['production-1.7b']:raise ValueError('Changed provisioned production model')
    provenance.update(productionReference=reference,modelFiles=frozen['modelFiles'],modelPaths=frozen['modelPaths'],modelDescriptions=engine.read(HERE/'model-provenance.json'),packagingBuild=engine.read(packaging.BUILD/'manifest.json'),packagingWorkerSha256=engine.sha(packaging.BUILD/'worker'))
    manifest=dict(status='running',phase='development',freezeSha256=engine.sha(HERE/'freeze.json'),provenance=provenance);engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        samples=engine.read(HERE/'cases.json')
        with (out/'results.jsonl').open('x') as stream:
            for variant in VARIANTS:
                for scene in ['work','email']:
                    template=HERE/f'templates/{scene}.json';worker=Worker(out,f'{variant}-{scene}',template,pathlib.Path(frozen['modelPaths'][variant]))
                    for sample in sorted([s for s in samples if s['scene']==scene],key=lambda s:bool(s['context'])):
                        start=time.monotonic();response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context=sample['context']);elapsed=(time.monotonic()-start)*1000
                        strict=engine.strict_check(sample['source'],response,sample['context'])
                        row=dict(**sample,variant=variant,effectiveModelScene=scene,templateSha256=engine.sha(template),modelFiles=frozen['modelFiles'][variant],response=response,generationMs=elapsed,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1
                        print(variant,sample['id'],strict,round(elapsed),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error),completedGenerations=count);engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
