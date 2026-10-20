"""Apply a source-bound retrospective policy view without changing historical scores."""
import hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
RUN=ROOT/'benchmarks/runs/rewrite-spelling-alignment-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(row):return hashlib.sha256(json.dumps(row,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def checked():
 manifest=read(RUN/'manifest.json')
 if manifest['status']!='complete' or manifest['newInference']:raise ValueError('Wrong retrospective status')
 for name,digest in manifest['files'].items():
  if sha(ROOT/name)!=digest:raise ValueError('Changed normalization evidence: '+name)
 review=read(RUN/'review.json');items=review['reviews'];keys=set()
 for j in items:
  key=(j['run'],j['variant'],j['id'])
  if key in keys:raise ValueError('Duplicate retrospective review')
  keys.add(key)
  raw=[json.loads(x) for x in (ROOT/'benchmarks/runs'/j['run']/'results.jsonl').read_text().splitlines()]
  row=next(x for x in raw if x['variant']==j['variant'] and x['id']==j['id'])
  output=row.get('renderedBody',row['response']['choices'][0]['message']['content'])
  if rowhash(row)!=j['rowSha256'] or any(row[k]!=j[k] for k in ['source','context']) or output!=j['output']:raise ValueError('Changed reviewed normalization output')
 if len(items)!=manifest['reviewedOutputs'] or sum(j['alignedFaithful'] for j in items)!=manifest['reclassifiedFaithful']:raise ValueError('Incomplete normalization review')
 return manifest,review,read(RUN/'policy.json')
def apply(cases):
 manifest,review,policy=checked();lookup={(c['run'],c['variant'],c['id']):c for c in cases}
 affected={(c['run'],c['variant'],c['id']) for c in cases if c['faithful'] is False and any(x in c['output'] for x in ['readCash','writeRight']) and any(x in c['source'] for x in ['read cash','write right'])}
 if affected!={(j['run'],j['variant'],j['id']) for j in review['reviews']}:raise ValueError('Missing affected output review')
 for j in review['reviews']:
  c=lookup[(j['run'],j['variant'],j['id'])]
  if c['faithful']!=j['originalFaithful'] or c['useful']!=j['originalSceneUseful'] or any(c[k]!=j[k] for k in ['source','context','output']):raise ValueError('Changed original judgment view')
  c['alignedReview']=dict(faithful=j['alignedFaithful'],useful=j['alignedSceneUseful'],note=j['note'],policy=review['policy'])
 return dict(policy=policy,reviewed=manifest['reviewedOutputs'],reclassified=manifest['reclassifiedFaithful'],stillUnfaithful=manifest['stillUnfaithful'],newInference=False)
if __name__=='__main__':
 manifest,review,policy=checked();print(json.dumps(dict(reviewed=manifest['reviewedOutputs'],reclassified=manifest['reclassifiedFaithful'],newInference=False)))
