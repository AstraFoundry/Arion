"""Verify paired actual requests and observed provider placement for all36 bodies."""
import pathlib,importlib.util,json,hashlib,argparse,math
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-metal-regression-v1/run'
spec=importlib.util.spec_from_file_location('metal_full_cpu_evidence',HERE.parent/'rewrite-formatter-v3/report.py');prior=importlib.util.module_from_spec(spec);spec.loader.exec_module(prior)
read=prior.read;sha=prior.sha;rowhash=prior.rowhash

def validate(rows,m,review,decision,directory=RUN):
 f=read(HERE/'freeze.json');p=read(HERE/'protocol.json');cpu,_=prior.checked();lookup={r['id']:r for r in cpu}
 for name,h in f['files'].items():
  if sha(ROOT/name)!=h:raise ValueError('Changed frozen provider input')
 if m['status']!='complete' or m['generations']!=36 or m['sourceCases']!=36 or len(rows)!=36 or len(review['reviews'])!=36 or m['freezeSha256']!=sha(HERE/'freeze.json') or m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles']:raise ValueError('Wrong provider coverage/identity')
 if sha(ROOT/p['reference'])!=p['referenceSha256']:raise ValueError('Changed CPU reference')
 js={(j['variant'],j['id']):j for j in review['reviews']}
 if len(js)!=36 or [r['id'] for r in rows]!=[r['id'] for r in cpu] or len(m['workers'])!=4:raise ValueError('Wrong pair/review/worker coverage')
 for w in m['workers']:
  for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1'),('-lv','5')]:
   if w['command'][w['command'].index(flag)+1]!=value:raise ValueError('Changed provider parameters')
  if 'offloaded 25/25 layers to GPU' not in (directory/('metal-'+w['scene']+'-server.log')).read_text():raise ValueError('Missing actual process placement')
 for r in rows:
  c=lookup[r['id']];j=js[(r['variant'],r['id'])]
  if r['sourceControl']['rowSha256']!=rowhash(c) or r['request']!=c['request'] or any(r[k]!=c[k] for k in ['source','context','scene','language','editedSource','edits','cleanup','encoding','modelSha256','templateSha256']):raise ValueError('Changed identical-request treatment')
  if j['rowSha256']!=rowhash(r) or r['stage']!='format' or r['effectiveModelScene']!=r['scene'] or r['actualContextUsed']!={}:raise ValueError('Changed review/context/stage')
  if r['response']['usage']['prompt_tokens']!=c['response']['usage']['prompt_tokens'] or r['response']['timings']['cache_n']!=0 or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Changed token/template/cache observation')
  if r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['renderError'] is not None:raise ValueError('Changed unfiltered body')
 eligible=all(j['faithful'] for j in js.values()) and sum(j['sceneUseful'] for j in js.values())>=35 and sum(j['repairSucceeded'] is True for j in js.values())==11 and all(js[('metal-full36',i)]['sceneUseful'] for i in ['owner-anchor','email-anchor']) and all(r['response']['choices'][0]['finish_reason']=='stop' for r in rows)
 if decision['qualified']!=eligible or decision['selected']!=('metal-regression-v1' if eligible else None) or decision['productionActivated'] or decision['freshExecuted'] or decision['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed full gate decision')
 return js

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');review=read(directory/'review.json');d=read(directory/'selection.json')
 if any(v!=sha(directory/'results.jsonl') for v in [m['resultsSha256'],review['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed paired evidence bytes')
 return rows,validate(rows,m,review,d,directory)
def collect():
 rows,js=checked();cpu,_=prior.checked();lookup={r['id']:r for r in cpu};times=sorted(r['generationMs'] for r in rows)
 result=dict(newGenerations=36,reusedActualSelections=36,faithful=sum(j['faithful'] for j in js.values()),sceneReady=sum(j['sceneUseful'] for j in js.values()),repairs=sum(j['repairSucceeded'] is True for j in js.values()),identicalRequests=sum(r['request']==lookup[r['id']]['request'] for r in rows),identicalBodies=sum(r['renderedBody']==lookup[r['id']]['renderedBody'] for r in rows),formatterMs={str(p):times[math.ceil(p*36)-1] for p in [.5,.95,.99]},maxSampledWorkerRssBytes=max(r['residentRssBytes'] for r in rows),selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits'])
 return result,rows,js

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Full36 Metal formatter provider review','','All36 new raw bodies personally inspected; actual CPU requests/native selector results reused, not counted again. Same prompts/budget; provider placement logged upfront.','']
 for r in rows:lines += [f"## {r['id']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','actualContextUsed','editedSource','edits','sourceControl','request']},ensure_ascii=False,indent=2),'```','','```text',r['renderedBody'],'```','',js[(r['variant'],r['id'])]['note'],'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
