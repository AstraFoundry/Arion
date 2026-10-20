"""Replay versioned full-chain evidence with explicit prompt ownership and source plans."""
import pathlib,json,hashlib,math
from gguf_template_identity import identity

def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def validate(root,here,pipeline,plans,rows,manifest,review,outcomes,decision,directory):
 freeze=read(here/'freeze.json');samples={(arm,item['sample']['id']):(item['sample'],item['sourcePath']) for arm,items in plans for item in items};count=sum(len(items) for _,items in plans)
 if len(samples)!=count:raise ValueError('Duplicate planned source')
 for path,h in freeze['files'].items():
  if sha(root/path)!=h:raise ValueError('Changed frozen source: '+path)
 if manifest['status']!='complete' or manifest['generations']!=len(rows) or manifest['formatterGenerations']!=count or manifest['attemptedSources']!=count or manifest['stageProtocolFailures']!=0 or len(outcomes)!=count:raise ValueError('Incomplete actual observation scope')
 if manifest['newAsrGenerations'] or manifest['newOcrGenerations'] or manifest['model']!=freeze['model'] or manifest['runtimeFiles']!=freeze['runtimeFiles'] or manifest['cleanupWorkerSha256']!=freeze['cleanupWorkerSha256'] or manifest['freezeSha256']!=sha(here/'freeze.json'):raise ValueError('Changed native/source identity')
 expected_workers={(arm,s['scene']) for (arm,_),(s,_) in samples.items()}
 if {(w['arm'],w['scene']) for w in manifest['workers']}!=expected_workers or len(manifest['workers'])!=len(expected_workers):raise ValueError('Changed actual native workers')
 for w in manifest['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed runtime treatment')
  if 'offloaded 25/25 layers to GPU' not in (directory/(w['name']+'-server.log')).read_text():raise ValueError('Missing actual Metal placement')
 labels={(j['variant'],j['id']):j for j in review['reviews']};selected={};finals={}
 if len(labels)!=len(rows):raise ValueError('Incomplete unique review')
 for r in rows:
  key=(r['arm'],r['id']);sample,path=samples[key];j=labels[(r['variant'],r['id'])];cleanup=r['cleanup'];data=r['source'].encode();cursor=0;pieces=[]
  if any(r[k]!=v for k,v in sample.items()) or r['sourceControl']!=dict(path=path,rowSha256=rowhash(sample)) or j['rowSha256']!=rowhash(r):raise ValueError('Changed source/review')
  if r['cleanupObservation']['cleanup']!=cleanup or r['cleanupObservation']['source']!=r['source'] or r['cleanupObservation']['id']!=r['id'] or cleanup['version']!='speech-spans-v2':raise ValueError('Changed actual cleanup')
  for span in cleanup.get('spans') or []:
   if not cursor<=span['start']<span['end']<=len(data):raise ValueError('Invalid cleanup span')
   pieces.append(data[cursor:span['start']]);cursor=span['end']
  pieces.append(data[cursor:])
  if b''.join(pieces).decode().strip()!=cleanup['text']:raise ValueError('Wrong cleanup replay')
  request,prepared=pipeline.selection_payload(r['arm'],sample,cleanup);raw=r['response']['choices'][0]['message']['content']
  if r['prepared']!=prepared or r['modelSha256']!=freeze['model']['sha256'] or r['renderError'] is not None or r['response']['choices'][0]['finish_reason']!='stop':raise ValueError('Changed producer/model/protocol')
  if r['stage']=='selection':
   if key in selected or not prepared['hypotheses'] or r['request']!=request or r['actualContextUsed']!=sample['context']:raise ValueError('Changed actual selector request')
   edited,edits=pipeline.candidates_for(r['arm']).apply(raw,prepared)
   if r['selection']!=json.loads(raw) or r['editedSource']!=edited or r['edits']!=edits:raise ValueError('Changed selected edits')
   selected[key]=r
  elif r['stage']=='format':
   prev=selected.get(key);called=bool(prepared['hypotheses'])
   if called!=(prev is not None):raise ValueError('Missing selector stage')
   edited=prev['editedSource'] if called else cleanup['text'];choice=prev['selection'] if called else {};edits=prev['edits'] if called else [];request,encoding=pipeline.format_payload(sample,edited,prepared,choice,edits)
   if key in finals or r['request']!=request or r['encoding']!=encoding or r['actualContextUsed']!={} or r['editedSource']!=edited or r['edits']!=edits or r['renderedBody']!=raw or r['selectionRowSha256']!=(rowhash(prev) if called else None) or r['sourcePreparation']['modelSelectorCalled'] is not called:raise ValueError('Changed complete unfiltered delivery')
   if j['faithful'] is not True and j['sceneUseful']:raise ValueError('Error/unknown scored useful')
   if r['pipelineMs']<r['generationMs'] or r['cleanupIpcMs']<0:raise ValueError('Invalid observed timing')
   finals[key]=r
  else:raise ValueError('Unknown stage')
  expected=identity(root,freeze,pipeline,r['arm'],r['stage'],sample['scene'],r['request'])
  if any(r[k]!=v for k,v in expected.items()):raise ValueError('Changed explicit actual template identity')
 if set(finals)!=set(samples) or len(selected)!=manifest['selectorGenerations'] or len(selected)+count!=len(rows) or len({(o['arm'],o['id']) for o in outcomes})!=count:raise ValueError('Incomplete source/stage coverage')
 for o in outcomes:
  key=(o['arm'],o['id']);prev=selected.get(key)
  if o['status']!='generated' or o['sourceRowSha256']!=rowhash(samples[key][0]) or o['finalRowSha256']!=rowhash(finals[key]) or o['selectionRowSha256']!=(rowhash(prev) if prev else None) or o['modelSelectorCalled']!=(prev is not None):raise ValueError('Changed source outcome')
 if decision['productionActivated'] or decision['freshExecuted']:raise ValueError('Unsupported production/fresh claim')
 return labels

def checked(root,here,pipeline,plans,directory):
 rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()));m=read(directory/'manifest.json');j=read(directory/'review.json');o=list(map(json.loads,(directory/'case-outcomes.jsonl').read_text().splitlines()));d=read(directory/'selection.json')
 if any(h!=sha(directory/'results.jsonl') for h in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or m['outcomesSha256']!=sha(directory/'case-outcomes.jsonl') or d['reviewSha256']!=sha(directory/'review.json') or d['freezeSha256']!=sha(here/'freeze.json'):raise ValueError('Changed completed evidence bytes')
 return rows,validate(root,here,pipeline,plans,rows,m,j,o,d,directory)

def summarize(rows,labels):
 groups={}
 for arm,cohort in dict.fromkeys((r['arm'],r['cohort']) for r in rows if r['stage']=='format'):
  subset=[r for r in rows if r['stage']=='format' and r['arm']==arm and r['cohort']==cohort];js=[labels[(r['variant'],r['id'])] for r in subset]
  groups[arm+'/'+cohort]=dict(cases=len(subset),faithful=sum(j['faithful'] is True for j in js),unfaithful=sum(j['faithful'] is False for j in js),uncertain=sum(j['faithful'] is None for j in js),useful=sum(j['sceneUseful'] for j in js),repairSucceeded=sum(j.get('repairSucceeded') is True for j in js),harmfulEditCases=sum(bool(r['edits']) and j.get('editSupported') is False for r,j in zip(subset,js)),pipelineMs={str(p):sorted(r['pipelineMs'] for r in subset)[math.ceil(p*len(subset))-1] for p in [.5,.95,.99]})
 return groups
