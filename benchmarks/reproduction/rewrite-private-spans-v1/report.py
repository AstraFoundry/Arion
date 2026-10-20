"""Replay complete cleanup-aligned native evidence without filtering observations."""
import argparse,importlib.util,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
previous=list(sys.path);sys.path.insert(0,str(HERE.parent/'rewrite-local-protection-v1'))
try:
 spec=importlib.util.spec_from_file_location('private_base_report',HERE.parent/'rewrite-local-protection-v1/report.py');base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
finally:sys.path[:]=previous
spec=importlib.util.spec_from_file_location('private_evidence_driver',HERE/'run.py');driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)
base.HERE=HERE;base.runner=driver.runner;ROOT=base.ROOT;RUN=ROOT/'benchmarks/runs/rewrite-private-spans-v1';ARMS=list(driver.runner.ARMS);read=base.read;sha=base.sha

def checked(directory):
 rows,reviews=base.checked(directory)
 for row in rows:
  item=reviews[(row['variant'],row['id'])]
  if any(type(item.get(k)) is not bool for k in ['nativeMeaningFaithful','rawRepairSucceeded']):raise ValueError('Missing raw meaning judgment')
  source=row['source'].encode();cursor=0;parts=[]
  for span in row['encoding']['cleanup'].get('spans',[]):
   if span['start']<cursor or span['end']<=span['start'] or span['end']>len(source):raise ValueError('Invalid cleanup span')
   parts.append(source[cursor:span['start']]);cursor=span['end']
  parts.append(source[cursor:])
  if b''.join(parts).decode().strip()!=row['encoding']['cleanedSource']:raise ValueError('Wrong actual cleaned source')
 return rows,reviews

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();rows=[];reviews={}
 for arm in ARMS:
  actual,judgments=checked(RUN/arm);rows+=actual;reviews.update(judgments)
 refs={r['id']:r for r in read(HERE/'expectations.json')};summaries={arm:base.evidence.summary([r for r in rows if r['variant']==arm],reviews,refs) for arm in ARMS}
 selection=read(RUN/'selection.json')
 if selection['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed selected freeze')
 for arm,hashes in selection['evidence'].items():
  for name,key in [('results.jsonl','resultsSha256'),('review.json','reviewSha256')]:
   if hashes[key]!=sha(RUN/arm/name):raise ValueError('Changed selected evidence')
 candidate=summaries['qwen-private-spans'];required=candidate['rawFaithful']==36 and candidate['repairs']==11 and candidate['rawSceneReady']>=35 and candidate['truncated']==0 and all(reviews[('qwen-private-spans',case)]['proposalSceneUseful'] for case in ['owner-anchor','email-anchor'])
 if bool(selection['qualified'])!=required:raise ValueError('Qualification differs from frozen36-case gate')
 lookup={(r['variant'],r['id']):r for r in rows}
 controls=[dict(id=s['id'],hasPrivateSpans=bool(lookup[('qwen-private-spans',s['id'])]['encoding']['protectedTerms']),requestEqual=lookup[('qwen-clean-system',s['id'])]['request']==lookup[('qwen-private-spans',s['id'])]['request'],bodyEqual=lookup[('qwen-clean-system',s['id'])]['renderedBody']==lookup[('qwen-private-spans',s['id'])]['renderedBody']) for s in read(HERE/'development.json')]
 result=dict(controls=controls,generations=len(rows),reviewedOutputs=len(reviews),summary=summaries,selection=selection,productionActivated=False,scope=read(HERE/'protocol.json'))
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);base.evidence.engine.write(out/'summary.json',result)
 lines=['# Complete cleanup-aligned private-span review','','All72 native outputs and restored bodies personally inspected; exposed Chinese synthetic data, no independent/release acceptance. All outputs retained without rejection or source fallback. Cleanup is precomputed actual production execution; its computation time is excluded from these request measurements.','']
 for row in rows:
  review=reviews[(row['variant'],row['id'])]
  lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}",'','**Original ASR-like source**','','```text',row['source'],'```','','**Actual cleanup/model encoding**','','```json',json.dumps(row['encoding'],ensure_ascii=False,indent=2),'```','','**Complete actual messages**','','```json',json.dumps(row['request']['messages'],ensure_ascii=False,indent=2),'```','','**Raw native output**','','```text',row['response']['choices'][0]['message']['content'],'```','','**Restored body (no rejection)**','','```text',row['renderedBody'],'```','',review['note'],'',f"Faithful: {review['proposalFaithful']}; scene useful: {review['proposalSceneUseful']}; required repair succeeded: {review['repairSucceeded']}",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
