"""Frozen development comparison for user-selected structural editing preferences."""
import argparse
import hashlib
import importlib.util
import json
import pathlib
import time
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
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
    engine.write(out/'manifest.json',manifest)
    worker=None;count=0
    try:
        with (out/'results.jsonl').open('x') as stream:
            for variant in ['examples-v2','structure-v3']:
                for scene in ['work','email']:
                    template=HERE/f'{scene}-zh.json' if variant=='structure-v3' else HERE.parent/f'scene-rewrite-prompt-v2/templates/examples-{scene}-zh.json'
                    worker=engine.Worker(out,f'{variant}-{scene}',template)
                    for sample in engine.read(HERE/'development.json'):
                        if sample['scene']!=scene:continue
                        start=time.monotonic()
                        response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'],context={})
                        strict=engine.strict_check(sample['source'],response,{})
                        row=dict(**sample,variant=variant,templateSha256=engine.sha(template),response=response,generationMs=(time.monotonic()-start)*1000,strict=strict,envelope=engine.envelope(response,strict))
                        stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1
                        print(variant,sample['id'],response.get('text'),flush=True)
                    worker.close();worker=None
        manifest.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
