"""Replay full-chain preview treatment and its exact frozen baseline projection."""
import pathlib,importlib.util,sys,json,argparse,statistics
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-selector-preview-v1/run';BASE=ROOT/'benchmarks/runs/rewrite-selector-semantic-v2/run'
def module(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
before=list(sys.path);sys.path.insert(0,str(HERE.parent))
try:evidence=module('preview_evidence_replay',HERE.parent/'gguf_chain_evidence_v3.py')
finally:sys.path[:]=before
pipeline=module('preview_report_pipeline',HERE/'pipeline.py');read=evidence.read;sha=evidence.sha;rowhash=evidence.rowhash
SOURCES=['benchmarks/corpora/rewrite-history-v1/inputs.json','benchmarks/corpora/rewrite-tone-controls-v1/inputs.json','benchmarks/corpora/rewrite-semantic-roles-v1/inputs.json']
def plans():return [('candidate',[dict(sample=s,sourcePath=p) for p in SOURCES for s in read(ROOT/p)])]
def development_qualified(rows,labels):
 baseline={(j['stage'],j['id']):j for j in read(BASE/'review.json')['reviews']}
 required={x['id'] for x in read(ROOT/'benchmarks/corpora/rewrite-semantic-roles-v1/evaluator.json') if x['condition']=='related' and x['intendedRecovery'] is not None}
 finals=[r for r in rows if r['stage']=='format']
 for r in finals:
  j=labels[(r['variant'],r['id'])];b=baseline[('format',r['id'])]
  if j['faithful'] is False or (b['sceneUseful'] and not j['sceneUseful']) or (b.get('repairSucceeded') is True and j.get('repairSucceeded') is not True):return False
  if r['id'] in required and j.get('repairSucceeded') is not True:return False
 return True

def validate_decision(rows,labels,d):
 if d['qualified'] or d['selected'] is not None or d['developmentPassed']!=development_qualified(rows,labels):raise ValueError('Unsupported development/release decision')

def validate(rows,m,j,o,d,directory=RUN):
 labels=evidence.validate(ROOT,HERE,pipeline,plans(),rows,m,j,o,d,directory);validate_decision(rows,labels,d);return labels
def strip_previews(request):
 request=json.loads(json.dumps(request,ensure_ascii=False))
 for msg in request['messages']:
  if msg['role']=='user':
   d=json.loads(msg['content'])
   for h in d['hypotheses']:h.pop('candidateRewrites')
   msg['content']=json.dumps(d,ensure_ascii=False,separators=(',',':'))
 return request

def supplementary(rows,labels):
 baseline={(r['stage'],r['id']):r for r in map(json.loads,(BASE/'results.jsonl').read_text().splitlines())};selectors=[r for r in rows if r['stage']=='selection'];changed=[r for r in selectors if r['selection']!=baseline[('selection',r['id'])]['selection']];direction=[(baseline[('selection',r['id'])]['selection'][k],v) for r in changed for k,v in r['selection'].items() if v!=baseline[('selection',r['id'])]['selection'][k]]
 return dict(selectorCalls=len(selectors),changedChoiceCases=len(changed),changedChoiceIds=len(direction),allChangedIdsKeepToReplace=all(a=='keep' and b=='replace' for a,b in direction),reviewedSupportedChanges=sum(labels[(r['variant'].replace('-selection','-format'),r['id'])]['faithful'] is True for r in changed),reviewedHarmfulChanges=sum(labels[(r['variant'].replace('-selection','-format'),r['id'])]['faithful'] is False for r in changed),medianAdditionalSelectorInputTokens=statistics.median(r['response']['usage']['prompt_tokens']-baseline[('selection',r['id'])]['response']['usage']['prompt_tokens'] for r in selectors),maximumSelectorInputTokens=max(r['response']['usage']['prompt_tokens'] for r in selectors),unchangedFormatterRequests=sum(r['request']==baseline[('format',r['id'])]['request'] for r in rows if r['stage']=='format'))

def checked(directory=RUN):
 rows,labels=evidence.checked(ROOT,HERE,pipeline,plans(),directory);baseline={(r['stage'],r['id']):r for r in map(json.loads,(BASE/'results.jsonl').read_text().splitlines())};current={(r['stage'],r['id']):r for r in rows};comparison=read(directory/'reference-comparison.json')['records']
 if set(current)!=set(baseline) or len(comparison)!=len(rows) or len({(x['stage'],x['id']) for x in comparison})!=len(rows):raise ValueError('Incomplete frozen paired comparison')
 for x in comparison:
  key=(x['stage'],x['id']);r=current[key];b=baseline[key]
  if x['rowSha256']!=rowhash(r) or x['referenceRowSha256']!=rowhash(b) or any(r[k]!=b[k] for k in ['source','context','cleanup','prepared']):raise ValueError('Changed paired source/producer')
  if x['sameBody']!=(r['response']['choices'][0]['message']['content']==b['response']['choices'][0]['message']['content']) or x['sameRequest']!=(r['request']==b['request']):raise ValueError('Wrong paired body/request observation')
  if r['stage']=='selection' and strip_previews(r['request'])!=b['request']:raise ValueError('Treatment changed more than candidate previews')
  if r['stage']=='format' and r['editedSource']==b['editedSource'] and r['request']!=b['request']:raise ValueError('Unchanged source changed formatter treatment')
 d=read(directory/'selection.json')
 validate_decision(rows,labels,d)
 observed=read(directory/'supplementary-observations.json')
 if observed['resultsSha256']!=sha(directory/'results.jsonl') or any(observed[k]!=v for k,v in supplementary(rows,labels).items()):raise ValueError('Changed supplementary paired observations')
 return rows,labels

def collect():
 rows,labels=checked();comparison=read(RUN/'reference-comparison.json')['records']
 return dict(newGenerations=len(rows),sourceAttempts=457,supplementary=supplementary(rows,labels),groups=evidence.summarize(rows,labels),changedSelectorDecisions=sum(not x['sameBody'] for x in comparison if x['stage']=='selection'),changedFinalBodies=sum(not x['sameBody'] for x in comparison if x['stage']=='format'),selection=read(RUN/'selection.json'),limits=read(HERE/'protocol.json')['limits']),rows,labels

def main():
 p=argparse.ArgumentParser();p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args();summary,rows,labels=collect();a.out.mkdir(parents=True,exist_ok=False);(a.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n');lines=['# Candidate sentence preview treatment','','All actual current outputs personally reviewed. Exact frozen baseline requests reconstruct after removing only deterministic preview fields from selector data. Baseline generations are not counted again. All final bodies are unfiltered.','']
 for r in rows:lines += [f"## {r['id']} / {r['scene']} / {r['stage']}",'','```json',json.dumps({k:r[k] for k in ['source','context','prepared','editedSource','edits','templateVersion','templatePath','templateSha256','actualMessagesSha256','request']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',labels[(r['variant'],r['id'])]['note'],'']
 (a.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
