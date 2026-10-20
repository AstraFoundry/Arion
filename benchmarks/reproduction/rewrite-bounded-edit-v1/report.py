"""Replay both actual model stages without conflating plans and delivered bodies."""
import argparse, hashlib, json, pathlib, sys
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
sys.path.insert(0,str(HERE))
import pipeline
RUN=ROOT/'benchmarks/runs/rewrite-bounded-edit-v1/run'
def read(path):return json.loads(path.read_text())
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def rowhash(row):return hashlib.sha256(json.dumps(row,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def validate(rows,manifest,review,decision):
 if manifest['status']!='complete' or manifest['generations']!=72 or manifest['sourceCases']!=36:raise ValueError('Incomplete run')
 if manifest['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed freeze')
 frozen=read(HERE/'freeze.json')
 for name,digest in frozen['files'].items():
  if sha(ROOT/name)!=digest:raise ValueError('Changed frozen producer: '+name)
 if manifest['model']!=frozen['model'] or manifest['runtimeFiles']!=frozen['runtimeFiles']:raise ValueError('Changed runtime/model')
 if len(manifest['workers'])!=4:raise ValueError('Missing actual workers')
 for w in manifest['workers']:
  for flag,value in [('--threads','4'),('--n-gpu-layers','0'),('--parallel','1'),('--ctx-size','4096')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed native execution')
 samples=read(HERE/'development.json');expected=[(s,stage) for scene in ['work','email','prompt','chat'] for s in samples if s['scene']==scene for stage in ['selection','format']]
 if len(rows)!=72 or len(review['reviews'])!=72:raise ValueError('Incomplete observations/review')
 judgments={(r['variant'],r['id']):r for r in review['reviews']}
 if len(judgments)!=72:raise ValueError('Duplicate reviews')
 previous=None
 for r,(s,stage) in zip(rows,expected):
  if any(r[k]!=v for k,v in s.items()) or r['stage']!=stage:raise ValueError('Changed source/order/stage')
  j=judgments[(r['variant'],r['id'])]
  if j['rowSha256']!=rowhash(r) or j['stage']!=stage:raise ValueError('Changed reviewed row')
  if r['modelSha256']!=frozen['model']['sha256'] or r['templateSha256']!=sha(HERE/'prompts.json'):raise ValueError('Changed model/template identity')
  raw=r['response']['choices'][0]['message']['content']
  timing=r['response']['timings']
  if timing['cache_n']!=0 or r['response']['choices'][0]['finish_reason']!='stop':raise ValueError('Cached/truncated observation')
  if stage=='selection':
   request,prepared,cleanup=pipeline.selection_payload(s)
   edited,edits=pipeline.candidates.apply(raw,prepared)
   if r['prepared']!=prepared or r['cleanup']!=cleanup or r['selection']!=json.loads(raw) or r['actualContextUsed']!=s['context'] or r['renderedBody'] is not None:raise ValueError('Changed selection authority')
   if r['effectiveModelScene']!='spelling-selection' or j['sceneUseful'] is not None:raise ValueError('Selection mislabeled as delivered body')
  else:
   if r['selectionRowSha256']!=rowhash(previous):raise ValueError('Changed stage handoff')
   edited,edits=previous['editedSource'],previous['edits']
   request,encoding=pipeline.format_payload(s,edited,previous['prepared'],previous['selection'],edits)
   if r['encoding']!=encoding or r['actualContextUsed']!={} or r['cleanup']!=previous['cleanup']:raise ValueError('Changed formatter context/protection')
   if r['renderedBody']!=pipeline.candidates.restore(raw,encoding) or r['effectiveModelScene']!=s['scene']:raise ValueError('Changed unfiltered body/scene')
  if r['request']!=request or r['editedSource']!=edited or r['edits']!=edits or r['renderError'] is not None:raise ValueError('Changed actual payload/edit')
  previous=r
 if decision['freezeSha256']!=sha(HERE/'freeze.json') or decision['qualified'] or decision['selected'] is not None or decision['freshExecuted'] or decision['productionActivated']:raise ValueError('Unqualified research promoted')
 return judgments

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];manifest=read(directory/'manifest.json');review=read(directory/'review.json');decision=read(directory/'selection.json')
 for value in [manifest['resultsSha256'],review['resultsSha256'],decision['resultsSha256']]:
  if value!=sha(directory/'results.jsonl'):raise ValueError('Changed results')
 if decision['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed decision review')
 return rows,validate(rows,manifest,review,decision)
def collect():
 rows,judgments=checked();final=[r for r in rows if r['stage']=='format'];select=[r for r in rows if r['stage']=='selection']
 def summary(group):
  js=[judgments[(r['variant'],r['id'])] for r in group]
  return dict(observations=len(group),faithful=sum(j['faithful'] for j in js),sceneReady=sum(j['sceneUseful'] is True for j in js),repairs=sum(j['repairSucceeded'] is True for j in js),requiredRepairs=sum(j['repairSucceeded'] is not None for j in js))
 return dict(newGenerations=72,sourceCases=36,intermediate=summary(select),final=summary(final),selection=read(RUN/'selection.json'),scope=read(HERE/'protocol.json'),productionActivated=False),rows,judgments

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();result,rows,judgments=collect();out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
 (out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
 lines=['# Bounded spelling and isolated formatting review','','All72 actual model observations inspected. Selection JSON is intermediate evidence; only36 final bodies are delivered rewrites. No output rejection or source fallback. New user spelling policy permits raw-source retention as well as same-letter normalization.','']
 for r in rows:
  j=judgments[(r['variant'],r['id'])];lines += [f"## {r['stage']} / {r['id']}",'',f"Fixture scene: {r['scene']}; actual model task: {r['effectiveModelScene']}",'','```json',json.dumps({k:r.get(k) for k in ['source','context','actualContextUsed','cleanup','prepared','selection','editedSource','edits','encoding','request']},ensure_ascii=False,indent=2),'```','','**Raw native output**','','```text',r['response']['choices'][0]['message']['content'],'```','','**Unfiltered final body**','','```text',r['renderedBody'] if r['stage']=='format' else '[Intermediate decision only; no delivery body]','```','',j['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps({k:v for k,v in result.items() if k!='scope'},ensure_ascii=False))
if __name__=='__main__':main()
