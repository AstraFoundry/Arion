"""Replay complete multi-arm source coverage, actual cleanup and unfiltered model stages."""
import pathlib,importlib.util,json,hashlib,argparse,math
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-tone-chain-v1/run'
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
pipeline=module('tone_chain_replay_payload',HERE/'pipeline.py')
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def validate(rows,m,review,outcomes,decision,directory=RUN):
 f=read(HERE/'freeze.json');history_path='benchmarks/corpora/rewrite-history-v1/inputs.json';control_path='benchmarks/corpora/rewrite-tone-controls-v1/inputs.json';history=read(ROOT/history_path);controls=read(ROOT/control_path);samples={('candidate',s['id']):(s,history_path) for s in history};samples.update({(arm,s['id']):(s,control_path) for arm in ['baseline','candidate'] for s in controls})
 for path,h in f['files'].items():
  if sha(ROOT/path)!=h:raise ValueError('Changed frozen tone-chain source')
 if m['status']!='complete' or m['generations']!=len(rows) or m['formatterGenerations']!=449 or m['attemptedSources']!=449 or m['stageProtocolFailures']!=0 or m['newAsrGenerations']!=0 or m['newOcrGenerations']!=0 or len(outcomes)!=449 or len(review['reviews'])!=len(rows):raise ValueError('Incomplete actual chain source/body/review scope')
 if m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles'] or m['cleanupWorkerSha256']!=f['cleanupWorkerSha256'] or m['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Wrong model/native/cleanup identity')
 expected_workers={('baseline',s['scene']) for s in controls}|{('candidate',s['scene']) for s in history+controls}
 if {(w['arm'],w['scene']) for w in m['workers']}!=expected_workers or len(m['workers'])!=len(expected_workers):raise ValueError('Wrong native scene/arm workers')
 for w in m['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed native runtime treatment')
  if 'offloaded 25/25 layers to GPU' not in (directory/(w['name']+'-server.log')).read_text():raise ValueError('Missing actual GPU placement')
 js={(j['variant'],j['id']):j for j in review['reviews']};selected={};finals={}
 if len(js)!=len(rows):raise ValueError('Duplicate reviews')
 for r in rows:
  key=(r['arm'],r['id']);sample,path=samples[key];j=js[(r['variant'],r['id'])]
  if any(r[k]!=v for k,v in sample.items()) or r['sourceControl']!=dict(path=path,rowSha256=rowhash(sample)) or j['rowSha256']!=rowhash(r):raise ValueError('Changed model input/review identity')
  c=r['cleanup'];obs=r['cleanupObservation'];data=r['source'].encode();cursor=0;pieces=[]
  if obs['id']!=r['id'] or obs['source']!=r['source'] or obs['cleanup']!=c or c['version']!='speech-spans-v2':raise ValueError('Wrong actual Go cleanup')
  for span in c.get('spans') or []:
   if not cursor<=span['start']<span['end']<=len(data):raise ValueError('Invalid cleanup span')
   pieces.append(data[cursor:span['start']]);cursor=span['end']
  pieces.append(data[cursor:])
  if b''.join(pieces).decode().strip()!=c['text']:raise ValueError('Wrong cleanup replay')
  request,prepared=pipeline.selection_payload(r['arm'],sample,c)
  if r['prepared']!=prepared or r['modelSha256']!=f['model']['sha256']:raise ValueError('Changed candidate producer/model')
  raw=r['response']['choices'][0]['message']['content']
  if r['stage']=='selection':
   if key in selected or not prepared['hypotheses'] or r['request']!=request or r['actualContextUsed']!=sample['context'] or r['templateSha256']!=sha(HERE.parent/'rewrite-bounded-edit-v2/prompts.json'):raise ValueError('Wrong actual selector call')
   edited,edits=pipeline.candidates_for(r['arm']).apply(raw,prepared)
   if r['selection']!=json.loads(raw) or r['editedSource']!=edited or r['edits']!=edits:raise ValueError('Wrong selected edits')
   selected[key]=r
  elif r['stage']=='format':
   called=bool(prepared['hypotheses']);previous=selected[key] if called else None;edited=previous['editedSource'] if called else c['text'];choice=previous['selection'] if called else {};edits=previous['edits'] if called else [];request,encoding=pipeline.format_payload(sample,edited,prepared,choice,edits)
   if key in finals or r['request']!=request or r['encoding']!=encoding or r['actualContextUsed']!={} or r['editedSource']!=edited or r['edits']!=edits or r['sourcePreparation']['modelSelectorCalled'] is not called or r['selectionRowSha256']!=(rowhash(previous) if called else None):raise ValueError('Wrong current conditional chain/isolated formatter')
   if r['renderedBody']!=raw or r['templateSha256']!=sha(HERE.parent/'rewrite-work-scope-v4/prompts.json') or r['pipelineMs']<r['generationMs']:raise ValueError('Changed unfiltered body/timing')
   if j['faithful'] is not True and j['faithful'] is not False and j['faithful'] is not None:raise ValueError('Wrong meaning label')
   if j['faithful'] is not True and j['sceneUseful']:raise ValueError('Unknown/error counted useful')
   finals[key]=r
  else:raise ValueError('Wrong stage')
  if r['renderError'] is not None or r['request']['cache_prompt'] is not True:raise ValueError('Wrong stage protocol/cache outcome')
 if set(finals)!=set(samples) or len(selected)!=m['selectorGenerations'] or len(selected)+449!=len(rows):raise ValueError('Missing actual coverage')
 for o in outcomes:
  key=(o['arm'],o['id']);called=key in selected
  if o['status']!='generated' or o['sourceRowSha256']!=rowhash(samples[key][0]) or o['modelSelectorCalled'] is not called or o['finalRowSha256']!=rowhash(finals[key]) or o['selectionRowSha256']!=(rowhash(selected[key]) if called else None):raise ValueError('Changed source outcome')
 if len({(o['arm'],o['id']) for o in outcomes})!=449 or decision['qualified'] or decision['selected'] is not None or decision['productionActivated'] or decision['freshExecuted']:raise ValueError('Unsupported outcome/release claim')
 return js

def checked(directory=RUN):
 rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()));m=read(directory/'manifest.json');j=read(directory/'review.json');o=list(map(json.loads,(directory/'case-outcomes.jsonl').read_text().splitlines()));d=read(directory/'selection.json')
 if any(h!=sha(directory/'results.jsonl') for h in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or m['outcomesSha256']!=sha(directory/'case-outcomes.jsonl') or d['reviewSha256']!=sha(directory/'review.json') or d['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed completed evidence bytes')
 amendment=read(directory/'review-amendment.json')
 if amendment['initialReviewSha256']!=sha(directory/'review-initial.json') or amendment['currentReviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed review correction provenance')
 labels=validate(rows,m,j,o,d,directory)
 parity=read(directory/'reference-parity.json');refs={}
 for path in {x['referencePath'] for x in parity['records']}:
  for row in map(json.loads,(ROOT/path).read_text().splitlines()):refs[(path,row['variant'],row['id'])]=row
 current={r['id']:r for r in rows if r['stage']=='format' and r['arm']=='candidate' and r['id'].startswith('history-')}
 if len(parity['records'])!=401 or len({x['id'] for x in parity['records']})!=401:raise ValueError('Incomplete historical output comparison')
 for x in parity['records']:
  a=current[x['id']];b=refs[(x['referencePath'],x['referenceVariant'],x['id'])]
  if x['newRowSha256']!=rowhash(a) or x['referenceRowSha256']!=rowhash(b) or any(a[k]!=b[k] for k in ['source','context','cleanup','editedSource','edits','request','renderedBody']):raise ValueError('Changed historical parity evidence')
 recovery=read(directory/'reference-recovery-observation.json');r=current[recovery['id']]
 if recovery['rowSha256']!=rowhash(r) or recovery['publishedReferenceSha256']!=sha(ROOT/recovery['publishedReferencePath']) or recovery['primaryAsrSourceFaithful'] is not None:raise ValueError('Changed separate reference-restoration evidence')
 return rows,labels
def collect():
 rows,js=checked();groups={}
 for arm,kind in dict.fromkeys((r['arm'],r['cohort']) for r in rows if r['stage']=='format'):
  subset=[r for r in rows if r['stage']=='format' and r['arm']==arm and r['cohort']==kind];labels=[js[(r['variant'],r['id'])] for r in subset];choices=[r for r in rows if r['stage']=='selection' and r['arm']==arm and r['cohort']==kind]
  groups[arm+'/'+kind]=dict(cases=len(subset),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),useful=sum(j['sceneUseful'] for j in labels),languagePreserved=sum(j['languagePreserved'] for j in labels),selectorCalls=len(choices),choiceConforms=sum(js[(r['variant'],r['id'])]['choiceConforms'] for r in choices),appliedEditCases=sum(bool(r['edits']) for r in subset),supportedEditCases=sum(bool(r['edits']) and j.get('editSupported') is True for r,j in zip(subset,labels)),harmfulEditCases=sum(bool(r['edits']) and j.get('editSupported') is False for r,j in zip(subset,labels)),reviewedRepairOpportunities=sum(j.get('repairOpportunity') is True for j in labels) if kind=='new-synthetic-tone-controls' else None,repairSucceeded=sum(j.get('repairSucceeded') is True for j in labels) if kind=='new-synthetic-tone-controls' else None,pipelineMs={str(p):sorted(r['pipelineMs'] for r in subset)[math.ceil(p*len(subset))-1] for p in [.5,.95,.99]})
 return dict(newGenerations=len(rows),sourceAttempts=449,groups=groups,failedBodies=[dict(arm=r['arm'],id=r['id']) for r in rows if r['stage']=='format' and js[(r['variant'],r['id'])]['faithful'] is False],selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits']),rows,js

def main():
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();s,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(s,ensure_ascii=False,indent=2)+'\n');lines=['# Tone-insensitive spelling complete chain review','','Every actual selector and final body personally inspected. Complete449 source attempts are not independent subjects; candidate401 historical sources and paired24 synthetic controls are separate. Current actual Go cleanup and conditional selection/formatting performed; no semantic output rejection or source fallback.','']
 for r in rows:lines += [f"## {r['arm']} / {r['id']} / {r['stage']} / {r['cohort']}",'','```json',json.dumps({k:r[k] for k in ['source','context','cleanupObservation','prepared','actualContextUsed','editedSource','edits','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(s,ensure_ascii=False))
if __name__=='__main__':main()
