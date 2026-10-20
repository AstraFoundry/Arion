"""Validate complete selector-only coverage without inventing final-body evidence."""
import pathlib,importlib.util,sys,json,hashlib,argparse
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run'
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
before=list(sys.path);sys.path.insert(0,str(HERE.parent))
try:ownership=module('screen_identity_replay',HERE.parent/'gguf_template_identity.py')
finally:sys.path[:]=before
pipeline=module('screen_pipeline_replay',HERE/'pipeline.py')
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(x):return hashlib.sha256(json.dumps(x,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
SOURCES=['benchmarks/corpora/rewrite-history-v1/inputs.json','benchmarks/corpora/rewrite-tone-controls-v1/inputs.json','benchmarks/corpora/rewrite-semantic-roles-v1/inputs.json']
def samples():return {s['id']:(s,p) for p in SOURCES for s in read(ROOT/p)}
def targets():return {x['id']:x for x in read(ROOT/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json')}
def assessment(row):
 allowed=targets()[row['id']]['allowedChoices']
 if row['renderError'] is not None:return dict(conforms=False,harmful=False,missed=False,required=0,repaired=0)
 choice=row['selection'];wrong={k:v for k,v in choice.items() if v not in allowed[k]};required=targets()[row['id']]['requiredRepairs']
 return dict(conforms=not wrong,harmful=any(v=='replace' for v in wrong.values()),missed=any(v=='keep' for v in wrong.values()),required=len(required),repaired=sum(choice[k]=='replace' for k in required))

def validate(rows,m,j,outcomes,preps,decision,directory=RUN):
 f=read(HERE/'freeze.json');expected=samples();evals=targets()
 for p,h in f['files'].items():
  if sha(ROOT/p)!=h:raise ValueError('Changed frozen screen source: '+p)
 if m['arms']!=pipeline.ARMS or m['status']!='complete' or m['generations']!=len(rows) or len(rows)!=404 or m['selectorGenerations']!=404 or m['formatterGenerations']!=0 or m['newAsrGenerations'] or m['newOcrGenerations'] or m['sourcePreparations']!=457 or m['attemptedSources']!=1828 or len(preps)!=457 or len(outcomes)!=1828 or m['skippedSourceTreatments']!=1424:raise ValueError('Incomplete selector-only scope')
 if m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles'] or m['cleanupWorkerSha256']!=f['cleanupWorkerSha256'] or m['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed actual runtime/source identity')
 ps={}
 for p in preps:
  s,path=expected[p['sample']['id']];c=p['cleanupObservation']['cleanup'];raw=s['source'].encode();cursor=0;pieces=[]
  if p['sample']!=s or p['sourceControl']!=dict(path=path,rowSha256=rowhash(s)) or p['cleanupObservation']['id']!=s['id'] or p['cleanupObservation']['source']!=s['source'] or c['version']!='speech-spans-v2':raise ValueError('Changed actual source preparation')
  for span in c.get('spans') or []:
   if not cursor<=span['start']<span['end']<=len(raw):raise ValueError('Invalid actual cleanup span')
   pieces.append(raw[cursor:span['start']]);cursor=span['end']
  pieces.append(raw[cursor:])
  if b''.join(pieces).decode().strip()!=c['text']:raise ValueError('Wrong actual cleanup replay')
  _,prepared=pipeline.selection_payload('baseline',s,c)
  if prepared!=p['prepared'] or s['id'] in ps:raise ValueError('Changed shared candidate preparation')
  ps[s['id']]=p
 if set(ps)!=set(expected):raise ValueError('Missing prepared source')
 active={i for i,p in ps.items() if p['prepared']['hypotheses']}
 if active!=set(evals):raise ValueError('Changed candidate-bearing evaluation scope')
 reference={r['id']:r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-semantic-v2/run/results.jsonl').read_text().splitlines()) if r['stage']=='selection'}
 for i,e in evals.items():
  if e['sourceRowSha256']!=rowhash(expected[i][0]) or e['referenceRowSha256']!=rowhash(reference[i]):raise ValueError('Changed evaluator source/reference binding')
 workers={(a,ps[i]['sample']['scene']) for a in pipeline.ARMS for i in active}
 if {(w['arm'],w['scene']) for w in m['workers']}!=workers or len(m['workers'])!=len(workers):raise ValueError('Missing actual scene worker')
 for w in m['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed screen runtime treatment')
  if 'offloaded 25/25 layers to GPU' not in (directory/(w['name']+'-server.log')).read_text():raise ValueError('Missing actual GPU placement')
 labels={(x['variant'],x['id']):x for x in j['reviews']};actual={}
 if len(labels)!=len(rows):raise ValueError('Incomplete unique reviews')
 for r in rows:
  key=(r['arm'],r['id']);s,path=expected[r['id']];p=ps[r['id']];c=p['cleanupObservation']['cleanup'];request,prepared=pipeline.selection_payload(r['arm'],s,c);label=labels[(r['variant'],r['id'])]
  if key in actual or r['id'] not in active or r['stage']!='selection' or r['renderedBody'] is not None or 'pipelineMs' in r or any(r[k]!=v for k,v in s.items()):raise ValueError('Unexpected/duplicate inference or invented final-body timing')
  if r['sourceControl']!=p['sourceControl'] or r['preparationRowSha256']!=rowhash(p) or r['cleanupObservation']!=p['cleanupObservation'] or r['cleanup']!=c or r['prepared']!=prepared or r['actualContextUsed']!=s['context'] or r['modelSha256']!=f['model']['sha256'] or label['rowSha256']!=rowhash(r):raise ValueError('Changed actual source/review identity')
  if r['sourcePreparation']!=dict(kind='actual-go-cleanup-reused-within-selector-screen',modelSelectorCalled=True,formatterExecuted=False) or r['cleanupIpcMs']!=p['cleanupIpcMs'] or p['cleanupIpcMs']<0:raise ValueError('Changed actual preparation/execution scope')
  if r['request']!=request or r['response']['choices'][0]['finish_reason']!='stop':raise ValueError('Changed actual selector treatment')
  identity=ownership.identity(ROOT,f,pipeline,r['arm'],'selection',s['scene'],request)
  if any(r[k]!=v for k,v in identity.items()):raise ValueError('Changed actual prompt ownership')
  raw=r['response']['choices'][0]['message']['content']
  if r['renderError'] is None:
   edited,edits=pipeline.candidates_for(r['arm']).apply(raw,prepared)
   if r['selection']!=json.loads(raw) or r['editedSource']!=edited or r['edits']!=edits:raise ValueError('Changed actual choices/edits')
  elif r['selection'] is not None or r['editedSource'] is not None:raise ValueError('Invalid choice substituted source')
  result=assessment(r)
  if label['faithful']!=(None if r['renderError'] is not None else not result['harmful']) or label['choiceConforms']!=result['conforms'] or label['harmfulChoice']!=result['harmful'] or label['missedRepair']!=result['missed'] or label['sceneUseful'] is not None:raise ValueError('Review contradicts frozen evaluator or invents final style')
  if r['arm']!='baseline' and pipeline.baseline_projection(request,r['arm'])!=actual[('baseline',r['id'])]['request']:raise ValueError('Treatment changed undeclared fields')
  actual[key]=r
 if set(actual)!={(a,i) for a in pipeline.ARMS for i in active} or m['stageProtocolFailures']!=sum(r['renderError'] is not None for r in rows):raise ValueError('Incomplete candidate inference coverage')
 seen=set()
 for o in outcomes:
  key=(o['arm'],o['id']);p=ps[o['id']];called=o['id'] in active;seen.add(key);r=actual.get(key)
  if o['sourceRowSha256']!=rowhash(p['sample']) or o['preparationRowSha256']!=rowhash(p) or o['modelSelectorCalled']!=called or o['formatterExecuted'] or o['resultRowSha256']!=(rowhash(r) if called else None) or o['status']!=('selection-invalid' if called and r['renderError'] else 'selected' if called else 'no-hypotheses'):raise ValueError('Changed complete source outcome')
 if seen!={(a,i) for a in pipeline.ARMS for i in expected} or len(seen)!=len(outcomes):raise ValueError('Missing/duplicate skipped source treatment')
 passed=[a for a in pipeline.ARMS if all(labels[(r['variant'],r['id'])]['choiceConforms'] for r in rows if r['arm']==a)]
 if decision['screenPassedArms']!=passed or decision['qualified'] or decision['productionActivated'] or decision['freshExecuted'] or decision['formatterExecuted']:raise ValueError('Unsupported screen/full/release claim')
 return labels

def checked(directory=RUN):
 rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()));m=read(directory/'manifest.json');j=read(directory/'review.json');o=list(map(json.loads,(directory/'case-outcomes.jsonl').read_text().splitlines()));p=list(map(json.loads,(directory/'preparations.jsonl').read_text().splitlines()));d=read(directory/'selection.json')
 if any(h!=sha(directory/'results.jsonl') for h in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or m['outcomesSha256']!=sha(directory/'case-outcomes.jsonl') or m['preparationsSha256']!=sha(directory/'preparations.jsonl') or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed completed screen bytes')
 labels=validate(rows,m,j,o,p,d,directory)
 references=read(directory/'reference-comparison.json')['records'];current={(r['arm'],r['id']):r for r in rows};external={}
 if len(references)!=202 or len({(x['arm'],x['id']) for x in references})!=202:raise ValueError('Incomplete stage/full control comparison')
 for x in references:
  expected_path={'baseline':'benchmarks/runs/rewrite-selector-semantic-v2/run/results.jsonl','after':'benchmarks/runs/rewrite-selector-preview-v1/run/results.jsonl'}[x['arm']]
  if x['referencePath']!=expected_path:raise ValueError('Wrong full-chain control identity')
  if x['referencePath'] not in external:external[x['referencePath']]={r['id']:r for r in map(json.loads,(ROOT/x['referencePath']).read_text().splitlines()) if r['stage']=='selection'}
  r=current[(x['arm'],x['id'])];b=external[x['referencePath']][x['id']]
  if x['rowSha256']!=rowhash(r) or x['referenceRowSha256']!=rowhash(b) or any(r[k]!=b[k] for k in ['request','selection','editedSource','prepared']) or not all(x[k] for k in ['sameRequest','sameChoice','sameEditedSource']):raise ValueError('Changed stage/full control parity')
 return rows,labels

def collect():
 rows,labels=checked();groups={}
 for arm in pipeline.ARMS:
  subset=[r for r in rows if r['arm']==arm];results=[assessment(r) for r in subset];groups[arm]=dict(cases=len(subset),conforming=sum(x['conforms'] for x in results),harmful=sum(x['harmful'] for x in results),missed=sum(x['missed'] for x in results),requiredRepairs=sum(x['required'] for x in results),repaired=sum(x['repaired'] for x in results),protocolFailures=sum(r['renderError'] is not None for r in subset))
 return dict(newGenerations=len(rows),sourcePreparations=457,sourceTreatments=1828,skippedSourceTreatments=1424,finalBodies=0,groups=groups,selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits']),rows,labels

def main():
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();summary,rows,labels=collect();a.out.mkdir(parents=True,exist_ok=False);(a.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n');lines=['# Four-arm selector-only development screen','','Every actual choice personally reviewed. All457 source preparations and1828 source treatments retained;356 zero-hypothesis sources per arm generate no model response. No formatter, final-body, native delivery, held-out or minimum16GB acceptance is claimed.','']
 for r in rows:lines += [f"## {r['arm']} / {r['id']} / {r['scene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','prepared','editedSource','edits','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',labels[(r['variant'],r['id'])]['note'],'']
 (a.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
