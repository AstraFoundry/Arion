"""Replay source plans, reconstructed bodies, protocol failures and manual review."""
import argparse,hashlib,json,pathlib,sys,math
HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parent))
import gguf_evidence as evidence
import importlib.util
spec=importlib.util.spec_from_file_location('source_plan_runner',HERE/'run.py');runner=importlib.util.module_from_spec(spec);spec.loader.exec_module(runner)
ROOT=evidence.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-source-plan-v1';read=evidence.read;sha=evidence.sha

def checked(directory):
 frozen=read(HERE/'freeze.json');manifest=read(directory/'manifest.json');review=read(directory/'review.json');rows=[json.loads(l) for l in (directory/'results.jsonl').read_text().splitlines()]
 for path,digest in frozen['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen input '+path)
 if manifest['status']!='complete' or manifest['phase']!='dev' or manifest['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Incomplete/wrong frozen inference')
 if manifest['model']!=frozen['model'] or manifest['runtimeFiles']!=frozen['runtimeFiles'] or manifest['generations']!=56:raise ValueError('Changed model/runtime/coverage')
 if any(item['resultsSha256']!=sha(directory/'results.jsonl') for item in [manifest,review]):raise ValueError('Changed reviewed results')
 samples={s['id']:s for s in read(HERE/'development.json')};expected={(v,s) for v in ['baseline','plan'] for s in samples};actual=[(r['variant'],r['id']) for r in rows];judgments={(r['variant'],r['id']):r for r in review['reviews']}
 if set(actual)!=expected or len(actual)!=len(expected) or set(judgments)!=expected or len(review['reviews'])!=len(expected):raise ValueError('Missing/duplicate observations or review')
 workers={(w['variant'],w['scene']):w for w in manifest['workers']}
 if set(workers)!={(v,s) for v in ['baseline','plan'] for s in ['work','email','prompt','chat']} or len(manifest['workers'])!=8:raise ValueError('Wrong worker coverage')
 for worker in workers.values():
  command=worker['command']
  if command[-len(evidence.CPU_ARGUMENTS):]!=evidence.CPU_ARGUMENTS or command[1]!='--model' or pathlib.Path(command[2]).name!=frozen['model']['filename'] or pathlib.Path(command[0]).name!='llama-server' or command[3:7]!=['--host','127.0.0.1','--port','18769']:raise ValueError('Wrong native command')
 for row in rows:
  sample=samples[row['id']]
  if any(row[k]!=v for k,v in sample.items()) or row['effectiveModelScene']!=sample['scene']:raise ValueError('Changed source/context/scene')
  request,prepared=runner.payload(sample,row['variant'])
  if request!=row['request'] or prepared!=row['preparedPlan']:raise ValueError('Changed actual schema/request/derived units')
  path=HERE/('plan.py' if row['variant']=='plan' else f"templates/baseline-{sample['scene']}.json")
  if row['packageSha256']!=sha(path):raise ValueError('Wrong package')
  raw=row['response']['choices'][0]['message']['content'];error=None
  try:body=runner.plan.render(raw,prepared) if row['variant']=='plan' else raw
  except (ValueError,KeyError,TypeError) as failure:body=None;error=type(failure).__name__+': '+str(failure)
  if body!=row['renderedBody'] or error!=row['renderError']:raise ValueError('Changed rendered body/error')
  if abs(row['textStageMs']-sum(row[k] for k in ['preparationMs','generationMs','renderMs']))>.001 or min(row[k] for k in ['preparationMs','generationMs','renderMs'])<0:raise ValueError('Wrong timing accounting')
  judgment=judgments[(row['variant'],row['id'])]
  if judgment['rowSha256']!=evidence.rowhash(row) or any(type(judgment[k]) is not bool for k in ['proposalFaithful','proposalSceneUseful','repairSucceeded']) or not judgment['note']:raise ValueError('Changed/incomplete review')
 return rows,judgments

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();rows,reviews=checked(RUN/'dev');refs={r['id']:r for r in read(HERE/'expectations.json')};selection=read(RUN/'dev/selection.json')
 for filename,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
  if selection[key]!=sha(RUN/'dev'/filename):raise ValueError('Changed selected evidence')
 if selection['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed selected freeze')
 result=dict(generations=len(rows),reviewedOutputs=len(reviews),summary={v:evidence.summary([r for r in rows if r['variant']==v],reviews,refs) for v in ['baseline','plan']},protocolFailures={v:sum(r['renderError'] is not None for r in rows if r['variant']==v) for v in ['baseline','plan']},selection=selection,evaluationExecuted=False,productionActivated=False,scope='Package comparison, not grammar alone: source units, different instructions, no demonstrations, same384 output-token cap. Source-index presence is not semantic completeness. All raw plans/bodies/protocol failures remain visible. Exposed synthetic development only; no independent review/natural ASR/native insertion or Apple Silicon16GB acceptance.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);evidence.engine.write(out/'summary.json',result);lines=['# Complete source-plan package review','','All56 raw generations personally inspected. A JSON plan is separate from its constructed body. Protocol failures have null body and remain in all denominators. No semantic output validator or fallback.','']
 for row in rows:
  review=reviews[(row['variant'],row['id'])];lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}; package: {row['packageSha256']}",'','**Source**','','```text',row['source'],'```','','**Complete context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw native generation**','','```text',row['response']['choices'][0]['message']['content'],'```','','**Constructed body**','','```text',row['renderedBody'] if row['renderedBody'] is not None else '[null: protocol failure]','```','',f"Protocol error: {row['renderError']}",'',review['note'],'',f"Faithful: {review['proposalFaithful']}; scene useful: {review['proposalSceneUseful']}; request {row['generationMs']:.1f}ms",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
