"""Replay the complete four-scene prompt-package development diagnostic."""
import argparse,pathlib,sys,json
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_evidence as evidence
ROOT=evidence.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-small-prompt-v1';read=evidence.read;sha=evidence.sha
def template(row):return HERE/f"templates/{row['variant']}-{row['scene']}.json"
def checked(directory):
 return evidence.checked(directory,HERE,{v:read(HERE/'development.json') for v in ['baseline','compact']},template,lambda row:False,'dev')
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();directory=RUN/'dev';rows,reviews=checked(directory);decision=read(directory/'selection.json');refs={r['id']:r for r in read(HERE/'expectations.json')}
 if decision['freezeSha256']!=sha(HERE/'freeze.json') or decision['resultsSha256']!=sha(directory/'results.jsonl') or decision['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed selection evidence')
 result=dict(generations=len(rows),reviewedOutputs=len(reviews),summary={v:evidence.summary([r for r in rows if r['variant']==v],reviews,refs) for v in ['baseline','compact']},selection=decision,evaluationExecuted=False,productionActivated=False,scope='Four-scene exposed synthetic development. New24 frozen before inference remain ungenerated after failed gate. Same model/runtime/cache policy; template policy and work demonstrations change as a package. No natural ASR or independent release review.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);evidence.engine.write(out/'summary.json',result);(out/'review.md').write_text(evidence.markdown(rows,reviews,template,'Complete latest0.8B prompt-package review'));print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
