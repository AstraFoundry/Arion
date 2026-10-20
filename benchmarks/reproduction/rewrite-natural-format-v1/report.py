"""Replay natural-source fidelity, source limits and uncertain edits separately."""
import pathlib,importlib.util,json,hashlib,argparse,math
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-natural-format-v1/run'
spec=importlib.util.spec_from_file_location('natural_replay_pipeline',HERE.parent/'rewrite-formatter-v3/pipeline.py');pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def validate(rows,m,review,decision,directory=RUN):
 f=read(HERE/'freeze.json');p=read(HERE/'protocol.json');inputs=read(ROOT/p['inputs']);cleanup={r['id']:r for r in read(ROOT/p['cleanup'])}
 for name,h in f['files'].items():
  if sha(ROOT/name)!=h:raise ValueError('Changed natural source/prompt/producer')
 if m['status']!='complete' or m['generations']!=120 or m['sourceCases']!=120 or m['newAsrGenerations']!=0 or m['newSelectorGenerations']!=0 or len(rows)!=120 or len(review['reviews'])!=120:raise ValueError('Incomplete or fabricated natural inference')
 if m['freezeSha256']!=sha(HERE/'freeze.json') or m['model']!=f['model'] or m['runtimeFiles']!=f['runtimeFiles']:raise ValueError('Changed natural runtime/model')
 js={(j['variant'],j['id']):j for j in review['reviews']}
 if len(js)!=120 or [r['id'] for r in rows]!=[s['id'] for s in inputs]:raise ValueError('Wrong natural review/order')
 if len(m['workers'])!=1:raise ValueError('Wrong worker coverage')
 command=m['workers'][0]['command']
 for flag,value in [('--n-gpu-layers','99'),('--threads','4'),('--ctx-size','4096'),('--parallel','1'),('-lv','5')]:
  if command[command.index(flag)+1]!=value:raise ValueError('Wrong native treatment')
 if 'offloaded 25/25 layers to GPU' not in (directory/'metal-chat-server.log').read_text():raise ValueError('Missing actual placement')
 for r,s in zip(rows,inputs):
  c=cleanup[s['id']];j=js[(r['variant'],r['id'])]
  if any(r[k]!=v for k,v in s.items()) or c['source']!=s['source'] or r['cleanup']!=c['cleanup'] or r['editedSource']!=c['cleanup']['text']:raise ValueError('Changed natural source/cleanup')
  prepared=pipeline.candidates.prepare(c['cleanup']['text'],s['context'],s['scene']);request,e=pipeline.format_payload(s,c['cleanup']['text'],prepared,{},[])
  if prepared['hypotheses'] or r['sourcePreparation']['modelSelectorCalled'] is not False or r['sourcePreparation']['hypotheses']!=[]:raise ValueError('Invented no-context selector')
  if r['request']!=request or r['encoding']!=e or r['actualContextUsed']!={} or r['edits']!=[] or r['stage']!='format' or r['effectiveModelScene']!='chat':raise ValueError('Changed natural prompt/context')
  if r['templateSha256']!=sha(HERE.parent/'rewrite-formatter-v3/prompts.json') or r['modelSha256']!=f['model']['sha256'] or j['rowSha256']!=rowhash(r):raise ValueError('Changed reviewed identity')
  if r['renderedBody']!=r['response']['choices'][0]['message']['content'] or r['renderError'] is not None or r['response']['timings']['cache_n']!=0 or r['response']['choices'][0]['finish_reason'] not in ['stop','length']:raise ValueError('Changed unfiltered natural result')
  if j['faithful'] not in [True,False,None] or (j['sceneUseful'] and j['faithful'] is not True):raise ValueError('Unverified meaning counted as usable')
 if decision['qualified'] or decision['selected'] is not None or decision['productionActivated'] or decision['freshExecuted'] or decision['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Diagnostic promoted to release')
 return js

def checked(directory=RUN):
 rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];m=read(directory/'manifest.json');review=read(directory/'review.json');d=read(directory/'selection.json')
 if any(x!=sha(directory/'results.jsonl') for x in [m['resultsSha256'],review['resultsSha256'],d['resultsSha256']]) or d['reviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed reviewed natural bytes')
 return rows,validate(rows,m,review,d,directory)
def collect():
 rows,js=checked()
 def summary(group):
  labels=[js[(r['variant'],r['id'])] for r in group]
  return dict(observations=len(group),faithful=sum(j['faithful'] is True for j in labels),unfaithful=sum(j['faithful'] is False for j in labels),uncertain=sum(j['faithful'] is None for j in labels),sceneReady=sum(j['sceneUseful'] for j in labels),sourceLimited=sum(j['sourceLimited'] for j in labels),languagePreserved=sum(j['languagePreserved'] for j in labels),truncated=sum(r['response']['choices'][0]['finish_reason']=='length' for r in group))
 times=sorted(r['generationMs'] for r in rows)
 return dict(newGenerations=120,summary=summary(rows),languages={lang:summary([r for r in rows if r['language']==lang]) for lang in ['zh','en']},formatterMs={str(p):times[math.ceil(p*120)-1] for p in [.5,.95,.99]},maxSampledWorkerRssBytes=max(r['residentRssBytes'] for r in rows),newAsrGenerations=0,newSelectorGenerations=0,productionActivated=False,scope=read(HERE/'protocol.json')),rows,js

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();result,rows,js=collect();out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);(out/'summary.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');lines=['# Natural source-only formatting diagnostic','','All120 complete source/output pairs personally reviewed. Source fidelity, language, usability and uncertainty are separate. No independent/audio rehearing/hidden-reference recovery or release claim. All copied corrupted/partial sources remain in denominators.','']
 for r in rows:
  j=js[(r['variant'],r['id'])];lines += [f"## {r['id']}",'',f"Language: {r['language']}; scene: {r['scene']} → {r['effectiveModelScene']}",'','**Actual source**','','```text',r['source'],'```','','**Actual full messages**','','```json',json.dumps(r['request']['messages'],ensure_ascii=False,indent=2),'```','','**Unfiltered native body**','','```text',r['renderedBody'],'```','',j['note'],'',f"Faithful: {j['faithful']}; usable: {j['sceneUseful']}; language retained: {j['languagePreserved']}; source limited: {j['sourceLimited']}",'']
 (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps({k:v for k,v in result.items() if k!='scope'},ensure_ascii=False))
if __name__=='__main__':main()
