"""Verify all pre-inference source-review identities without inference or audio access."""
import hashlib,json,pathlib
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-natural-text-review-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def checked():
 corpus=ROOT/'benchmarks/corpora/rewrite-natural-asr-v1';manifest=read(RUN/'manifest.json');rows=read(RUN/'review.json');source=read(corpus/'inputs.json');references={r['id']:r for r in read(corpus/'evaluator.json')}
 for path,key in [(corpus/'inputs.json','inputSha256'),(corpus/'evaluator.json','evaluatorSha256'),(RUN/'review.json','reviewSha256')]:
  if sha(path)!=manifest[key]:raise ValueError('Changed natural text review identity')
 if len(rows)!=120 or len({r['id'] for r in rows})!=120 or [r['id'] for r in rows]!=[r['id'] for r in source]:raise ValueError('Incomplete source review')
 for row,sample in zip(rows,source):
  ref=references[row['id']]
  if row['source']!=sample['source'] or row['sourceSha256']!=hashlib.sha256(sample['source'].encode()).hexdigest() or row['referenceText']!=ref['referenceText'] or row['sourceSpeakerId']!=ref['sourceSpeakerId'] or row['originalSplit']!=ref['originalSplit'] or row['language']!=sample['language']:raise ValueError('Changed reviewed source/reference lineage')
  if not row['assistantTextReview'] or any(row[k] is not False for k in ['independentHumanReviewed','audioReheard','modelInputMayContainReference']):raise ValueError('Unsupported review attestation')
 cleanup=read(RUN/'cleanup-observations.json');execution=read(RUN/'cleanup-execution.json')
 if execution['observationsSha256']!=sha(RUN/'cleanup-observations.json') or execution['inputSha256']!=sha(corpus/'inputs.json') or execution['sourceSnapshotProvenanceSha256']!=sha(ROOT/execution['sourceSnapshotProvenance']):raise ValueError('Changed actual cleanup lineage')
 if len(cleanup)!=120 or [r['id'] for r in cleanup]!=[r['id'] for r in source] or any(r['source']!=s['source'] or r['cleanup']['text']!=s['source'] or r['cleanup'].get('spans',[]) for r,s in zip(cleanup,source)):raise ValueError('Changed source cleanup observation')
 archive=RUN/'archive.json'
 if archive.exists():
  for path,digest in read(archive)['files'].items():
   if sha(ROOT/path)!=digest:raise ValueError('Changed source review archive')
 return manifest
if __name__=='__main__':print('Verified',checked()['reviewed'],'assistant text source reviews; no independent ratings or new inference')
