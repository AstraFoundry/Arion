"""Replay frozen semantic selector experiment, including its raw metadata defect."""
import pathlib,json,hashlib,importlib.util,argparse,math
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
RUN=ROOT/'benchmarks/runs/rewrite-selector-semantic-v1/run'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
spec=importlib.util.spec_from_file_location('semantic_replay_pipeline',HERE/'pipeline.py');pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)
def validate(rows,m,review,outcomes,decision,directory=RUN):
 freeze=read(HERE/'freeze.json')
 for path,h in freeze['files'].items():
  if sha(ROOT/path)!=h:raise ValueError('Changed frozen experiment source: '+path)
 samples={}
 for path,arms in [('benchmarks/corpora/rewrite-history-v1/inputs.json',['candidate']),('benchmarks/corpora/rewrite-tone-controls-v1/inputs.json',['candidate']),('benchmarks/corpora/rewrite-semantic-roles-v1/inputs.json',['baseline','candidate'])]:
  for s in read(ROOT/path):
   for arm in arms:samples[(arm,s['id'])]=(s,path)
 if m['status']!='complete' or m['generations']!=len(rows) or len(rows)!=609 or m['formatterGenerations']!=489 or m['selectorGenerations']!=120 or m['attemptedSources']!=489 or m['stageProtocolFailures']!=0 or len(outcomes)!=489:raise ValueError('Incomplete actual observations')
 if m['newAsrGenerations'] or m['newOcrGenerations'] or m['model']!=freeze['model'] or m['runtimeFiles']!=freeze['runtimeFiles'] or m['cleanupWorkerSha256']!=freeze['cleanupWorkerSha256'] or m['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed runtime/source identity')
 if {(w['arm'],w['scene']) for w in m['workers']}!={(arm,s['scene']) for (arm,_),(s,_) in samples.items()}:raise ValueError('Missing native worker')
 for w in m['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed runtime treatment')
  if 'offloaded 25/25 layers to GPU' not in (directory/(w['name']+'-server.log')).read_text():raise ValueError('Missing Metal placement')
 labels={(j['variant'],j['id']):j for j in review['reviews']};selected={};finals={}
 if len(labels)!=len(rows):raise ValueError('Incomplete unique review')
 for r in rows:
  key=(r['arm'],r['id']);s,path=samples[key];j=labels[(r['variant'],r['id'])];cleanup=r['cleanup'];data=r['source'].encode();cursor=0;pieces=[]
  if any(r[k]!=v for k,v in s.items()) or r['sourceControl']!=dict(path=path,rowSha256=rowhash(s)) or j['rowSha256']!=rowhash(r):raise ValueError('Changed source/review')
  if r['cleanupObservation']['cleanup']!=cleanup or r['cleanupObservation']['source']!=r['source'] or r['cleanupObservation']['id']!=r['id'] or cleanup['version']!='speech-spans-v2':raise ValueError('Changed actual cleanup')
  for span in cleanup.get('spans') or []:
   if not cursor<=span['start']<span['end']<=len(data):raise ValueError('Invalid cleanup span')
   pieces.append(data[cursor:span['start']]);cursor=span['end']
  pieces.append(data[cursor:])
  if b''.join(pieces).decode().strip()!=cleanup['text']:raise ValueError('Wrong cleanup replay')
  request,prepared=pipeline.selection_payload(r['arm'],s,cleanup);raw=r['response']['choices'][0]['message']['content']
  if r['prepared']!=prepared or r['modelSha256']!=freeze['model']['sha256'] or r['renderError'] is not None or r['response']['choices'][0]['finish_reason']!='stop':raise ValueError('Changed producer/model/protocol')
  if r['stage']=='selection':
   if key in selected or not prepared['hypotheses'] or r['request']!=request or r['actualContextUsed']!=s['context']:raise ValueError('Changed actual selector request')
   edited,edits=pipeline.candidates_for(r['arm']).apply(raw,prepared)
   if r['selection']!=json.loads(raw) or r['editedSource']!=edited or r['edits']!=edits or r['templateVersion']!='bounded-edit-v2-selection' or r['templateSha256']!=sha(HERE.parent/'rewrite-bounded-edit-v2/prompts.json'):raise ValueError('Changed raw selection/known metadata')
   selected[key]=r
  elif r['stage']=='format':
   prev=selected.get(key);called=bool(prepared['hypotheses'])
   if called!=(prev is not None):raise ValueError('Missing selector stage')
   edited=prev['editedSource'] if called else cleanup['text'];choice=prev['selection'] if called else {};edits=prev['edits'] if called else [];request,encoding=pipeline.format_payload(s,edited,prepared,choice,edits)
   if key in finals or r['request']!=request or r['encoding']!=encoding or r['actualContextUsed']!={} or r['editedSource']!=edited or r['edits']!=edits or r['renderedBody']!=raw or r['selectionRowSha256']!=(rowhash(prev) if called else None) or r['sourcePreparation']['modelSelectorCalled'] is not called:raise ValueError('Changed complete unfiltered delivery')
   if j['faithful'] is not True and j['sceneUseful']:raise ValueError('Error/unknown scored useful')
   finals[key]=r
  else:raise ValueError('Unknown stage')
 if set(finals)!=set(samples) or len(selected)!=120 or len({(o['arm'],o['id']) for o in outcomes})!=489:raise ValueError('Incomplete coverage')
 for o in outcomes:
  key=(o['arm'],o['id']);prev=selected.get(key)
  if o['status']!='generated' or o['sourceRowSha256']!=rowhash(samples[key][0]) or o['finalRowSha256']!=rowhash(finals[key]) or o['selectionRowSha256']!=(rowhash(prev) if prev else None) or o['modelSelectorCalled']!=(prev is not None):raise ValueError('Changed source outcome')
 if decision['qualified'] or decision['selected'] is not None or decision['productionActivated'] or decision['freshExecuted']:raise ValueError('Unsupported adoption claim')
 return labels

def checked(directory=RUN):
 rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()));m=read(directory/'manifest.json');j=read(directory/'review.json');o=list(map(json.loads,(directory/'case-outcomes.jsonl').read_text().splitlines()));d=read(directory/'selection.json');a=read(directory/'selector-template-provenance.json')
 if any(x!=sha(directory/'results.jsonl') for x in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256'],a['resultsSha256']]) or m['outcomesSha256']!=sha(directory/'case-outcomes.jsonl') or d['reviewSha256']!=sha(directory/'review.json') or d['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed evidence bytes')
 amendment=read(directory/'review-amendment.json')
 if amendment['initialReviewSha256']!=sha(directory/'review-initial.json') or amendment['currentReviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed review amendment')
 labels=validate(rows,m,j,o,d,directory);selected={r['id']:r for r in rows if r['stage']=='selection' and r['arm']=='candidate'}
 if len(a['records'])!=101 or len({x['id'] for x in a['records']})!=101:raise ValueError('Incomplete metadata correction')
 for x in a['records']:
  r=selected[x['id']]
  if x['rowSha256']!=rowhash(r) or x['requestSha256']!=rowhash(r['request']) or x['recordedVersion']!=r['templateVersion'] or x['recordedSha256']!=r['templateSha256'] or x['effectiveVersion']!='selector-semantic-v1-selection' or x['effectivePromptPath']!='benchmarks/reproduction/rewrite-selector-semantic-v1/prompts.json' or x['effectivePromptSha256']!=sha(HERE/'prompts.json'):raise ValueError('Unbound effective prompt identity')
 refs={r['id']+'|'+r['stage']:r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-tone-chain-v1/run/results.jsonl').read_text().splitlines()) if r['arm']=='candidate'};current={r['id']+'|'+r['stage']:r for r in rows if not r['id'].startswith('role-')};comparison=read(directory/'reference-comparison.json')['records']
 if len(comparison)!=len(current) or len({x['id']+'|'+x['stage'] for x in comparison})!=len(current):raise ValueError('Incomplete paired review')
 for x in comparison:
  key=x['id']+'|'+x['stage'];r=current[key];b=refs[key]
  if x['newRowSha256']!=rowhash(r) or x['referenceRowSha256']!=rowhash(b) or x['sameBody']!=(r['response']['choices'][0]['message']['content']==b['response']['choices'][0]['message']['content']) or x['sameRequest']!=(r['request']==b['request']):raise ValueError('Wrong paired comparison')
 return rows,labels

def collect():
 rows,labels=checked();groups={}
 for arm,cohort in dict.fromkeys((r['arm'],r['cohort']) for r in rows if r['stage']=='format'):
  subset=[r for r in rows if r['stage']=='format' and r['arm']==arm and r['cohort']==cohort];js=[labels[(r['variant'],r['id'])] for r in subset]
  groups[arm+'/'+cohort]=dict(cases=len(subset),faithful=sum(j['faithful'] is True for j in js),unfaithful=sum(j['faithful'] is False for j in js),uncertain=sum(j['faithful'] is None for j in js),useful=sum(j['sceneUseful'] for j in js),repairSucceeded=sum(j.get('repairSucceeded') is True for j in js),harmfulEditCases=sum(bool(r['edits']) and j.get('editSupported') is False for r,j in zip(subset,js)),pipelineMs={str(p):sorted(r['pipelineMs'] for r in subset)[math.ceil(p*len(subset))-1] for p in [.5,.95,.99]})
 return dict(newGenerations=len(rows),sourceAttempts=489,groups=groups,metadataAnomaly=dict(candidateSelectorRecords=101,rawPreserved=True,releaseProvenanceQualified=False),selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits']),rows,labels

def main():
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();summary,rows,labels=collect();a.out.mkdir(parents=True,exist_ok=False);(a.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n');lines=['# Semantic selector experiment','','All609 actual responses reviewed;489 source attempts,120 intermediate choices. Known101 candidate template metadata defects are retained and bound to actual requests in the provenance sidecar. Development evidence, not release acceptance.','']
 for r in rows:lines += [f"## {r['arm']} / {r['id']} / {r['scene']} / {r['stage']}",'','```json',json.dumps({k:r[k] for k in ['source','context','prepared','editedSource','edits','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',labels[(r['variant'],r['id'])]['note'],'']
 (a.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
