"""Verify all other307 scene requests remain identical without new model inference."""
import pathlib,json,importlib.util,hashlib,argparse
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2]
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
pipeline=module('unchanged_scene_v4_payload',HERE/'pipeline.py')
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def collect():
 path=ROOT/'benchmarks/runs/rewrite-history-format-v2/run/results.jsonl';rows=list(map(json.loads,path.read_text().splitlines()));choices={r['id']:r['selection'] for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-history-chain-v1/run/results.jsonl').read_text().splitlines()) if r['stage']=='selection'};records=[]
 for r in rows:
  if r['scene']=='work':continue
  request,encoding=pipeline.format_payload('candidate',r,r['editedSource'],r['prepared'],choices.get(r['id'],{}),r['edits'])
  if request!=r['request'] or encoding!=r['encoding']:raise ValueError('Other scene payload changed: '+r['id'])
  records.append(dict(id=r['id'],scene=r['scene'],referenceRowSha256=rowhash(r),actualRequestSha256=rowhash(request)))
 if len(records)!=307:raise ValueError('Incomplete other-scene coverage')
 return dict(newGenerations=0,sourceCases=307,exactRequestParity=307,referencePath=str(path.relative_to(ROOT)),referenceSha256=hashlib.sha256(path.read_bytes()).hexdigest(),records=records,scope='Zero-inference payload audit only. Other307 scenes retain identical actual old requests/encoding; old bodies remain reviewed evidence, not a new full401-order or current-chain run.')
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();result=collect()
 with a.out.open('x') as f:json.dump(result,f,ensure_ascii=False,indent=2);f.write('\n')
 print('Verified307 unchanged scene payloads;0 new generations')
