"""Compare entity evidence policies with authored and actual Vision contexts."""
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
ARMS=['baseline','exact']
CORPUS=ROOT/'benchmarks/corpora/entity-ocr-v1'
def check_freeze():
    frozen=engine.read(HERE/'freeze.json')
    for name,digest in frozen['files'].items():
        if engine.sha(ROOT/name)!=digest:raise ValueError('Changed input: '+name)
    packaging.check_freeze()
    if engine.read(packaging.BUILD/'manifest.json')!=frozen['build']:raise ValueError('Changed common build')
def samples_for(phase,ocr=None):
    if phase=='dev':return engine.read(HERE/'development.json')
    samples={s['id']:s for s in engine.read(CORPUS/'corpus.json')['samples']}
    observations=[json.loads(x) for x in ocr.read_text().splitlines()]
    expected={(i,l) for i in samples for l in ['accurate','fast']}
    keys=[(r['caseId'],r['level']) for r in observations]
    if len(keys)!=len(expected) or set(keys)!=expected or any(r.get('error') for r in observations):raise ValueError('Incomplete or failed OCR observations')
    return [dict(id=r['caseId']+'-'+r['level'],ocrCaseId=r['caseId'],ocrLevel=r['level'],scene=samples[r['caseId']]['scene'],source=samples[r['caseId']]['source'],context={'Screen':r['text']},cohort='actual-vision-'+r['level'],language='zh') for r in observations]
def main():
    parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','ocr'],required=True);parser.add_argument('--ocr',type=pathlib.Path);parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();check_freeze()
    if (args.phase=='ocr')!=bool(args.ocr):raise ValueError('OCR phase requires observations')
    samples=samples_for(args.phase,args.ocr);out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    provenance=engine.provenance();provenance['packagingBuild']=engine.read(packaging.BUILD/'manifest.json');provenance['packagingWorkerSha256']=engine.sha(packaging.BUILD/'worker')
    manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),ocrSha256=engine.sha(args.ocr) if args.ocr else None,provenance=provenance)
    engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for arm in ARMS:
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
