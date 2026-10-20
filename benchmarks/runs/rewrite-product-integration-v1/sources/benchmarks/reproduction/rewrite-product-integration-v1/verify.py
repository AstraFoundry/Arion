"""Verify archived product integration bytes and output-review bindings offline."""
import argparse,hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
RUN=ROOT/'benchmarks/runs/rewrite-product-integration-v1'

def digest(path):
 h=hashlib.sha256()
 with path.open('rb') as stream:
  for block in iter(lambda:stream.read(1024*1024),b''):h.update(block)
 return h.hexdigest()

def verify():
 manifest=json.loads((RUN/'archive.json').read_text())
 for name,sha in manifest['files'].items():
  path=RUN/name
  if not path.is_file() or digest(path)!=sha:raise ValueError('Changed archived source or observation: '+name)
 summary=json.loads((RUN/'summary.json').read_text())
 reviews=json.loads((RUN/'review.json').read_text());complete=0
 for review in reviews:
  p=RUN/review['source']
  rows=[json.loads(z) for z in p.read_text().splitlines()] if p.suffix=='.jsonl' else json.loads(p.read_text())
  row=rows[review['index']];response=row['response'];text=response.get('text',response.get('Text',''))
  if hashlib.sha256(text.encode()).hexdigest()!=review['outputSha256']:raise ValueError('Unbound output review')
  if not review['reviewed']:raise ValueError('Unreviewed observation')
  if text:complete+=1
 if complete!=summary['completedTexts']:raise ValueError('Wrong completed text scope')
 if summary['independentHeldoutCases']!=0 or summary['actualAsrRuns']!=0 or summary['actualOsOcrRuns']!=0 or summary['foreignEditorCommits']!=0:raise ValueError('Overclaimed native integration scope')
 receipts=[json.loads((RUN/name).read_text()) for name in ['removed-weights.json','retired-installed-model.json']]
 if any(not x.get('removed') for x in receipts[0]['files']) or not receipts[1]['removed']:raise ValueError('Incomplete weight retirement')
 if sum(f['bytes'] for r in receipts for f in r['files'])!=summary['retiredWeightBytes']:raise ValueError('Weight accounting mismatch')
 print(json.dumps(dict(archivedFiles=len(manifest['files']),reviewRecords=len(reviews),completedTexts=complete,verified=True)))

if __name__=='__main__':verify()
