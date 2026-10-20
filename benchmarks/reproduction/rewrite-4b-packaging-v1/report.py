"""Validate and replay the complete larger-model development diagnostic."""
import argparse,sys,pathlib,json
import run as study
ROOT=study.ROOT;HERE=study.HERE;RUN=ROOT/'benchmarks/runs/rewrite-4b-packaging-v1'
sys.path.insert(0,str(HERE.parent))
import review_evidence as evidence
engine=study.engine
VARIANTS=['labeled','json','xml']

def checked(directory):
    frozen=engine.read(HERE/'freeze.json')
    for name,digest in frozen['files'].items():
        if engine.sha(ROOT/name)!=digest:raise ValueError('Changed frozen input: '+name)
    original=engine.read(ROOT/'benchmarks/reproduction/rewrite-model-screen-v1/freeze.json');manifest=engine.read(directory/'manifest.json')
    if manifest['phase']!='dev' or manifest['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('Wrong frozen phase')
    if manifest['provenance']['modelFiles']!=frozen['modelFiles'] or manifest['provenance']['packagingBuild']!=frozen['build']:raise ValueError('Wrong model/build')
    worker=str(ROOT/'.workspace/rewrite-packaging-v1/build/worker');evidence.check_runtime(manifest['provenance'],original['runtimeFiles'],worker)
    return evidence.checked_rows(directory,engine.read(HERE/'cases.json'),VARIANTS,lambda row:HERE/f"templates/{row['scene']}.json",lambda row:frozen['modelFiles'])

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();directory=RUN/'dev';rows,reviews=checked(directory);refs={r['id']:r for r in engine.read(HERE/'expectations.json')};decision=engine.read(directory/'decision.json')
    if decision['freezeSha256']!=engine.sha(HERE/'freeze.json') or decision['devResultsSha256']!=engine.sha(directory/'results.jsonl') or decision['devReviewSha256']!=engine.sha(directory/'review.json'):raise ValueError('Changed reviewed decision')
    result=dict(generations=len(rows),reviewedOutputs=len(reviews),summary=evidence.summarize(rows,reviews,refs),decision=decision,evaluationExecuted=False,productionActivated=False,scope='Exposed work/email diagnostic; same pinned model/template/budget within the stated factor comparison. No natural ASR, independent ratings or native acceptance. Fresh16 remain ungenerated after failed development gate. Request timing excludes native rendering/insertion and ASR contention.')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);engine.write(out/'summary.json',result);(out/'review.md').write_text(evidence.markdown(rows,reviews,'Complete fixed-4B serializer review'));print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
