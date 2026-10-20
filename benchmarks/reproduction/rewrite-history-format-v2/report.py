"""Validate paired formatter isolation and retain all personally reviewed bodies."""
import pathlib,json,hashlib,importlib.util,argparse,math
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-history-format-v2/run';BASE=ROOT/'benchmarks/runs/rewrite-history-chain-v1/run'
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
pipeline=module('history_v2_payload_evidence',HERE/'pipeline.py');baseline=module('history_v1_reference_evidence',HERE.parent/'rewrite-history-chain-v1/report.py')
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def validate(rows,manifest,review,decision,directory=RUN):
 original,_=baseline.checked();refs={r['id']:r for r in original if r['stage']=='format'};choices={r['id']:r['selection'] for r in original if r['stage']=='selection'};f=read(HERE/'freeze.json')
 for path,h in f['files'].items():
  if sha(ROOT/path)!=h:raise ValueError('Changed frozen formatter study')
 if len(rows)!=401 or len({r['id'] for r in rows})!=401 or manifest['status']!='complete' or manifest['generations']!=401 or manifest['sourceCases']!=401 or manifest['newSelectorGenerations']!=0 or manifest['reusedActualSelections']!=64 or manifest['newAsrGenerations']!=0 or len(review['reviews'])!=401:raise ValueError('Changed actual formatter scope')
 if manifest['freezeSha256']!=sha(HERE/'freeze.json') or manifest['model']!=f['model'] or manifest['runtimeFiles']!=f['runtimeFiles']:raise ValueError('Wrong native identity')
 if len(manifest['workers'])!=4:raise ValueError('Missing actual scene worker')
 for w in manifest['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Wrong runtime setting')
  if 'offloaded 25/25 layers to GPU' not in (directory/('format-v2-'+w['scene']+'-server.log')).read_text():raise ValueError('No observed Metal placement')
 js={(j['variant'],j['id']):j for j in review['reviews']}
 if len(js)!=401:raise ValueError('Duplicate reviewed body')
 for r in rows:
  ref=refs[r['id']];j=js[(r['variant'],r['id'])];payload,encoding=pipeline.format_payload(ref,ref['editedSource'],ref['prepared'],choices.get(ref['id'],{}),ref['edits'])
  if r['sourceControl']!=dict(path=str((BASE/'results.jsonl').relative_to(ROOT)),rowSha256=rowhash(ref)) or j['rowSha256']!=rowhash(r):raise ValueError('Wrong source/review binding')
  for k in ['source','context','scene','cohort','editedSource','edits','prepared','cleanup','cleanupObservation','actualContextUsed','selectionRowSha256']:
   if r[k]!=ref[k]:raise ValueError('Changed fixed prior stage: '+k)
  if r['sourcePreparation']!=dict(kind='reused-actual-historical-cleanup-and-selection',modelSelectorCalled=False,priorModelSelectorCalled=ref['sourcePreparation']['modelSelectorCalled']) or r['cleanupIpcMs'] is not None or r['pipelineMs'] is not None:raise ValueError('Reused timing mistaken for new full chain')
  if r['request']!=payload or r['encoding']!=encoding or r['templateSha256']!=sha(HERE/'prompts.json') or r['modelSha256']!=f['model']['sha256'] or r['stage']!='format':raise ValueError('Wrong actual prompt/model')
  if r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['renderError'] is not None or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Filtered or missing body')
  if j['faithful'] not in [True,False,None] or j['faithful'] is not True and j['sceneUseful']:raise ValueError('Uncertain/error body counted useful')
 if decision['qualified'] or decision['selected'] is not None or decision['productionActivated'] or decision['freshExecuted']:raise ValueError('Unsupported release claim')
 return js

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');j=read(directory/'review.json');d=read(directory/'selection.json')
 if any(h!=sha(directory/'results.jsonl') for h in [m['resultsSha256'],j['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json') or d['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed result/review bytes')
 return rows,validate(rows,m,j,d,directory)
def collect():
 rows,js=checked();old,oldjs=baseline.checked();refs={r['id']:r for r in old if r['stage']=='format'};groups={}
 for kind in ['historical-synthetic-regression','historical-natural-read-speech']:
  subset=[r for r in rows if r['cohort']==kind];labels=[js[(r['variant'],r['id'])] for r in subset];before=[oldjs[(refs[r['id']]['variant'],r['id'])] for r in subset]
  groups[kind]=dict(cases=len(subset),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),useful=sum(j['sceneUseful'] for j in labels),baselineFaithful=sum(j['faithful'] is True for j in before),baselineUseful=sum(j['sceneUseful'] for j in before),rawParity=sum(r['renderedBody']==refs[r['id']]['renderedBody'] for r in subset),languagePreserved=sum(j['languagePreserved'] for j in labels))
 transitions=[]
 for r in rows:
  j=js[(r['variant'],r['id'])];ref=refs[r['id']];b=oldjs[(ref['variant'],r['id'])]
  if (j['faithful'],j['sceneUseful'])!=(b['faithful'],b['sceneUseful']):transitions.append(dict(id=r['id'],beforeFaithful=b['faithful'],afterFaithful=j['faithful'],beforeUseful=b['sceneUseful'],afterUseful=j['sceneUseful'],note=j['note']))
 return dict(newGenerations=401,newSelectorGenerations=0,reusedActualChoices=64,groups=groups,transitions=transitions,selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits']),rows,js

def main():
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();s,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(s,ensure_ascii=False,indent=2)+'\n');lines=['# Full401-source formatter prompt comparison','','Only401 formatting calls newly generated;64 actual prior spelling choices and Go cleanup retained by original row identity. All bodies personally reviewed without validator rejection. Historical exposed synthetic281 and natural120 are separate.','']
 for r in rows:lines += [f"## {r['id']} / {r['scene']} / {r['cohort']}",'','```json',json.dumps({k:r[k] for k in ['source','context','editedSource','edits','sourcePreparation','request']},ensure_ascii=False,indent=2),'```','','```text',r['renderedBody'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(s,ensure_ascii=False))
if __name__=='__main__':main()
