"""Validate complete native decision requests against immutable source messages."""
import hashlib,json
from pathlib import Path
HERE=Path(__file__).resolve().parent
ROOT=HERE.parents[2]
def rowhash(value):return hashlib.sha256(json.dumps(value,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def validate(records):
 references={(r['arm'],r['id']):r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
 seen=set()
 for item in records:
  ref=item['sourceReference'];row=references[ref['arm'],item['id']]
  if item['id'] in seen or ref!=dict(path='benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl',arm='baseline',rowSha256=rowhash(row)):raise ValueError('Changed source identity')
  request=item['request']
  if set(request)!= {'state','questions'} or request['state']!=dict(originalMessages=row['request']['messages']):raise ValueError('Lost original rules, examples, source or complete context')
  if item['scene']!=row['scene'] or len(request['state']['originalMessages'])!=22:raise ValueError('Changed source scene or examples')
  hypotheses=row['prepared']['hypotheses']
  if set(request['questions'])!={h['id'] for h in hypotheses} or any(len(h['candidateSpellings'])!=1 or h['status']!='unique' for h in hypotheses):raise ValueError('Changed native question scope')
  for question in request['questions'].values():
   if question['type']!='choice' or set(question['criteria'])!={'keep','replace'} or not question['instructions'] or not all(isinstance(s,str) and s for s in question['criteria'].values()):raise ValueError('Changed native choice contract')
  seen.add(item['id'])
 if len(records)!=6:raise ValueError('Incomplete native probe scope')
 return records
