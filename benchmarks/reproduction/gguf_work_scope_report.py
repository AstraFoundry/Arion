"""Replay exact work formatter jobs, complete manual reviews and paired comparisons."""
import pathlib,sys,importlib.util,json,hashlib,argparse
ROOT=pathlib.Path(__file__).resolve().parents[2]
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
def load_jobs(here):
 before=list(sys.path);previous=sys.modules.get('pipeline');sys.path.insert(0,str(here.parent))
 try:
  sys.modules['pipeline']=module('scoped_job_payload_'+here.name,here/'pipeline.py');return module('scoped_formatter_jobs_'+here.name,here/'run.py').jobs()
 finally:
  sys.path[:]=before;sys.modules.pop('pipeline',None)
  if previous is not None:sys.modules['pipeline']=previous
class Study:
 def __init__(self,here,previous):self.here=here;self.run=ROOT/'benchmarks/runs'/here.name/'run';self.previous=previous
 def validate(self,rows,m,j,d,directory=None):
  directory=directory or self.run;f=read(self.here/'freeze.json');jobs=load_jobs(self.here);expected={(x['variant'],x['id']):x for x in jobs};protocol=read(self.here/'protocol.json')
  for path,h in f['files'].items():
   if sha(ROOT/path)!=h:raise ValueError('Changed frozen work-study source')
  if m['status']!='complete' or m['generations']!=protocol['newGenerations'] or len(rows)!=len(jobs) or len(rows)!=m['generations'] or len(j['reviews'])!=len(rows) or m['sourceCases']!=110 or m['newSelectorGenerations']!=0 or m['newAsrGenerations']!=0:raise ValueError('Wrong new/reused scope')
  if m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles'] or m['freezeSha256']!=sha(self.here/'freeze.json'):raise ValueError('Wrong native identity')
  arms=set(x['variant'] for x in jobs)
  if {w['arm'] for w in m['workers']}!=arms or len(m['workers'])!=len(arms):raise ValueError('Wrong native arm workers')
  for w in m['workers']:
   for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1')]:
    if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed actual runtime settings')
   if 'offloaded 25/25 layers to GPU' not in (directory/(w['name']+'-server.log')).read_text():raise ValueError('Missing observed GPU placement')
  labels={(x['variant'],x['id']):x for x in j['reviews']}
  if len(labels)!=len(jobs) or {(r['variant'],r['id']) for r in rows}!=set(expected):raise ValueError('Missing or duplicate output/review')
  for r in rows:
   key=(r['variant'],r['id']);seed=expected[key];label=labels[key]
   if any(r.get(k)!=v for k,v in seed.items()):raise ValueError('Changed source, reuse or actual request metadata')
   if label['rowSha256']!=rowhash(r) or r['renderError'] is not None or r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Changed reviewed/unfiltered body')
   if label['faithful'] is not True and label['faithful'] is not False and label['faithful'] is not None:raise ValueError('Invalid semantic label')
   if label['faithful'] is not True and label['sceneUseful']:raise ValueError('Uncertain/error body counted useful')
  if d['qualified'] or d['selected'] is not None or d['productionActivated'] or d['freshExecuted']:raise ValueError('Unsupported release/selection')
  return labels
 def checked(self,directory=None):
  directory=directory or self.run;rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()));m=read(directory/'manifest.json');j=read(directory/'review.json');d=read(directory/'selection.json')
  if any(h!=sha(directory/'results.jsonl') for h in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json') or d['freezeSha256']!=sha(self.here/'freeze.json'):raise ValueError('Changed evidence bytes')
  labels=self.validate(rows,m,j,d,directory)
  extra=self.here/'unchanged-scenes.py'
  if extra.exists():
   audit=module('unchanged_scene_audit_'+self.here.name,extra).collect()
   if read(directory/'unchanged-scene-request-parity.json')!=audit:raise ValueError('Changed other-scene payload audit')
  return rows,labels
 def collect(self):
  rows,js=self.checked();previous=module('work_scope_previous_'+self.here.name,self.here.parent/self.previous/'report.py');old,oldjs=previous.checked();refs={r['id']:r for r in old if r['variant']!='baseline'};groups={};transitions=[]
  for arm,cohort in dict.fromkeys((r['variant'],r['cohort']) for r in rows):
   subset=[r for r in rows if r['variant']==arm and r['cohort']==cohort];labels=[js[(r['variant'],r['id'])] for r in subset];group=dict(cases=len(subset),faithful=sum(x['faithful'] is True for x in labels),unfaithful=sum(x['faithful'] is False for x in labels),uncertain=sum(x['faithful'] is None for x in labels),useful=sum(x['sceneUseful'] for x in labels),languagePreserved=sum(x['languagePreserved'] for x in labels));groups[arm+'/'+cohort]=group
   if arm=='candidate':
    matched=[]
    for r in subset:
     ref=refs.get(r['id'])
     if ref is None:
      ref=next(x for x in rows if x['variant']=='baseline' and x['id']==r['id']);before=js[('baseline',r['id'])]
     else:before=oldjs[(ref['variant'],ref['id'])]
     after=js[(r['variant'],r['id'])];matched.append((r,ref,before))
     if (before['faithful'],before['sceneUseful'])!=(after['faithful'],after['sceneUseful']):transitions.append(dict(id=r['id'],cohort=cohort,beforeFaithful=before['faithful'],afterFaithful=after['faithful'],beforeUseful=before['sceneUseful'],afterUseful=after['sceneUseful'],note=after['note']))
    group.update(baselineFaithful=sum(b['faithful'] is True for _,_,b in matched),baselineUseful=sum(b['sceneUseful'] for _,_,b in matched),rawParity=sum(r['renderedBody']==ref['renderedBody'] for r,ref,_ in matched))
  return dict(newGenerations=len(rows),sourceCases=110,newSelectorGenerations=0,groups=groups,transitions=transitions,selection=read(self.run/'selection.json'),limits=read(self.here/'protocol.json')['limits']),rows,js
 def main(self):
  p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();s,rows,js=self.collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(s,ensure_ascii=False,indent=2)+'\n');lines=['# Complete work-scope formatter review','','New formatter generations, reused prior stages and new source-only controls are separate. Every actual response personally read. All raw bodies retained without validator rejection or source fallback.','']
  for r in rows:lines += [f"## {r['variant']} / {r['id']} / {r['cohort']}",'','```json',json.dumps({k:r[k] for k in ['source','context','editedSource','sourcePreparation','sourceControl','request']},ensure_ascii=False,indent=2),'```','','```text',r['renderedBody'],'```','',js[(r['variant'],r['id'])]['note'],'']
  (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(s,ensure_ascii=False))
