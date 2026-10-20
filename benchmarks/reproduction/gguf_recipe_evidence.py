"""Shared source-bound evidence checks for versioned source-only prompt recipes."""
import pathlib,json,hashlib,math,argparse
ROOT=pathlib.Path(__file__).resolve().parents[2]
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def references(here):
 p=read(here/'protocol.json');result={}
 for kind,path in p['references'].items():
  rows=[json.loads(x) for x in (ROOT/path).read_text().splitlines()]
  result[kind]=rows if p['version']=='language-profile-v1' else [r for r in rows if r['evaluationSet']==kind]
 return result

def validate(here,pipeline,rows,m,review,decision,directory):
 f=read(here/'freeze.json');p=read(here/'protocol.json');refs=references(here);expected=[(kind,r) for kind in ['development','natural'] for r in refs[kind]]
 for name,h in f['files'].items():
  if sha(ROOT/name)!=h:raise ValueError('Changed frozen recipe/source')
 if m['status']!='complete' or m['generations']!=156 or m['sourceCases']!=156 or len(rows)!=156 or len(review['reviews'])!=156 or m['newSelectorGenerations']!=0 or m['newAsrGenerations']!=0:raise ValueError('Wrong actual generation coverage')
 if m['freezeSha256']!=sha(here/'freeze.json') or m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles']:raise ValueError('Changed recipe model/runtime')
 js={(j['variant'],j['id']):j for j in review['reviews']}
 if len(js)!=156 or len(m['workers'])!=5:raise ValueError('Incomplete review/workers')
 for w in m['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1'),('-lv','5')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed provider settings')
  if 'offloaded 25/25 layers to GPU' not in (directory/(w['evaluationSet']+'-'+w['scene']+'-server.log')).read_text():raise ValueError('Missing placement in actual worker')
 selections={r['id']:r for r in [json.loads(x) for x in (ROOT/'benchmarks/runs/rewrite-bounded-edit-v2/run/results.jsonl').read_text().splitlines()] if r['stage']=='selection'}
 for r,(kind,c) in zip(rows,expected):
  j=js[(r['variant'],r['id'])]
  if r['evaluationSet']!=kind or r['sourceControl']['path']!=p['references'][kind] or r['sourceControl']['rowSha256']!=rowhash(c) or any(r[k]!=c[k] for k in ['id','source','context','scene','language','editedSource','edits','cleanup','modelSha256']):raise ValueError('Changed source/control')
  if kind=='development':prepared,choices=selections[r['id']]['prepared'],selections[r['id']]['selection']
  else:
   prepared=pipeline.candidates.prepare(r['editedSource'],r['context'],r['scene']);choices={}
   if prepared['hypotheses'] or r['sourcePreparation']['modelSelectorCalled'] is not False:raise ValueError('Simulated natural selector')
  request,e=pipeline.format_payload(c,c['editedSource'],prepared,choices,c['edits'])
  if r['request']!=request or r['encoding']!=e or r['actualContextUsed']!={} or r['effectiveModelScene']!=r['scene'] or r['stage']!='format':raise ValueError('Changed script routing/payload/context')
  if r['templateSha256']!=sha(here/'prompts.json') or j['rowSha256']!=rowhash(r):raise ValueError('Changed reviewed output/prompt')
  if r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['renderError'] is not None or r['response']['timings']['cache_n']!=0 or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Changed unfiltered native response')
  if j['faithful'] is not True and j['faithful'] is not False and j['faithful'] is not None:raise ValueError('Invalid meaning label')
  if j['sceneUseful'] and j['faithful'] is not True:raise ValueError('Unknown meaning counted as usable')
 dev=[r for r in rows if r['evaluationSet']=='development'];dj=[js[(r['variant'],r['id'])] for r in dev]
 eligible=all(j['faithful'] is True for j in dj) and sum(j['sceneUseful'] for j in dj)>=35 and sum(j['repairSucceeded'] is True for j in dj)==11 and all(js[(r['variant'],r['id'])]['sceneUseful'] for r in dev if r['id'] in ['owner-anchor','email-anchor']) and all(r['response']['choices'][0]['finish_reason']=='stop' for r in dev)
 if decision['developmentQualified']!=eligible or decision['qualified'] or decision['selected'] is not None or decision['productionActivated'] or decision['freshExecuted'] or decision['freezeSha256']!=sha(here/'freeze.json'):raise ValueError('Wrong development/production decision boundary')
 return js

def checked(here,pipeline,directory):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');review=read(directory/'review.json');d=read(directory/'selection.json')
 if any(v!=sha(directory/'results.jsonl') for v in [m['resultsSha256'],review['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed reviewed bytes')
 return rows,validate(here,pipeline,rows,m,review,d,directory)
def collect(here,pipeline,directory):
 rows,js=checked(here,pipeline,directory)
 def summary(group):
  labels=[js[(r['variant'],r['id'])] for r in group];times=sorted(r['generationMs'] for r in group)
  return dict(observations=len(group),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),sceneReady=sum(j['sceneUseful'] for j in labels),repairs=sum(j['repairSucceeded'] is True for j in labels),languagePreserved=sum(j.get('languagePreserved',True) for j in labels),formatterMs={str(p):times[math.ceil(p*len(times))-1] for p in [.5,.95,.99]})
 return dict(newGenerations=156,groups={kind:summary([r for r in rows if r['evaluationSet']==kind]) for kind in ['development','natural']},selection=read(directory/'selection.json'),productionActivated=False),rows,js
def main(here,pipeline,directory):
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect(here,pipeline,directory);out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Full paired source/language recipe review','','All156 actual complete outputs personally reviewed.36 scene probes and120 natural sources remain separate. No references/old bodies in model data. Uncertain meaning and copied-source usability are not passing quality.','']
 for r in rows:lines += [f"## {r['evaluationSet']} / {r['id']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','actualContextUsed','sourceControl','editedSource','edits','request','encoding']},ensure_ascii=False,indent=2),'```','','```text',r['renderedBody'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False))
