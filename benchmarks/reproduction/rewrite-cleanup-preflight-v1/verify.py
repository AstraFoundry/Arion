"""Verify source snapshots and exact deletion-span replay without model inference."""
import pathlib,json,hashlib
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-cleanup-preflight-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def checked():
 archive=RUN/'archive.json'
 if archive.exists():
  for path,digest in read(archive)['files'].items():
   if sha(ROOT/path)!=digest:raise ValueError('Changed archived cleanup evidence: '+path)
 provenance=read(RUN/'provenance.json');rows=read(RUN/'observations.json');inputs=read(HERE/'inputs.json')
 if sha(HERE/'inputs.json')!=provenance['inputSha256'] or sha(RUN/'observations.json')!=provenance['observationsSha256']:raise ValueError('Changed preflight data')
 for item in provenance['sourceSnapshots']:
  if sha(ROOT/item['archivedPath'])!=item['sha256']:raise ValueError('Changed frozen production source')
 expected={r['id']:r['source'] for r in inputs}
 if len(rows)!=36 or {r['id'] for r in rows}!=set(expected):raise ValueError('Wrong preflight coverage')
 for row in rows:
  if row['source']!=expected[row['id']]:raise ValueError('Changed raw input')
  source=row['source'].encode();cursor=0;parts=[]
  for span in row['cleanup'].get('spans',[]):
   if span['start']<cursor or span['end']<=span['start'] or span['end']>len(source):raise ValueError('Invalid frozen span')
   parts.append(source[cursor:span['start']]);cursor=span['end']
  parts.append(source[cursor:])
  if b''.join(parts).decode().strip()!=row['cleanup']['text']:raise ValueError('Span replay mismatch')
 if [r['id'] for r in rows if r['source']!=r['cleanup']['text']]!=provenance['changedCases']:raise ValueError('Changed transformation summary')
 return provenance
if __name__=='__main__':
 data=checked();print('Verified36 production-cleanup observations; one changed source, zero new model inference')
