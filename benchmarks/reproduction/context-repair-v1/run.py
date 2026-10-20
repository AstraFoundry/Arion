"""Run a frozen local context-repair diagnostic; evaluation references stay out of inference."""
import argparse
import importlib.util
import json
import pathlib
import time
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('prompt_v2',HERE.parent/'scene-rewrite-prompt-v2/run.py')
v2=importlib.util.module_from_spec(spec);spec.loader.exec_module(v2)
engine=v2.engine

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args()
    for name,digest in engine.read(HERE/'freeze.json')['files'].items():
        if engine.sha(HERE/name)!=digest:raise ValueError('Changed frozen input: '+name)
    v2.check_freeze();engine.check_freeze()
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    manifest=dict(status='running',freezeSha256=engine.sha(HERE/'freeze.json'),provenance=engine.provenance())
    engine.write(out/'manifest.json',manifest);worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for variant in ['production','structure-v3','context-repair-v1']:
                groups=['all'] if variant=='production' else ['work','email']
                for scene in groups:
                    template=None if variant=='production' else (HERE/f'{scene}-zh.json' if variant=='context-repair-v1' else HERE.parent/f'scene-structure-v3/{scene}-zh.json')
                    worker=engine.Worker(out,f'{variant}-{scene}',template)
                    for sample in engine.read(HERE/'cases.json'):
                        if scene!='all' and sample['scene']!=scene:continue
                        modelscene=('email' if sample['scene']=='email' else 'chat') if variant=='production' else sample['scene']
                        start=time.monotonic()
                        response=worker.ask('rewrite',session=sample['id'],scene=modelscene,text=sample['source'],context=sample['context'])
                        ms=(time.monotonic()-start)*1000
                        strict=engine.strict_check(sample['source'],response,sample['context'])
                        row=dict(**sample,variant=variant,effectiveModelScene=modelscene,templateSha256=engine.sha(template) if template else None,response=response,generationMs=ms,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1
                        print(variant,sample['id'],strict,round(ms),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
