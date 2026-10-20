"""Inspect the hard system-placement probe and matched prior-subset evidence."""
import argparse,importlib.util,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
previous=list(sys.path);sys.path.insert(0,str(HERE.parent/'rewrite-local-protection-v1'))
try:
 spec=importlib.util.spec_from_file_location('system_probe_base_report',HERE.parent/'rewrite-local-protection-v1/report.py');base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
finally:sys.path[:]=previous
spec=importlib.util.spec_from_file_location('system_probe_driver',HERE/'run.py');driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
base.HERE=HERE;base.runner=driver.runner
ROOT=base.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-system-task-probe-v1';read=base.read;sha=base.sha

def checked(directory):
 rows,reviews=base.checked(directory)
 for review in reviews.values():
  if any(type(review.get(k)) is not bool for k in ['nativeMeaningFaithful','rawRepairSucceeded']):raise ValueError('Missing raw review')
 return rows,reviews

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();rows,reviews=checked(RUN/'dev');refs={r['id']:r for r in read(HERE/'expectations.json')};selection=read(RUN/'selection.json')
 for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
  if selection[key]!=sha(RUN/'dev'/name):raise ValueError('Changed probe decision evidence')
 if selection['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed probe freeze')
 prior_path=HERE.parent/'rewrite-current-task-v1/report.py';spec=importlib.util.spec_from_file_location('system_probe_prior_report',prior_path);prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior);ids={r['id'] for r in rows};comparisons={}
 for arm in prior.ARMS:
  actual,judgments=prior.checked(prior.RUN/arm);subset=[r for r in actual if r['id'] in ids];comparisons[arm]=base.evidence.summary(subset,judgments,refs)
 result=dict(generations=len(rows),reviewedOutputs=len(reviews),summary=base.evidence.summary(rows,reviews,refs),matchedPriorSubset=comparisons,selection=selection,productionActivated=False,scope='Hard12-case exposed feasibility probe, not full36 regression or newly held-out data. Same instruction meanings move from user tail to system scope with an explicit data locator. No output rejection; all native/restored bodies retained. Four observed boundary regressions are repaired, but four stress cases still fail. No expanded36 or fresh24 inference after failed probe. Raw-input correction case is separately clarified against production CleanSpeech. No native/16GB/independent review or timing acceptance.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);base.evidence.engine.write(out/'summary.json',result);lines=['# Complete system-task feasibility probe review','','All12 native outputs and restored bodies personally inspected. This hard subset is not a replacement for the mandatory full36 regressions. No output rejection or source fallback.','']
 for row in rows:
  review=reviews[(row['variant'],row['id'])];lines += [f"## {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}",'','**Original source**','','```text',row['source'],'```','','**Full reference**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw native output**','','```text',row['response']['choices'][0]['message']['content'],'```','','**Restored body**','','```text',str(row['renderedBody']),'```','',review['note'],'',f"Faithful: {review['proposalFaithful']}; scene useful: {review['proposalSceneUseful']}",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
