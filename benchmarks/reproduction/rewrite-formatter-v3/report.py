"""Replay a formatter intervention with explicit actual-selector reuse identities."""
import pathlib,json,hashlib,importlib.util,argparse,math
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-formatter-v3/run'
def load(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
prior=load('formatter_v3_actual_v2_evidence',HERE.parent/'rewrite-bounded-edit-v2/report.py');pipeline=load('formatter_v3_replay_pipeline',HERE/'pipeline.py')
read=prior.read;sha=prior.sha;rowhash=prior.rowhash

def validate(rows,manifest,review,decision):
 freeze=read(HERE/'freeze.json');protocol=read(HERE/'protocol.json')
 for name,digest in freeze['files'].items():
  if sha(ROOT/name)!=digest:raise ValueError('Changed frozen formatter input: '+name)
 if manifest['status']!='complete' or manifest['generations']!=36 or manifest['newGenerations']!=36 or manifest['reusedSelections']!=36 or manifest['sourceCases']!=36:raise ValueError('Incomplete actual formatter observations')
 if manifest['freezeSha256']!=sha(HERE/'freeze.json') or manifest['model']!=freeze['model'] or manifest['runtimeFiles']!=freeze['runtimeFiles']:raise ValueError('Changed model/runtime/freeze')
 reference=ROOT/protocol['selectionReference']
 if sha(reference)!=protocol['selectionReferenceSha256'] or manifest['selectionReferenceSha256']!=sha(reference) or manifest['selectionReference']!=protocol['selectionReference']:raise ValueError('Changed actual selector reference')
 previous,_=prior.checked();selected={r['id']:r for r in previous if r['stage']=='selection'}
 samples=read(HERE.parent/'rewrite-bounded-edit-v2/development.json');expected=[s for scene in ['work','email','prompt','chat'] for s in samples if s['scene']==scene]
 if len(rows)!=36 or len(review['reviews'])!=36:raise ValueError('Incomplete review coverage')
 judgments={(j['variant'],j['id']):j for j in review['reviews']}
 if len(judgments)!=36:raise ValueError('Duplicate review')
 if len(manifest['workers'])!=4:raise ValueError('Missing actual workers')
 for w in manifest['workers']:
  for flag,value in [('--threads','4'),('--n-gpu-layers','0'),('--parallel','1'),('--ctx-size','4096')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed native execution')
 for r,s in zip(rows,expected):
  if any(r[k]!=v for k,v in s.items()) or r['stage']!='format' or r['variant']!='formatter-v3' or r['effectiveModelScene']!=s['scene']:raise ValueError('Changed source/stage/scene')
  actual=selected[s['id']];j=judgments[(r['variant'],r['id'])]
  if r['selectionRowSha256']!=rowhash(actual) or r['selectionReference']!=protocol['selectionReference']:raise ValueError('Changed reused native decision')
  if j['rowSha256']!=rowhash(r):raise ValueError('Changed reviewed output')
  request,encoding=pipeline.format_payload(s,actual['editedSource'],actual['prepared'],actual['selection'],actual['edits'])
  if r['request']!=request or r['encoding']!=encoding or r['actualContextUsed']!={} or any(r[k]!=actual[k] for k in ['cleanup','editedSource','edits']):raise ValueError('Changed actual formatting request/source')
  if r['templateSha256']!=sha(HERE/'prompts.json') or r['modelSha256']!=freeze['model']['sha256']:raise ValueError('Changed model/prompt identity')
  raw=r['response']['choices'][0]['message']['content']
  if r['renderedBody']!=pipeline.candidates.restore(raw,encoding) or r['renderError'] is not None or r['response']['timings']['cache_n']!=0 or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Changed unfiltered/native response')
 eligible=all(j['faithful'] for j in judgments.values()) and sum(j['sceneUseful'] for j in judgments.values())>=35 and sum(j['repairSucceeded'] is True for j in judgments.values())==11 and all(judgments[('formatter-v3',i)]['sceneUseful'] for i in ['owner-anchor','email-anchor']) and all(r['response']['choices'][0]['finish_reason']=='stop' for r in rows)
 if decision['freezeSha256']!=sha(HERE/'freeze.json') or decision['qualified']!=eligible or decision['selected']!=('formatter-v3' if eligible else None) or decision['freshExecuted'] or decision['productionActivated']:raise ValueError('Decision mismatches frozen gate/scope')
 return judgments

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');review=read(directory/'review.json');decision=read(directory/'selection.json')
 if any(v!=sha(directory/'results.jsonl') for v in [m['resultsSha256'],review['resultsSha256'],decision['resultsSha256']]) or decision['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed result/review hash')
 return rows,validate(rows,m,review,decision)
def collect():
 rows,js=checked();previous,pj=prior.checked();baseline=[r for r in previous if r['stage']=='format'];refs={r['id']:r for r in baseline}
 def summary(group,j):
  labels=[j[(r['variant'],r['id'])] for r in group]
  return dict(observations=len(group),faithful=sum(l['faithful'] for l in labels),sceneReady=sum(l['sceneUseful'] for l in labels),repairs=sum(l['repairSucceeded'] is True for l in labels),faithfulRepairMessages=sum(l['repairSucceeded'] is True and l['faithful'] for l in labels),truncated=sum(r['response']['choices'][0]['finish_reason']=='length' for r in group))
 return dict(newGenerations=36,reusedActualSelections=36,reusedBaselineBodies=36,final=summary(rows,js),baseline=summary(baseline,pj),identicalEditedSources=sum(r['editedSource']==refs[r['id']]['editedSource'] for r in rows),performance=dict(formatOnlyMs={str(p):sorted(r['generationMs'] for r in rows)[math.ceil(p*36)-1] for p in [.5,.95,.99]},scope='Formatter-only requests; old actual selector rows reused. No continuous full-pipeline or concurrent ASR/native/min16GB timing claim.'),selection=read(RUN/'selection.json'),productionActivated=False),rows,js

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Formatter-only v3 complete review','','All36 new raw bodies personally reviewed. Actual v2 selector results are referenced, not fabricated or counted again. Source/model/runtime/budget/routing unchanged; prompt recipe differs. No output rejection/source fallback.','']
 for r in rows:lines += [f"## {r['id']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','actualContextUsed','selectionReference','selectionRowSha256','editedSource','edits','request']},ensure_ascii=False,indent=2),'```','','**Complete native body (unfiltered)**','','```text',r['renderedBody'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
