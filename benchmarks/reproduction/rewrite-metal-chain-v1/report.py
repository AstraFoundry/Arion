"""Check actual continuous stage evidence and replay every reviewed model output."""
import pathlib,sys,importlib.util,json,hashlib,argparse,math
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-metal-chain-v1/run'
spec=importlib.util.spec_from_file_location('chain_report_pipeline',HERE/'pipeline.py');pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def validate(rows,m,review,decision,directory=RUN):
 f=read(HERE/'freeze.json');inputs=read(HERE/'inputs.json');samples={s['id']:s for s in inputs};cleanups={r['id']:r['cleanup'] for r in read(ROOT/'benchmarks/runs/rewrite-cleanup-copula-v1/observations.json')};dev={s['id'] for s in read(HERE.parent/'rewrite-bounded-edit-v2/development.json')}
 for path,h in f['files'].items():
  if sha(ROOT/path)!=h:raise ValueError('Changed frozen chain source')
 if m['status']!='complete' or m['generations']!=174 or m['sourceCases']!=156 or m['newSelectorGenerations']!=18 or m['formatterGenerations']!=156 or m['newAsrGenerations']!=0 or len(rows)!=174 or len(review['reviews'])!=174:raise ValueError('Incomplete actual chain coverage')
 if m['freezeSha256']!=sha(HERE/'freeze.json') or m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles'] or m['cleanupWorkerSha256']!=f['cleanupWorkerSha256']:raise ValueError('Changed native chain identity')
 if len(m['workers'])!=4:raise ValueError('Wrong native workers')
 for w in m['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1'),('-lv','5')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed native settings')
  if 'offloaded 25/25 layers to GPU' not in (directory/('chain-'+w['scene']+'-server.log')).read_text():raise ValueError('Missing actual Metal placement')
 js={(j['variant'],j['id']):j for j in review['reviews']};selection_rows={};final_ids=[]
 if len(js)!=174:raise ValueError('Duplicate reviews')
 for r in rows:
  sample=samples[r['id']];j=js[(r['variant'],r['id'])];kind='development' if r['id'] in dev else 'natural'
  if any(r[k]!=v for k,v in sample.items()) or r['evaluationSet']!=kind or r['sourceControl']!=dict(path='benchmarks/reproduction/rewrite-metal-chain-v1/inputs.json',rowSha256=rowhash(sample)):raise ValueError('Changed source identity')
  if r['cleanup']!=cleanups[r['id']] or r['cleanupObservation']['source']!=r['source'] or r['cleanupObservation']['id']!=r['id'] or r['cleanupObservation']['cleanup']!=r['cleanup']:raise ValueError('Changed actual Go cleanup')
  request,prepared,_=pipeline.selection_payload(sample,r['cleanup'])
  if r['prepared']!=prepared or r['modelSha256']!=f['model']['sha256'] or j['rowSha256']!=rowhash(r):raise ValueError('Changed producer/review')
  if r['stage']=='selection':
   if r['id'] in selection_rows or not prepared['hypotheses'] or r['request']!=request or r['actualContextUsed']!=sample['context'] or r['renderError'] is not None:raise ValueError('Invented/changed selector call')
   edited,edits=pipeline.candidates.apply(r['response']['choices'][0]['message']['content'],prepared)
   if r['selection']!=json.loads(r['response']['choices'][0]['message']['content']) or r['editedSource']!=edited or r['edits']!=edits:raise ValueError('Changed actual spelling edits')
   if r['templateSha256']!=sha(HERE.parent/'rewrite-bounded-edit-v2/prompts.json'):raise ValueError('Changed selector prompt')
   selection_rows[r['id']]=r
  elif r['stage']=='format':
   final_ids.append(r['id']);called=bool(prepared['hypotheses']);selected=selection_rows.get(r['id']);choices=selected['selection'] if called else {};edited=selected['editedSource'] if called else r['cleanup']['text'];edits=selected['edits'] if called else []
   if r['sourcePreparation']['modelSelectorCalled'] is not called or r['selectionRowSha256']!=(rowhash(selected) if called else None) or r['editedSource']!=edited or r['edits']!=edits or r['actualContextUsed']!={}:raise ValueError('Changed conditional full chain')
   payload,encoding=pipeline.format_payload(sample,edited,prepared,choices,edits)
   if r['request']!=payload or r['encoding']!=encoding or r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['renderError'] is not None or r['templateSha256']!=sha(HERE.parent/'rewrite-language-profile-v2/prompts.json'):raise ValueError('Changed unfiltered format response')
   if not math.isfinite(r['pipelineMs']) or r['pipelineMs']<r['generationMs'] or r['pipelineMs']<r['cleanupIpcMs']:raise ValueError('Invalid continuous timing')
   if j['faithful'] is not True and j['faithful'] is not False and j['faithful'] is not None:raise ValueError('Wrong meaning label')
   if j['sceneUseful'] and j['faithful'] is not True:raise ValueError('Unknown meaning counted as useful')
  else:raise ValueError('Unknown actual stage')
  if r['response']['timings']['cache_n']!=0 or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Changed output/cache provenance')
 if len(final_ids)!=156 or set(final_ids)!=set(samples) or len(selection_rows)!=18:raise ValueError('Wrong final/source coverage')
 dev_rows=[r for r in rows if r['stage']=='format' and r['evaluationSet']=='development'];dj=[js[(r['variant'],r['id'])] for r in dev_rows];eligible=all(j['faithful'] is True for j in dj) and sum(j['sceneUseful'] for j in dj)>=35 and sum(j['repairSucceeded'] is True for j in dj)==11 and all(js[(r['variant'],r['id'])]['sceneUseful'] for r in dev_rows if r['id'] in ['owner-anchor','email-anchor']) and all(r['response']['choices'][0]['finish_reason']=='stop' for r in dev_rows)
 if decision['developmentQualified']!=eligible or decision['qualified'] or decision['selected'] is not None or decision['productionActivated'] or decision['freshExecuted'] or decision['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Wrong release/development boundary')
 return js

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');j=read(directory/'review.json');d=read(directory/'selection.json')
 if any(x!=sha(directory/'results.jsonl') for x in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed raw/review bytes')
 return rows,validate(rows,m,j,d,directory)
def quantiles(values):
 values=sorted(values);return {str(p):values[math.ceil(p*len(values))-1] for p in [.5,.95,.99]}
def collect():
 rows,js=checked();finals=[r for r in rows if r['stage']=='format'];groups={}
 for kind in ['development','natural']:
  subset=[r for r in finals if r['evaluationSet']==kind];labels=[js[(r['variant'],r['id'])] for r in subset]
  groups[kind]=dict(observations=len(subset),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),sceneReady=sum(j['sceneUseful'] for j in labels),repairs=sum(j['repairSucceeded'] is True for j in labels),languagePreserved=sum(j['languagePreserved'] for j in labels),pipelineMs=quantiles([r['pipelineMs'] for r in subset]),formatterMs=quantiles([r['generationMs'] for r in subset]))
 result=dict(newGenerations=len(rows),actualSelectorCalls=18,skippedSelectorCalls=138,finalBodies=156,groups=groups,selectorConforms=sum(js[(r['variant'],r['id'])]['choiceConforms'] for r in rows if r['stage']=='selection'),maxSampledWorkerRssBytes=max(r['residentRssBytes'] for r in rows),selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits'])
 return result,rows,js

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Actual continuous Metal chain review','','All174 actual model outputs personally reviewed:18 selections and156 final bodies. Zero-produced-hypotheses cases skip selection uniformly; no invented model outputs. Original full sources/OCR and actual Go cleanup remain separate. Continuous chain times exclude startup/ASR/native insertion and are not production stop-to-insertion acceptance.','']
 for r in rows:
  lines += [f"## {r['evaluationSet']} / {r['id']} / {r['stage']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','cleanupObservation','sourcePreparation','actualContextUsed','editedSource','edits','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
