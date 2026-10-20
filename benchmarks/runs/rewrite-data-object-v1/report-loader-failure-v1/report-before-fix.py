"""Replay new input-format observations and a separately identified historical baseline."""
import argparse,importlib.util,pathlib,json
HERE=pathlib.Path(__file__).resolve().parent

def load(path,name):
 spec=importlib.util.spec_from_file_location(name,path);module=importlib.util.module_from_spec(spec);spec.loader.exec_module(module);return module
prior=load(HERE.parent/'rewrite-private-spans-v1/report.py','object_prior_evidence')
driver=load(HERE/'run.py','object_evidence_driver')
prior.HERE=HERE;prior.base.HERE=HERE;prior.base.runner=driver.runner
ROOT=prior.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-data-object-v1';ARMS=list(driver.runner.ARMS);read=prior.read;sha=prior.sha

def checked(directory):return prior.checked(directory)
def baseline():
 protocol=read(HERE/'protocol.json');reference=protocol['baseline'];directory=ROOT/reference['run']
 for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
  if sha(directory/name)!=reference[key]:raise ValueError('Changed reused baseline')
 module=load(HERE.parent/'rewrite-private-spans-v1/report.py','object_baseline_evidence');return module.checked(directory)

HEADER='# Complete input-object packaging review'
REVIEW_SCOPE='All72 new native and restored bodies personally inspected. Historical36 baseline bodies are reused and counted only in their original round. No output filtering or source fallback.'

def collect():
 rows=[];reviews={}
 for arm in ARMS:
  actual,judgments=checked(RUN/arm);rows+=actual;reviews.update(judgments)
 old,old_reviews=baseline();refs={r['id']:r for r in read(HERE/'expectations.json')};lookup={r['id']:r for r in old};summaries={arm:prior.base.evidence.summary([r for r in rows if r['variant']==arm],reviews,refs) for arm in ARMS};decision=read(RUN/'selection.json')
 if decision['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed decision freeze')
 for arm,hashes in decision['evidence'].items():
  for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
   if sha(RUN/arm/name)!=hashes[key]:raise ValueError('Changed decision evidence')
 eligible=[]
 for arm,s in summaries.items():
  if s['rawFaithful']==36 and s['repairs']==11 and s['rawSceneReady']>=35 and s['truncated']==0 and all(reviews[(arm,case)]['proposalSceneUseful'] for case in ['owner-anchor','email-anchor']):eligible.append(arm)
 if decision['qualified']!=bool(eligible) or (eligible and decision['selected'] not in eligible) or (not eligible and decision['selected'] is not None):raise ValueError('Decision does not match frozen full36 gate')
 controls=[dict(arm=r['variant'],id=r['id'],requestEqual=r['request']==lookup[r['id']]['request'],bodyEqual=r['renderedBody']==lookup[r['id']]['renderedBody'],hasContext=bool(r['context'])) for r in rows]
 result=dict(newGenerations=len(rows),reviewedNewOutputs=len(reviews),reusedBaselineGenerations=len(old),reusedBaselineSummary=prior.base.evidence.summary(old,old_reviews,refs),summary=summaries,controls=controls,selection=decision,productionActivated=False,scope=read(HERE/'protocol.json'))
 return result,rows,reviews

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();result,rows,reviews=collect()
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);prior.base.evidence.engine.write(out/'summary.json',result)
 lines=[HEADER,'',REVIEW_SCOPE,'']
 for row in rows:
  review=reviews[(row['variant'],row['id'])];lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}",'','**Original source**','','```text',row['source'],'```','','**Complete context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Actual model messages**','','```json',json.dumps(row['request']['messages'],ensure_ascii=False,indent=2),'```','','**Private restoration/cleanup trace**','','```json',json.dumps(row['encoding'],ensure_ascii=False,indent=2),'```','','**Raw native output**','','```text',row['response']['choices'][0]['message']['content'],'```','','**Restored body (unfiltered)**','','```text',row['renderedBody'],'```','',review['note'],'',f"Faithful under frozen rubric: {review['proposalFaithful']}; scene useful: {review['proposalSceneUseful']}; required repair succeeded: {review['repairSucceeded']}",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
