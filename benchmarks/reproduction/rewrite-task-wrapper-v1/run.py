"""Compare source/reference packages with fixed templates, weights and decoding budget."""
import argparse
import importlib.util
import json
import os
import pathlib
import subprocess
import time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
spec=importlib.util.spec_from_file_location('ocr_alignment',HERE.parent/'ocr-alignment-v2/run.py')
previous=importlib.util.module_from_spec(spec);spec.loader.exec_module(previous)
engine=previous.engine
BUILD=ROOT/'.workspace/rewrite-task-wrapper-v1/build'
FORMATS=('labeled','tasked')

class Worker(engine.Worker):
    def __init__(self,out,name,template,packaging):
        self.seq=0
        env=dict(os.environ,ORT_LIB_PATH=str(engine.RUNTIME),ARION_EXPERIMENT_TEMPLATE=str(template),ARION_EXPERIMENT_FORMAT=packaging)
        self.stderr=(out/(name+'-stderr.log')).open('x')
        self.p=subprocess.Popen([str(BUILD/'worker')],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.stderr,text=True,bufsize=1,env=env)
        self.load=self.ask('load',timeout=90,runtime=str(engine.RUNTIME),model=str(engine.MODEL),threads=4,maxLength=4096,maxNewTokens=384)
        engine.write(out/(name+'-load.json'),self.load)

def check_freeze():
    data=engine.read(HERE/'freeze.json')
    for group,base in [('files',HERE),('dependencies',ROOT)]:
        for name,digest in data[group].items():
            if engine.sha(base/name)!=digest:raise ValueError('Changed frozen input: '+name)
    previous.check_freeze()

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--phase',choices=['dev','evaluation'],required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--selection',type=pathlib.Path);args=parser.parse_args()
    check_freeze()
    if (args.phase=='evaluation')!=bool(args.selection):raise ValueError('Evaluation requires reviewed selection')
    cases=engine.read(HERE/'development.json');variants=list(FORMATS)
    if args.phase=='evaluation':
        selection=engine.read(args.selection)
        if selection['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('Changed selected freeze')
        for name,key in [('results.jsonl','devResultsSha256'),('review.json','devReviewSha256')]:
            if selection[key]!=engine.sha(args.selection.parent/name):raise ValueError('Changed development evidence')
        variants=['labeled',selection['selected']]
        if variants[0]==variants[1]:raise ValueError('No improved package selected; freeze a diagnostic alternate explicitly')
        cases=engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    provenance=engine.provenance();provenance['packagingBuild']=engine.read(BUILD/'manifest.json');provenance['packagingWorkerSha256']=engine.sha(BUILD/'worker')
    manifest=dict(status='running',phase=args.phase,freezeSha256=engine.sha(HERE/'freeze.json'),selectionSha256=engine.sha(args.selection) if args.selection else None,provenance=provenance)
    engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for variant in variants:
                for scene in ['work','email','prompt','chat']:
                    template=HERE/f'templates/{scene}.json';worker=Worker(out,f'{variant}-{scene}',template,variant)
                    for sample in sorted([s for s in cases if s['scene']==scene],key=lambda s:bool(s['context'])):
                        start=time.monotonic();response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context=sample['context']);elapsed=(time.monotonic()-start)*1000
                        strict=engine.strict_check(sample['source'],response,sample['context'])
                        row=dict(**sample,variant=variant,effectiveModelScene=scene,templateSha256=engine.sha(template),response=response,generationMs=elapsed,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1
                        print(args.phase,variant,sample['id'],strict,row['envelope'],round(elapsed),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
