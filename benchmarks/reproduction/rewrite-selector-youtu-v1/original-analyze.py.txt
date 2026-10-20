"""Prepare full failed tuples and changed actual-control pairs for personal review."""
from pathlib import Path
import json,importlib.util,collections
R=Path.cwd();D=Path(__file__).resolve().parent;H=R/'benchmarks/reproduction/rewrite-selector-youtu-v1';s=importlib.util.spec_from_file_location('youtu_analysis',H/'report.py');m=importlib.util.module_from_spec(s);s.loader.exec_module(m)
rows=list(map(json.loads,(D/'run/results.jsonl').read_text().splitlines()));assert len(rows)==202
targets={r['id']:r for r in json.loads((R/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json').read_text())};refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};groups=collections.defaultdict(collections.Counter);failed={};changed=[]
for r in rows:
 a=m.assessment(r,targets);groups[r['arm']].update(conforming=a['conforms'],harmful=a['harmful'],missed=a['missed'],repaired=a['repaired'],required=a['required']);ref=refs[r['arm'],r['id']];old=m.assessment(ref,targets)
 d=dict(id=r['id'],arm=r['arm'],source=r['source'],context=r['context'],hypotheses=r['prepared']['hypotheses'],raw=r['response']['choices'][0]['message']['content'],edited=r['editedSource'],assessment=a)
 if not a['conforms']:failed.setdefault(json.dumps([r['source'],r['context'],r['prepared']['hypotheses'],r['editedSource']],sort_keys=True,ensure_ascii=False),d)
 if r['selection']!=ref['selection']:changed.append(dict(**d,priorRaw=ref['response']['choices'][0]['message']['content'],priorEdited=ref['editedSource'],priorAssessment=old))
for n,x in [('unique-failed-tuples.json',list(failed.values())),('changed-pairs.json',changed),('aggregate.json',dict(groups=dict(groups),uniqueFailedTuples=len(failed),changedPairs=len(changed),improvements=sum(not x['priorAssessment']['conforms'] and x['assessment']['conforms'] for x in changed),regressions=sum(x['priorAssessment']['conforms'] and not x['assessment']['conforms'] for x in changed)))]: (D/n).write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
print((D/'aggregate.json').read_text())
