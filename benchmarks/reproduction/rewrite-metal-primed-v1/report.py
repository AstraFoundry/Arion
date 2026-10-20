"""Check all measured and priming outputs with explicit order and canary limits."""
import pathlib,importlib.util,json,hashlib,math,argparse
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];BASE=ROOT/'benchmarks/runs/rewrite-metal-primed-v1';ARMS=['forward','reverse']
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
prefix=module('primed_prefix_evidence',HERE.parent/'rewrite-metal-prefix-v1/report.py');pipeline=module('primed_evidence_payload',HERE/'pipeline.py');prefix.HERE=HERE;prefix.pipeline=pipeline
read=prefix.read;sha=prefix.sha;rowhash=prefix.rowhash;quantiles=prefix.quantiles

def validate_warmups(rows,m,review,study,manifest,directory):
 samples=read(HERE/'warmups.json');f=read(HERE/'freeze.json')
 if len(rows)!=16 or len(review['reviews'])!=16 or study['status']!='complete' or study['order']!=directory.name or study['measuredGenerations']!=174 or study['warmupGenerations']!=16 or study['actualGenerations']!=190:raise ValueError('Wrong actual priming scope')
 if study['warmupResultsSha256']!=sha(directory/'warmups.jsonl') or review['resultsSha256']!=study['warmupResultsSha256'] or study['measuredResultsSha256']!=sha(directory/'results.jsonl') or study['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed startup/raw evidence')
 js={(j['variant'],j['id']):j for j in review['reviews']}
 if len(js)!=16 or len(study['workers'])!=4:raise ValueError('Duplicate priming review')
 for r,s in zip(rows,samples):
  j=js[(r['variant'],r['id'])]
  if any(r[k]!=v for k,v in s.items()) or r['order']!=directory.name or r['role']!='warmup' or r['modelSha256']!=f['model']['sha256'] or j['rowSha256']!=rowhash(r):raise ValueError('Changed synthetic priming input/review')
  observed=r['cleanupObservation'];cleanup=r['cleanup']
  if observed['id']!=r['id'] or observed['source']!=r['source'] or observed['cleanup']!=cleanup or cleanup['text']!=r['source'] or cleanup.get('spans') or cleanup['version']!='speech-spans-v2':raise ValueError('Changed actual Go priming cleanup')
  payload,prepared,_=pipeline.selection_payload(s,cleanup)
  if prepared!=r['prepared']:raise ValueError('Changed warmup producer')
  if r['stage']=='selection':
   edited,edits=pipeline.candidates.apply(r['response']['choices'][0]['message']['content'],prepared)
   if r['actualContextUsed']!=s['context'] or r['editedSource']!=edited or r['edits']!=edits or r['selection']!=json.loads(r['response']['choices'][0]['message']['content']):raise ValueError('Changed actual warmup selection')
  else:
   payload,encoding=pipeline.format_payload(s,cleanup['text'],prepared,{},[])
   if r['actualContextUsed']!={} or r['encoding']!=encoding or r['renderedBody']!=r['response']['choices'][0]['message']['content']:raise ValueError('Changed actual priming body')
  if payload!=r['request'] or r['renderError'] is not None:raise ValueError('Changed priming payload/result')
 for worker,actual in zip(study['workers'],manifest['workers']):
  stage_sum=sum(r['generationMs'] for r in rows if r['workerScene']==worker['scene'])
  if worker['scene']!=actual['scene'] or worker['warmupResponses']!=4 or worker['primingMs']<stage_sum or not math.isclose(worker['readyMs'],worker['modelStartupMs']+worker['primingMs']) or actual['startupMs']!=worker['readyMs']:raise ValueError('Invalid disclosed priming/readiness timing')
 return js

def checked(directory):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');j=read(directory/'review.json');d=read(directory/'selection.json')
 if any(x!=sha(directory/'results.jsonl') for x in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed measured/review bytes')
 labels=prefix.validate(rows,m,j,d,directory)
 expected=[s['id'] for scene in ['work','email','prompt','chat'] for s in (read(HERE/'inputs.json') if directory.name=='forward' else list(reversed(read(HERE/'inputs.json')))) if s['scene']==scene]
 if [r['id'] for r in rows if r['stage']=='format']!=expected:raise ValueError('Wrong actual source order')
 warm=[json.loads(x) for x in (directory/'warmups.jsonl').read_text().splitlines()];wj=read(directory/'warmup-review.json');study=read(directory/'study.json');warm_labels=validate_warmups(warm,m,wj,study,m,directory)
 return rows+warm,{**labels,**warm_labels}

def collect():
 result=dict(newGenerations=380,measuredGenerations=348,warmupGenerations=32,groups={},productionActivated=False,limits=read(HERE/'protocol.json')['limits']);all_rows=[];all_labels={};actual={}
 markers=[s['canary'] for s in read(HERE/'warmups.json') if s['canary']]
 for arm in ARMS:
  rows,js=checked(BASE/arm);measured=[r for r in rows if r.get('role')!='warmup'];warm=[r for r in rows if r.get('role')=='warmup'];groups={};actual[arm]={(r['stage'],r['id']):r for r in measured}
  for kind in ['development','natural']:
   subset=[r for r in measured if r['stage']=='format' and r['evaluationSet']==kind];labels=[js[(r['variant'],r['id'])] for r in subset]
   groups[kind]=dict(cases=len(subset),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),useful=sum(j['sceneUseful'] for j in labels),repairs=sum(j['repairSucceeded'] is True for j in labels),languagePreserved=sum(j['languagePreserved'] for j in labels),pipelineMs=quantiles([r['pipelineMs'] for r in subset]))
  finals=[r for r in measured if r['stage']=='format'];study=read(BASE/arm/'study.json');control,_=prefix.baseline.checked();ctrl={(r['stage'],r['id']):r for r in control}
  result['groups'][arm]=dict(quality=groups,cacheHits=sum(r['response']['timings']['cache_n']>0 for r in measured),positiveCanaries=sum(bool(r.get('canary')) and r['canary'] in r['response']['choices'][0]['message']['content'] for r in warm),negativeCanaryLeaks=[r['id'] for r in finals if any(c in r['renderedBody'] for c in markers)],uncachedRawParity=sum(r['response']['choices'][0]['message']['content']==ctrl[(r['stage'],r['id'])]['response']['choices'][0]['message']['content'] for r in measured),startup=study['workers'],maxSampledWorkerRssBytes=max(r['residentRssBytes'] for r in rows))
  all_rows.extend([dict(r,studyArm=arm) for r in rows]);all_labels.update({(arm,*key):j for key,j in js.items()})
 result['crossOrderRawParity']=sum(actual['forward'][key]['response']['choices'][0]['message']['content']==actual['reverse'][key]['response']['choices'][0]['message']['content'] for key in actual['forward'])
 return result,all_rows,all_labels

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Explicit priming, order and canary review','','All380 actual model responses personally reviewed:348 measured and32 startup warmups. Priming costs are separately recorded and included in readiness, not free. Positive synthetic markers and their absence in measured bodies are limited controls, not proof of complete private-context/native lifecycle isolation.','']
 for r in rows:lines += [f"## {r['studyArm']} / {r.get('role','measured')} / {r['id']} / {r['stage']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','cleanupObservation','actualContextUsed','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',js[(r['studyArm'],r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
