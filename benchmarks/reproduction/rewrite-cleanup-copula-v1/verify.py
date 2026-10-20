"""Verify actual deterministic cleanup observations and exact byte-span replay."""
import pathlib,json,hashlib
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-cleanup-copula-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def checked():
 for path,digest in read(RUN/'archive.json')['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed cleanup archive: '+path)
 m=read(RUN/'manifest.json');rows=read(RUN/'observations.json');inputs=read(HERE/'inputs.json')
 if m['status']!='complete' or m['newLlmGenerations'] or m['newAsrGenerations'] or len(rows)!=156:raise ValueError('Wrong cleanup evidence scope')
 if sha(HERE/'inputs.json')!=m['inputSha256'] or sha(RUN/'observations.json')!=m['observationsSha256'] or (RUN/'observations.json').read_bytes()!=(RUN/'isolated-observations.json').read_bytes():raise ValueError('Changed cleanup replay')
 for p,h in m['sourceFiles'].items():
  if sha(ROOT/p)!=h:raise ValueError('Changed snapshot')
 for r,s in zip(rows,inputs):
  if r['id']!=s['id'] or r['source']!=s['source'] or r['cleanup']['version']!='speech-spans-v2':raise ValueError('Changed source/version')
  source=r['source'].encode();cursor=0;parts=[]
  for span in r['cleanup'].get('spans',[]):
   if not cursor<=span['start']<span['end']<=len(source):raise ValueError('Invalid span')
   parts.append(source[cursor:span['start']]);cursor=span['end']
  parts.append(source[cursor:])
  if b''.join(parts).decode().strip()!=r['cleanup']['text']:raise ValueError('Changed deletion contract')
 if [r['id'] for r in rows if r['source']!=r['cleanup']['text']]!=m['changedCases']:raise ValueError('Changed scope')
 return m
if __name__=='__main__':print('Verified',checked()['observations'],'actual Go cleanup observations; zero LLM/ASR')
