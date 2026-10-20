"""Replay the complete failed2B phonetic-contrast template diagnostic."""
import argparse,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_evidence as evidence
ROOT=evidence.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-2b-contrast-v1';read=evidence.read;sha=evidence.sha
def template(row):return HERE/f"templates/{row['variant']}-{row['scene']}.json"
def checked(directory):return evidence.checked(directory,HERE,{v:read(HERE/'development.json') for v in ['baseline','compact']},template,lambda row:False,'dev')
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();directory=RUN/'dev';rows,reviews=checked(directory);selection=read(directory/'selection.json');refs={r['id']:r for r in read(HERE/'expectations.json')}
 if selection['freezeSha256']!=sha(HERE/'freeze.json') or selection['resultsSha256']!=sha(directory/'results.jsonl') or selection['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed selection evidence')
 lookup={(r['variant'],r['id']):r for r in rows};controls=[]
 for sample in read(HERE/'development.json'):
  if sample['scene']=='work':continue
  a=lookup[('baseline',sample['id'])];b=lookup[('compact',sample['id'])];controls.append(dict(id=sample['id'],requestEqual=a['request']==b['request'],bodyEqual=a['response']['choices'][0]['message']['content']==b['response']['choices'][0]['message']['content'],baselineMs=a['generationMs'],candidateMs=b['generationMs']))
 result=dict(generations=32,reviewedOutputs=32,summary={v:evidence.summary([r for r in rows if r['variant']==v],reviews,refs) for v in ['baseline','compact']},unchangedSceneControls=controls,selection=selection,evaluationExecuted=False,productionActivated=False,scope='Latest2B four-scene exposed synthetic diagnostic. Only work template changes; compact is inherited machine ID for contrast package, not a compactness claim. Fresh24 remain ungenerated after failed gate. Identical non-work requests show timing drift; serial arm order prevents attributing their timing changes to work-prompt edits. Current M4/32GB observations do not prove user minimum Apple Silicon16GB acceptance.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);evidence.engine.write(out/'summary.json',result);(out/'review.md').write_text(evidence.markdown(rows,reviews,template,'Complete2B phonetic-contrast template review'));print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
