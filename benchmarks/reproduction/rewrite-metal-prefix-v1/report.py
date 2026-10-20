"""Bind cache-enabled actual responses to complete uncached chain controls."""
import pathlib,importlib.util,json,hashlib,math,argparse,copy
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-metal-prefix-v1/run'
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
pipeline=module('prefix_report_payload',HERE/'pipeline.py');baseline=module('prefix_uncached_evidence',HERE.parent/'rewrite-metal-chain-v1/report.py')
read=baseline.read;sha=baseline.sha;rowhash=baseline.rowhash;quantiles=baseline.quantiles

def validate(rows,m,review,decision,directory=RUN):
 old,_=baseline.checked();controls={(r['stage'],r['id']):r for r in old};f=read(HERE/'freeze.json');samples={s['id']:s for s in read(HERE/'inputs.json')}
 pairing=read(directory/'paired-controls.json')
 if sha(ROOT/pairing['baselineArchivePath'])!=pairing['baselineArchiveSha256'] or sha(ROOT/pairing['baselineResultsPath'])!=pairing['baselineResultsSha256'] or sha(directory/'results.jsonl')!=pairing['newResultsSha256']:raise ValueError('Changed paired baseline archive')
 pairs={(p['stage'],p['id']):p for p in pairing['pairs']}
 if len(pairs)!=174:raise ValueError('Incomplete paired reference audit')
 for path,digest in f['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen prefix recipe')
 if m['status']!='complete' or m['generations']!=174 or len(rows)!=174 or len(review['reviews'])!=174 or m['newSelectorGenerations']!=18 or m['formatterGenerations']!=156 or m['sourceCases']!=156 or m['newAsrGenerations']!=0:raise ValueError('Incomplete prefix response coverage')
 if m['freezeSha256']!=sha(HERE/'freeze.json') or m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles'] or m['cleanupWorkerSha256']!=f['cleanupWorkerSha256']:raise ValueError('Changed actual model/runtime/cleanup')
 if len(m['workers'])!=4:raise ValueError('Wrong cache worker count')
 old_workers={w['scene']:w for w in read(baseline.RUN/'manifest.json')['workers']}
 for w in m['workers']:
  if w['command']!=old_workers[w['scene']]['command'] or 'offloaded 25/25 layers to GPU' not in (directory/('chain-'+w['scene']+'-server.log')).read_text():raise ValueError('Changed native command/placement')
 js={(j['variant'],j['id']):j for j in review['reviews']};selected={};seen=set();parity=0
 if len(js)!=174:raise ValueError('Duplicate review')
 for r in rows:
  key=(r['stage'],r['id']);c=controls[key];sample=samples[r['id']];j=js[(r['variant'],r['id'])]
  if pairs[key]['baselineRowSha256']!=rowhash(c) or pairs[key]['newRowSha256']!=rowhash(r):raise ValueError('Changed paired raw rows')
  if key in seen:raise ValueError('Duplicate actual response')
  seen.add(key)
  for field in ['source','context','scene','language','cohort','stage','variant','modelSha256','sourceControl','cleanup','prepared','effectiveModelScene','templateVersion','templateSha256','evaluationSet']:
   if r[field]!=c[field]:raise ValueError('Changed fixed source/producer/template: '+field)
  observed=r['cleanupObservation']
  if observed['id']!=r['id'] or observed['source']!=r['source'] or observed['cleanup']!=r['cleanup'] or j['rowSha256']!=rowhash(r):raise ValueError('Changed actual Go/review binding')
  request,prepared,_=pipeline.selection_payload(sample,r['cleanup'])
  if r['stage']=='selection':
   if r['request']!=request or not prepared['hypotheses'] or r['actualContextUsed']!=sample['context']:raise ValueError('Changed selector payload/context')
   raw=r['response']['choices'][0]['message']['content'];edited,edits=pipeline.candidates.apply(raw,prepared)
   if r['selection']!=json.loads(raw) or r['editedSource']!=edited or r['edits']!=edits or r['renderError'] is not None:raise ValueError('Changed actual selector result')
   selected[r['id']]=r
  else:
   called=bool(prepared['hypotheses']);choice=selected[r['id']]['selection'] if called else {};edited=selected[r['id']]['editedSource'] if called else r['cleanup']['text'];edits=selected[r['id']]['edits'] if called else []
   request,encoding=pipeline.format_payload(sample,edited,prepared,choice,edits)
   if r['request']!=request or r['encoding']!=encoding or r['editedSource']!=edited or r['edits']!=edits or r['actualContextUsed']!={} or r['sourcePreparation']['modelSelectorCalled'] is not called or r['selectionRowSha256']!=(rowhash(selected[r['id']]) if called else None):raise ValueError('Changed actual conditional formatter')
   if r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['renderError'] is not None or not math.isfinite(r['pipelineMs']) or r['pipelineMs']<r['generationMs']:raise ValueError('Changed unfiltered body/chain timer')
   if j['faithful'] is not True and j['faithful'] is not False and j['faithful'] is not None:raise ValueError('Invalid meaning label')
   if j['sceneUseful'] and j['faithful'] is not True:raise ValueError('Unknown meaning counted as useful')
  timing=r['response']['timings']
  if not isinstance(timing['cache_n'],int) or timing['cache_n']<0 or timing['cache_n']>r['response']['usage']['prompt_tokens'] or r['request']['cache_prompt'] is not True or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Invalid actual cache/token provenance')
  if r['response']['choices'][0]['message']['content']==c['response']['choices'][0]['message']['content']:parity+=1
 if seen!=set(controls) or len(selected)!=18:raise ValueError('Missing paired stage/control')
 dev=[r for r in rows if r['stage']=='format' and r['evaluationSet']=='development'];dj=[js[(r['variant'],r['id'])] for r in dev];eligible=all(j['faithful'] is True for j in dj) and sum(j['sceneUseful'] for j in dj)>=35 and sum(j['repairSucceeded'] is True for j in dj)==11 and all(js[(r['variant'],r['id'])]['sceneUseful'] for r in dev if r['id'] in ['owner-anchor','email-anchor']) and all(r['response']['choices'][0]['finish_reason']=='stop' for r in dev)
 if decision['developmentQualified']!=eligible or decision['parityQualified']!=(parity==174) or decision['qualified'] or decision['selected'] is not None or decision['freshExecuted'] or decision['productionActivated'] or decision['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Wrong cache/quality/release boundary')
 return js

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');j=read(directory/'review.json');d=read(directory/'selection.json')
 if any(x!=sha(directory/'results.jsonl') for x in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed raw/reviewed response bytes')
 return rows,validate(rows,m,j,d,directory)
def collect():
 rows,js=checked();old,_=baseline.checked();controls={(r['stage'],r['id']):r for r in old};groups={}
 for kind in ['development','natural']:
  subset=[r for r in rows if r['stage']=='format' and r['evaluationSet']==kind];labels=[js[(r['variant'],r['id'])] for r in subset]
  groups[kind]=dict(observations=len(subset),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),sceneReady=sum(j['sceneUseful'] for j in labels),repairs=sum(j['repairSucceeded'] is True for j in labels),languagePreserved=sum(j['languagePreserved'] for j in labels),pipelineMs=quantiles([r['pipelineMs'] for r in subset]),formatterMs=quantiles([r['generationMs'] for r in subset]))
 matched=0;identical=0;changes=[]
 for r in rows:
  c=controls[(r['stage'],r['id'])];request=copy.deepcopy(r['request']);request['cache_prompt']=False;matched+=request==c['request'];same=r['response']['choices'][0]['message']['content']==c['response']['choices'][0]['message']['content'];identical+=same
  if not same:changes.append(dict(id=r['id'],stage=r['stage'],baselineRowSha256=rowhash(c),newRowSha256=rowhash(r)))
 result=dict(newGenerations=174,finalBodies=156,actualSelectorCalls=18,normalizedMatchedRequests=matched,identicalRawResponses=identical,changes=changes,cacheHits=sum(r['response']['timings']['cache_n']>0 for r in rows),cachedTokens=sum(r['response']['timings']['cache_n'] for r in rows),groups=groups,maxSampledWorkerRssBytes=max(r['residentRssBytes'] for r in rows),selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits'])
 return result,rows,js

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Actual full-chain native prefix reuse review','','All174 actual outputs personally reviewed against original full source, OCR, Go cleanup and cache-off controls. Nonzero cache_n is observed reuse, not quality or privacy proof. Same source156, not independent subjects; timing baseline is historical serial, no isolated causal speedup claim.','']
 for r in rows:lines += [f"## {r['evaluationSet']} / {r['id']} / {r['stage']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','cleanupObservation','sourcePreparation','actualContextUsed','editedSource','edits','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
