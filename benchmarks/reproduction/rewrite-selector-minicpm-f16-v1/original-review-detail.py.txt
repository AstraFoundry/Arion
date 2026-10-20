"""Expose all failed source tuples and changed Q4/F16 pairs for personal inspection."""
from pathlib import Path
import json
R=Path.cwd();W=R/'.workspace/rewrite-selector-minicpm-f16-v1';rows=list(map(json.loads,(W/'run/results.jsonl').read_text().splitlines()));refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-minicpm-v1/comparison-run/results.jsonl').read_text().splitlines())};targets={r['id']:r for r in json.loads((R/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json').read_text())}
def conform(r):return all(v in targets[r['id']]['allowedChoices'][k] for k,v in r['selection'].items())
failed={};changed=[]
for r in rows:
 if not conform(r):
  key=json.dumps([r[k] for k in ['source','context','selection','editedSource']],ensure_ascii=False,sort_keys=True)
  if key not in failed:failed[key]=dict(ids=[],source=r['source'],context=r['context'],hypotheses=r['prepared']['hypotheses'],selection=r['selection'],editedSource=r['editedSource'])
  failed[key]['ids'].append([r['arm'],r['id']])
 q=refs[r['arm'],r['id']]
 if r['selection']!=q['selection']:changed.append(dict(arm=r['arm'],id=r['id'],source=r['source'],context=r['context'],q4Choice=q['selection'],f16Choice=r['selection'],q4Edited=q['editedSource'],f16Edited=r['editedSource'],q4Conforms=conform(q),f16Conforms=conform(r)))
for name,value in [('unique-failed-tuples.json',list(failed.values())),('changed-pairs.json',changed)]: (W/name).write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
print('Failed tuples',len(failed),'changedpairs',len(changed),'improved',sum(not p['q4Conforms'] and p['f16Conforms'] for p in changed),'regressed',sum(p['q4Conforms'] and not p['f16Conforms'] for p in changed))
for i,r in enumerate(failed.values()):print(i,json.dumps(r,ensure_ascii=False))
