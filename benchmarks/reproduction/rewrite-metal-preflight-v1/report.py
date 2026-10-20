"""Verify one identical-request provider probe and separately observed placement."""
import pathlib,json,hashlib
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-metal-preflight-v1/run'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def checked():
 m=read(RUN/'manifest.json');rows=[json.loads(x) for x in (RUN/'results.jsonl').read_text().splitlines()];review=read(RUN/'review.json');freeze=read(HERE/'freeze.json')
 if m['status']!='complete' or m['generations']!=1 or len(rows)!=1 or len(review['reviews'])!=1 or m['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Wrong capability coverage')
 for p,h in freeze['files'].items():
  if sha(ROOT/p)!=h:raise ValueError('Changed probe source')
 if m['resultsSha256']!=sha(RUN/'results.jsonl') or review['resultsSha256']!=sha(RUN/'results.jsonl'):raise ValueError('Changed probe output')
 r=rows[0];j=review['reviews'][0];ref=next(json.loads(x) for x in (ROOT/'benchmarks/runs/rewrite-formatter-v3/run/results.jsonl').read_text().splitlines() if json.loads(x)['id']==r['id'])
 if j['rowSha256']!=rowhash(r) or r['sourceControl']['rowSha256']!=rowhash(ref) or r['request']!=ref['request'] or r['editedSource']!=ref['editedSource'] or r['modelSha256']!=freeze['model']['sha256']:raise ValueError('Changed identical-request identity')
 if r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['response']['choices'][0]['finish_reason']!='stop' or r['response']['timings']['cache_n']!=0:raise ValueError('Changed unfiltered/native output')
 if m['command'][m['command'].index('--n-gpu-layers')+1]!='99':raise ValueError('Wrong requested provider')
 load=read(RUN.parent/'load-observation/manifest.json');log=(RUN.parent/'load-observation/load.log').read_text()
 if load['status']!='complete' or load['newGenerations']!=0 or 'offloaded 25/25 layers to GPU' not in log or 'MTL0' not in log:raise ValueError('Missing observed Metal placement')
 return rows,{(j['variant'],j['id']):j}
if __name__=='__main__':
 rows,js=checked();r=rows[0];print(json.dumps(dict(newGenerations=1,requestedMetalRequestMs=r['generationMs'],actualBody=r['renderedBody'],observedSupplementaryOffload='25/25',productionActivated=False),ensure_ascii=False))
