"""Check cold/forward/reverse output parity without changing the frozen main experiment."""
import argparse
import hashlib
import json
import pathlib
import time
import run as study
HERE=study.HERE
engine=study.engine

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();study.check_freeze()
    plan=engine.read(HERE/'parity-plan.json')
    if plan['runnerSha256']!=engine.sha(pathlib.Path(__file__)):raise ValueError('Changed parity runner')
    samples={s['id']:s for s in engine.read(HERE/'regression.json')}
    jobs=[samples[id] for id in plan['caseIds']];template=study.template_path('aligned','work')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);manifest=dict(status='running',planSha256=engine.sha(HERE/'parity-plan.json'),templateSha256=engine.sha(template),provenance=engine.provenance());engine.write(out/'manifest.json',manifest)
    rows=[];worker=None
    try:
        with (out/'results.jsonl').open('x') as stream:
            for arm,ordered in [('cold',jobs),('forward',jobs),('reverse',list(reversed(jobs)))]:
                if arm!='cold':worker=engine.Worker(out,arm,template)
                for index,sample in enumerate(ordered):
                    if arm=='cold':worker=engine.Worker(out,arm+'-'+str(index),template)
                    start=time.monotonic();response=worker.ask('rewrite',session=sample['id'],scene='work',text=sample['source'],context=sample['context']);ms=(time.monotonic()-start)*1000
                    row=dict(arm=arm,position=index,id=sample['id'],source=sample['source'],context=sample['context'],templateSha256=engine.sha(template),response=response,generationMs=ms)
                    stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();rows.append(row);print(arm,sample['id'],repr(response.get('text','')),flush=True)
                    if arm=='cold':worker.close();worker=None
                if worker:worker.close();worker=None
        mismatches=[]
        for id in plan['caseIds']:
            outputs={r['arm']:r['response'].get('text','') for r in rows if r['id']==id}
            if len(set(outputs.values()))>1:mismatches.append(dict(id=id,outputs=outputs))
        manifest.update(status='complete',generations=len(rows),resultsSha256=engine.sha(out/'results.jsonl'),mismatches=mismatches);engine.write(out/'manifest.json',manifest)
    except Exception as error:
        if worker:worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=str(error));engine.write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
