"""Replay full historical coverage, actual stage evidence and complete manual review."""
import pathlib,importlib.util,json,hashlib,math,argparse
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-history-chain-v1/run';SOURCE=ROOT/'benchmarks/corpora/rewrite-history-v1/inputs.json'
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
pipeline=module('history_evidence_payload',HERE/'pipeline.py');source_audit=module('history_original_rows',HERE.parent/'rewrite-history-source-v1/verify.py');enumeration=module('history_enumeration',HERE.parent/'rewrite-history-source-v1/verify-enumeration.py')
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def quantiles(values):
 values=sorted(values);return {str(p):values[math.ceil(p*len(values))-1] for p in [.5,.95,.99]}
def validate(rows,m,review,outcomes,decision,directory=RUN):
 source_audit.checked();enumeration.checked();f=read(HERE/'freeze.json');samples={s['id']:s for s in read(SOURCE)}
 for path,digest in f['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen full-history source')
 if m['status']!='complete' or m['generations']!=465 or len(rows)!=465 or len(review['reviews'])!=465 or m['attemptedSources']!=401 or m['sourceCases']!=401 or len(outcomes)!=401 or m['selectorGenerations']!=64 or m['formatterGenerations']!=401 or m['stageProtocolFailures']!=0 or m['newAsrGenerations']!=0:raise ValueError('Wrong actual coverage')
 if m['freezeSha256']!=sha(HERE/'freeze.json') or m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles'] or m['cleanupWorkerSha256']!=f['cleanupWorkerSha256']:raise ValueError('Changed actual native identity')
 if len(m['workers'])!=4:raise ValueError('Wrong scene worker count')
 for w in m['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1'),('-lv','5')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed native provider treatment')
  if 'offloaded 25/25 layers to GPU' not in (directory/('history-'+w['scene']+'-server.log')).read_text():raise ValueError('Missing actual GPU placement')
 js={(j['variant'],j['id']):j for j in review['reviews']};selected={};finals={}
 if len(js)!=465:raise ValueError('Duplicate review')
 for r in rows:
  sample=samples[r['id']];j=js[(r['variant'],r['id'])]
  if any(r[k]!=v for k,v in sample.items()) or r['evaluationSet']!=sample['cohort'] or r['sourceControl']!=dict(path=str(SOURCE.relative_to(ROOT)),rowSha256=rowhash(sample)) or j['rowSha256']!=rowhash(r):raise ValueError('Changed source/review binding')
  observed=r['cleanupObservation'];cleanup=r['cleanup']
  if observed['id']!=r['id'] or observed['source']!=r['source'] or observed['cleanup']!=cleanup or cleanup['version']!='speech-spans-v2':raise ValueError('Changed actual Go cleanup')
  data=r['source'].encode();cursor=0;parts=[]
  for span in cleanup.get('spans') or []:
   if not cursor<=span['start']<span['end']<=len(data):raise ValueError('Invalid UTF-8 deletion spans')
   parts.append(data[cursor:span['start']]);cursor=span['end']
  parts.append(data[cursor:])
  if b''.join(parts).decode().strip()!=cleanup['text']:raise ValueError('Changed cleanup replay')
  payload,prepared,_=pipeline.selection_payload(sample,cleanup)
  if prepared!=r['prepared'] or r['modelSha256']!=f['model']['sha256']:raise ValueError('Changed producer/model')
  if r['stage']=='selection':
   if r['id'] in selected or not prepared['hypotheses'] or r['actualContextUsed']!=sample['context'] or r['request']!=payload or r['templateSha256']!=sha(HERE.parent/'rewrite-bounded-edit-v2/prompts.json'):raise ValueError('Changed actual selection call')
   edited,edits=pipeline.candidates.apply(r['response']['choices'][0]['message']['content'],prepared)
   if r['selection']!=json.loads(r['response']['choices'][0]['message']['content']) or r['editedSource']!=edited or r['edits']!=edits:raise ValueError('Changed actual applied edits')
   selected[r['id']]=r
  elif r['stage']=='format':
   called=bool(prepared['hypotheses']);choice=selected[r['id']]['selection'] if called else {};edited=selected[r['id']]['editedSource'] if called else cleanup['text'];edits=selected[r['id']]['edits'] if called else [];payload,encoding=pipeline.format_payload(sample,edited,prepared,choice,edits)
   if r['id'] in finals or r['actualContextUsed']!={} or r['request']!=payload or r['encoding']!=encoding or r['editedSource']!=edited or r['edits']!=edits or r['selectionRowSha256']!=(rowhash(selected[r['id']]) if called else None) or r['sourcePreparation']['modelSelectorCalled'] is not called:raise ValueError('Changed conditional source-only formatter')
   if r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['templateSha256']!=sha(HERE.parent/'rewrite-language-profile-v2/prompts.json') or not math.isfinite(r['pipelineMs']) or r['pipelineMs']<r['generationMs']:raise ValueError('Changed unfiltered body/timing')
   if j['faithful'] is not True and j['faithful'] is not False and j['faithful'] is not None:raise ValueError('Invalid meaning label')
   if j['sceneUseful'] and j['faithful'] is not True:raise ValueError('Unknown meaning counted as useful')
   finals[r['id']]=r
  else:raise ValueError('Unexpected native stage')
  if r['renderError'] is not None or r['request']['cache_prompt'] is not True or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Wrong raw outcome/cache treatment')
 if set(finals)!=set(samples) or len(selected)!=64:raise ValueError('Missing source coverage')
 for outcome in outcomes:
  sid=outcome['id'];called=sid in selected
  if outcome['status']!='generated' or outcome['sourceRowSha256']!=rowhash(samples[sid]) or outcome['modelSelectorCalled'] is not called or outcome['finalRowSha256']!=rowhash(finals[sid]) or outcome['selectionRowSha256']!=(rowhash(selected[sid]) if called else None):raise ValueError('Changed per-source outcome')
 if len({o['id'] for o in outcomes})!=401 or decision['qualified'] or decision['selected'] is not None or decision['freshExecuted'] or decision['productionActivated'] or decision['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Wrong completion/release boundary')
 return js

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');j=read(directory/'review.json');o=[json.loads(x) for x in (directory/'case-outcomes.jsonl').read_text().splitlines()];d=read(directory/'selection.json')
 amendment=read(directory/'review-amendment.json')
 if amendment['initialReviewSha256']!=sha(directory/'review-v1.json') or amendment['currentReviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed second-review provenance')
 if any(x!=sha(directory/'results.jsonl') for x in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or m['outcomesSha256']!=sha(directory/'case-outcomes.jsonl') or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed original result/review bytes')
 return rows,validate(rows,m,j,o,d,directory)
def collect():
 rows,js=checked();groups={}
 for kind in ['historical-synthetic-regression','historical-natural-read-speech']:
  subset=[r for r in rows if r['stage']=='format' and r['cohort']==kind];labels=[js[(r['variant'],r['id'])] for r in subset]
  groups[kind]=dict(cases=len(subset),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),useful=sum(j['sceneUseful'] for j in labels),languagePreserved=sum(j['languagePreserved'] for j in labels),supportedEdits=sum(bool(r['edits']) for r in subset),missedReviewedNormalization=sum(j.get('repairOpportunity') is True and j.get('repairSucceeded') is False for j in labels),pipelineMs=quantiles([r['pipelineMs'] for r in subset]))
 result=dict(newGenerations=465,sourceCases=401,finalBodies=401,actualSelectorCalls=64,choiceConforms=sum(js[(r['variant'],r['id'])]['choiceConforms'] for r in rows if r['stage']=='selection'),groups=groups,failedFinalIds=[r['id'] for r in rows if r['stage']=='format' and js[(r['variant'],r['id'])]['faithful'] is False],uncertainFinalIds=[r['id'] for r in rows if r['stage']=='format' and js[(r['variant'],r['id'])]['faithful'] is None],selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits'])
 return result,rows,js

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Complete401-tuple historical chain review','','All465 actual raw responses personally reviewed:64 spelling choices and401 final bodies. Synthetic281 and historical natural120 remain separate. Normalization/meaning/usable organization/uncertainty are distinct; no overlap or validator proxy establishes success. Two retained-criterion failures include one explicit-emphasis omission with correct final value; cancellation framing and incomplete natural predicate stay uncertain. Original v1 judgment and second-review provenance retained. No source fallback or hidden rejected outputs.','']
 for r in rows:lines += [f"## {r['cohort']} / {r['id']} / {r['stage']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','cleanupObservation','prepared','actualContextUsed','editedSource','edits','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
