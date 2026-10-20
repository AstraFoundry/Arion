"""Replay all36-case prompt/protocol ablations with no hidden semantic rejection."""
import argparse,importlib.util,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
previous=list(sys.path);sys.path.insert(0,str(HERE.parent/'rewrite-local-protection-v1'))
try:
 spec=importlib.util.spec_from_file_location('current_task_base_report',HERE.parent/'rewrite-local-protection-v1/report.py');base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
finally:sys.path[:]=previous
spec=importlib.util.spec_from_file_location('current_task_driver',HERE/'run.py');driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
base.HERE=HERE;base.runner=driver.runner
ROOT=base.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-current-task-v1';read=base.read;sha=base.sha
ARMS=list(driver.runner.ARMS)
def checked(directory):
 rows,reviews=base.checked(directory)
 for review in reviews.values():
  if any(type(review.get(k)) is not bool for k in ['nativeMeaningFaithful','rawRepairSucceeded']):raise ValueError('Missing raw meaning review')
 return rows,reviews

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();rows=[];reviews={}
 for arm in ARMS:
  actual,judgments=checked(RUN/arm);rows+=actual;reviews.update(judgments)
 refs={r['id']:r for r in read(HERE/'expectations.json')};selection=read(RUN/'selection.json')
 if selection['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed selected freeze')
 for arm,hashes in selection['evidence'].items():
  for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
   if hashes[key]!=sha(RUN/arm/name):raise ValueError('Changed selection evidence')
 result=dict(generations=len(rows),reviewedOutputs=len(reviews),summary={v:base.evidence.summary([r for r in rows if r['variant']==v],reviews,refs) for v in ARMS},protocolFailures={v:sum(r['renderError'] is not None for r in rows if r['variant']==v) for v in ARMS},nativeMeaningFaithful={v:sum(reviews[(r['variant'],r['id'])]['nativeMeaningFaithful'] for r in rows if r['variant']==v) for v in ARMS},observedFalseProtocolRejections={v:sum(r['renderError'] is not None and reviews[(r['variant'],r['id'])]['nativeMeaningFaithful'] for r in rows if r['variant']==v) for v in ARMS},selection=selection,evaluationExecuted=False,productionActivated=False,scope='All36 exposed Chinese synthetic regressions, not newly held-out or independent ratings. Recency adds only the current-input task; flexible also changes token instruction and removes output rejection, so its effects are a package comparison. Complete original/encoded source, full OCR, raw native output and restored/null body separate. Same latest2B/CPU4/cacheoff; no4B/olderQwen/Granite inference. CurrentM4/32GB is not16GB/native ASR acceptance. Native/external clock inconsistencies and large host timing drift preclude latency acceptance or causal speed claims. Original-language preservation beyond Chinese remains untested.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);base.evidence.engine.write(out/'summary.json',result);lines=['# Complete current-task prompt/protocol review','','Every raw native output, mapping and restored/null body personally inspected. Native meaning and delivered-body fidelity are separately reviewed; no hidden source fallback. Flexible output restores known tokens without rejecting content.','']
 for row in rows:
  review=reviews[(row['variant'],row['id'])];lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}; template: {read(HERE/'templates'/(row['scene']+'.json'))['version']}",'','**Original source**','','```text',row['source'],'```','','**Full OCR/context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Encoding and policy**','','```json',json.dumps(row['encoding'],ensure_ascii=False,indent=2),'```','','**Raw native body**','','```text',row['response']['choices'][0]['message']['content'],'```','','**Restored body**','','```text',row['renderedBody'] if row['renderedBody'] is not None else '[null: '+str(row['renderError'])+']','```','',review['note'],'',f"Native meaning faithful: {review['nativeMeaningFaithful']}; delivered faithful: {review['proposalFaithful']}; scene usable: {review['proposalSceneUseful']}; request {row['generationMs']:.1f}ms",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
