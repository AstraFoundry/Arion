"""Replay lossless-encoding stress outputs, including any false protocol rejection."""
import argparse,importlib.util,json,pathlib,sys
HERE=pathlib.Path(__file__).resolve().parent
previous_path=list(sys.path);sys.path.insert(0,str(HERE.parent/'rewrite-local-protection-v1'))
try:
 spec=importlib.util.spec_from_file_location('protection_stress_base_report',HERE.parent/'rewrite-local-protection-v1/report.py');base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
finally:sys.path[:]=previous_path
base.HERE=HERE;base.RUN=base.ROOT/'benchmarks/runs/rewrite-local-protection-stress-v1';base.runner.HERE=HERE
ROOT=base.ROOT;RUN=base.RUN;read=base.read;sha=base.sha
def checked(directory):
 rows,reviews=base.checked(directory)
 for judgment in reviews.values():
  if any(type(judgment.get(k)) is not bool for k in ['nativeMeaningFaithful','rawRepairSucceeded']):raise ValueError('Incomplete raw meaning assessment')
 return rows,reviews
ARMS=['qwen-direct','qwen-protected']
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();rows=[];reviews={}
 for arm in ARMS:
  actual,judgments=checked(RUN/arm);rows+=actual;reviews.update(judgments)
 refs={r['id']:r for r in read(HERE/'expectations.json')};result=dict(generations=len(rows),reviewedOutputs=len(reviews),summary={v:base.evidence.summary([r for r in rows if r['variant']==v],reviews,refs) for v in ARMS},protocolFailures={v:sum(r['renderError'] is not None for r in rows if r['variant']==v) for v in ARMS},observedFalseProtocolRejections=sum(r['renderError'] is not None and reviews[(r['variant'],r['id'])]['nativeMeaningFaithful'] for r in rows),scope='New synthetic stress supplement, all raw/decoded bodies inspected; not natural ASR, independent ratings, or release acceptance. Superseded/repeated terms must be assessed semantically; a missing marker is not by itself a semantic LLM failure. No hidden fallback or content filtering.')
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);base.evidence.engine.write(out/'summary.json',result);lines=['# Complete local-protection stress review','','All16 raw native generations and decoded bodies personally inspected. Null decoded bodies remain visible alongside their actual raw outputs.','']
 for row in rows:
  review=reviews[(row['variant'],row['id'])];lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}",'','**Source**','','```text',row['source'],'```','','**Full context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Encoding**','','```json',json.dumps(row['encoding'],ensure_ascii=False,indent=2),'```','','**Raw output**','','```text',row['response']['choices'][0]['message']['content'],'```','','**Restored body**','','```text',row['renderedBody'] if row['renderedBody'] is not None else '[null: '+str(row['renderError'])+']','```','',review['note'],'',f"Faithful: {review['proposalFaithful']}; scene useful: {review['proposalSceneUseful']}",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
