"""Expose complete failed tuples and changed control pairs for personal inspection."""
import json,pathlib
R=pathlib.Path.cwd();W=R/'.workspace/rewrite-selector-minicpm-v1'
rows=list(map(json.loads,(W/'comparison-run/results.jsonl').read_text().splitlines()));targets={x['id']:x for x in json.loads((R/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json').read_text())};refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
def conform(r):return all(v in targets[r['id']]['allowedChoices'][k] for k,v in r['selection'].items())
failed={};pairs=[]
for r in rows:
 if not conform(r):
  key=json.dumps([r[k] for k in ['source','context','selection','editedSource']],ensure_ascii=False,sort_keys=True)
  if key not in failed:failed[key]=dict(ids=[],source=r['source'],context=r['context'],hypotheses=r['prepared']['hypotheses'],selection=r['selection'],editedSource=r['editedSource'])
  failed[key]['ids'].append([r['arm'],r['id']])
 q=refs[r['arm'],r['id']]
 if r['selection']!=q['selection']:pairs.append(dict(arm=r['arm'],id=r['id'],source=r['source'],context=r['context'],priorChoice=q['selection'],newChoice=r['selection'],priorEdited=q['editedSource'],newEdited=r['editedSource'],priorConforms=conform(q),newConforms=conform(r)))
(W/'unique-failed-tuples.json').write_text(json.dumps(list(failed.values()),ensure_ascii=False,indent=2)+'\n');(W/'changed-pairs.json').write_text(json.dumps(pairs,ensure_ascii=False,indent=2)+'\n')
print('failed tuples',len(failed),'changes',len(pairs),'gains',sum(not x['priorConforms'] and x['newConforms'] for x in pairs),'regress',sum(x['priorConforms'] and not x['newConforms'] for x in pairs))
for i,x in enumerate(failed.values()): print(i,json.dumps(x,ensure_ascii=False))
