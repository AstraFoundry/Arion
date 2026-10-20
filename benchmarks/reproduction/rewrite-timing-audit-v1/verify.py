"""Recompute a fixed native/external-clock consistency snapshot without inference."""
import pathlib,json,hashlib
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2]
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def checked():
 snapshot=read(HERE/'consistency.json');issues=[];count=0
 for relative,digest in snapshot['sources'].items():
  path=ROOT/relative
  if sha(path)!=digest:raise ValueError('Changed original timing observation '+relative)
  for line in path.read_text().splitlines():
   row=json.loads(line);timing=row.get('response',{}).get('timings');external=row.get('generationMs')
   if not timing or external is None:continue
   count+=1;native=timing['prompt_ms']+timing['predicted_ms']
   if native-external>max(25,external*.05):issues.append(dict(path=relative,id=row['id'],variant=row['variant'],rowSha256=hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest(),createdEpochSeconds=row['response'].get('created'),externalMs=external,nativeStageTotalMs=native,gapMs=native-external))
 if count!=snapshot['observations'] or len(issues)!=snapshot['flagged'] or issues!=snapshot['issues']:raise ValueError('Changed derived timing consistency audit')
 for record in read(HERE/'source-provenance.json'):
  if sha(HERE/'sources'/record['path'])!=record['sha256']:raise ValueError('Changed exact-build source')
 archive=HERE/'archive.json'
 if archive.exists():
  for path,digest in read(archive)['files'].items():
   if sha(ROOT/path)!=digest:raise ValueError('Changed timing-audit archive')
 return snapshot
if __name__=='__main__':
 data=checked();print('Verified clock consistency snapshot:',data['flagged'],'flagged pairs among',data['observations'],'completed observations; no new inference')
