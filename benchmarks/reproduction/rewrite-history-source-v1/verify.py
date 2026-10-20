"""Verify all historical inputs against exact authoritative original raw rows."""
import pathlib,json,hashlib
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];CORPUS=ROOT/'benchmarks/corpora/rewrite-history-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def checked():
 m=read(CORPUS/'manifest.json');inputs=read(CORPUS/'inputs.json');lineage=read(CORPUS/'lineage.json')['records'];cache={}
 if m['status']!='complete' or m['newLlmGenerations'] or m['newAsrGenerations'] or len(inputs)!=401 or len(lineage)!=401 or sha(CORPUS/'inputs.json')!=m['inputsSha256'] or sha(CORPUS/'lineage.json')!=m['lineageSha256']:raise ValueError('Changed corpus scope/identity')
 for p,h in m['sourceFiles'].items():
  if sha(ROOT/p)!=h:raise ValueError('Changed authoritative input file')
 for sample,ref in zip(inputs,lineage):
  if ref['path'] not in cache:cache[ref['path']]=[json.loads(x) for x in (ROOT/ref['path']).read_text().splitlines()]
  row=cache[ref['path']][ref['line']-1]
  if sample['id']!=ref['id'] or rowhash(sample)!=ref['inputSha256'] or rowhash(row)!=ref['rawRowSha256'] or row['id']!=ref['originalId'] or row.get('variant',row.get('arm'))!=ref['variant'] or any(sample[k]!=row.get(k,{} if k=='context' else None) for k in ['source','context','scene']):raise ValueError('Changed exact source/context/scene')
 keys=[json.dumps([s['source'],s['context'],s['scene']],ensure_ascii=False,sort_keys=True) for s in inputs]
 if len(set(keys))!=401 or sum(s['cohort']=='historical-natural-read-speech' for s in inputs)!=120:raise ValueError('Changed deduplication/strata')
 return m
if __name__=='__main__':print('Verified',checked()['sources'],'source/context/scene inputs; zero inference')
