"""Verify completeness against the original archived enumeration snapshot."""
import pathlib,json,hashlib
ROOT=pathlib.Path(__file__).resolve().parents[3];CORPUS=ROOT/'benchmarks/corpora/rewrite-history-v1'
def checked():
 m=json.loads((CORPUS/'manifest.json').read_text());data=(CORPUS/'enumeration-view.json').read_bytes()
 if hashlib.sha256(data).hexdigest()!=m['selectionView']['sha256']:raise ValueError('Changed original enumeration snapshot')
 seen=[];keys=set()
 for row in json.loads(data)['cases']:
  if row.get('role')=='warmup' or row.get('stage')=='selection':continue
  value=[row['source'],row['context'],row['scene']];key=json.dumps(value,ensure_ascii=False,sort_keys=True)
  if key not in keys:keys.add(key);seen.append(value)
 inputs=json.loads((CORPUS/'inputs.json').read_text())
 if seen!=[[s['source'],s['context'],s['scene']] for s in inputs] or len(seen)!=401:raise ValueError('Omitted or changed historical tuple')
 return len(seen)
if __name__=='__main__':print('Verified all',checked(),'historical tuples against original enumeration bytes; zero inference')
