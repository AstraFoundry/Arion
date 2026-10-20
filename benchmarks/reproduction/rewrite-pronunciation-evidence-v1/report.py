"""Replay all pronunciation metadata observations with exact serializer reconstruction."""
import argparse,importlib.util,json,pathlib,sys,math
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_evidence as evidence
spec=importlib.util.spec_from_file_location('pronunciation_presentation',HERE/'pronunciation_messages.py');presentation=importlib.util.module_from_spec(spec);spec.loader.exec_module(presentation)
ROOT=evidence.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-pronunciation-evidence-v1';read=evidence.read;sha=evidence.sha
def template(row):return HERE/f"templates/{row['variant']}-{row['scene']}.json"
def checked(directory):
 original=evidence.engine.messages
 try:
  evidence.engine.messages=lambda sample,package:presentation.messages(sample,package)[0]
  rows,reviews=evidence.checked(directory,HERE,{v:read(HERE/'development.json') for v in ['baseline','evidence']},template,lambda row:False,'dev')
 finally:evidence.engine.messages=original
 for row in rows:
  expected=presentation.messages(row,read(template(row)))[1]
  if row['pronunciationEvidence']!=expected:raise ValueError('Changed derived pronunciation evidence')
  if min(row['preparationMs'],row['generationMs'])<0 or abs(row['textStageMs']-row['preparationMs']-row['generationMs'])>.001:raise ValueError('Wrong text stage timing accounting')
 return rows,reviews
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();directory=RUN/'dev';rows,reviews=checked(directory);selection=read(directory/'selection.json');refs={r['id']:r for r in read(HERE/'expectations.json')}
 for filename,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
  if selection[key]!=sha(directory/filename):raise ValueError('Changed selection evidence')
 if selection['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed selection freeze')
 lookup={(r['variant'],r['id']):r for r in rows};controls=[]
 for sample in read(HERE/'development.json'):
  if sample['scene'] not in ['email','chat']:continue
  a=lookup[('baseline',sample['id'])];b=lookup[('evidence',sample['id'])];controls.append(dict(id=sample['id'],requestEqual=a['request']==b['request'],bodyEqual=a['response']['choices'][0]['message']['content']==b['response']['choices'][0]['message']['content'],baselineMs=a['generationMs'],candidateMs=b['generationMs']))
 result=dict(generations=48,reviewedOutputs=48,summary={v:evidence.summary([r for r in rows if r['variant']==v],reviews,refs) for v in ['baseline','evidence']},unchangedSceneControls=controls,selection=selection,evaluationExecuted=False,productionActivated=False,scope='Exposed synthetic development only; raw free generation without content validator. Text-derived CMU hypotheses are not acoustic observations or verified truth. Three ambiguity controls remain unsupported guesses. Unknown candidates are not exhaustively covered. Fresh24 ungenerated after failed gate. Current M4/32GB CPU4 observations do not establish Apple Silicon16GB acceptance. Native post-response sampled RSS excludes Python dictionary memory; textStageMs excludes ASR and target insertion.')
 result['preparation']={v:dict(p50Ms=sorted(r['preparationMs'] for r in rows if r['variant']==v)[11],maxMs=max(r['preparationMs'] for r in rows if r['variant']==v)) for v in ['baseline','evidence']}
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);evidence.engine.write(out/'summary.json',result)
 text=evidence.markdown(rows,reviews,template,'Complete pronunciation evidence review')
 text+='\n# Derived input evidence (all rows)\n'
 for row in rows:text+='\n## '+row['variant']+' / '+row['id']+'\n\n```json\n'+json.dumps(row['pronunciationEvidence'],ensure_ascii=False,indent=2)+'\n```\n'
 (out/'review.md').write_text(text);print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
