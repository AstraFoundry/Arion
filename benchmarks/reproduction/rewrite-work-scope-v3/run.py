"""Prepare all historical work cases and paired new source-only controls."""
import pathlib,sys,json,hashlib
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_formatter_jobs as executor
import pipeline
ROOT=executor.engine.ROOT;HIST=ROOT/'benchmarks/runs/rewrite-history-format-v2/run/results.jsonl';CHAIN=ROOT/'benchmarks/runs/rewrite-history-chain-v1/run/results.jsonl';CONTROLS=ROOT/'benchmarks/corpora/rewrite-work-scope-v3/inputs.json'
def rowhash(r):return hashlib.sha256(json.dumps(r,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def jobs():
 refs=[json.loads(x) for x in HIST.read_text().splitlines()];choices={r['id']:r['selection'] for r in map(json.loads,CHAIN.read_text().splitlines()) if r['stage']=='selection'};controls=json.loads(CONTROLS.read_text());result=[]
 for arm in ['baseline','candidate']:
  samples=([r for r in refs if r['scene']=='work'] if arm=='candidate' else [])+controls
  for s in samples:
   historical=s['cohort']=='historical-synthetic-regression'
   if historical:
    seed=dict(s);source=s['editedSource'];prepared=s['prepared'];choice=choices.get(s['id'],{});edits=s['edits'];control=dict(path=str(HIST.relative_to(ROOT)),rowSha256=rowhash(s));prep=dict(kind='reused-actual-historical-cleanup-and-selection',modelSelectorCalled=False,priorModelSelectorCalled=s['sourcePreparation']['priorModelSelectorCalled'])
   else:
    source=s['source'];prepared=dict(source=source,context={},scene='work',hypotheses=[],scope='Source-only synthetic formatter control; no candidate producer or selector executed.');choice={};edits=[];control=dict(path=str(CONTROLS.relative_to(ROOT)),rowSha256=rowhash(s));prep=dict(kind='synthetic-source-only-format',modelSelectorCalled=False,priorModelSelectorCalled=False);seed=dict(s,evaluationSet=s['cohort'],cleanup=dict(text=source,version='not-executed-source-only-control'),cleanupObservation=None,prepared=prepared,editedSource=source,edits=[],selectionRowSha256=None)
   request,encoding=pipeline.format_payload(arm,s,source,prepared,choice,edits);template=HERE/'prompts.json' if arm=='candidate' else HERE.parent/'rewrite-history-format-v2/prompts.json'
   seed.update(variant=arm,pipeline='work-scope-v3',stage='format',effectiveModelScene='work',templateVersion=('work-scope-v3' if arm=='candidate' else 'history-format-v2')+'-work',templateSha256=executor.engine.sha(template),modelSha256=json.loads((HERE/'freeze.json').read_text())['model']['sha256'],request=request,encoding=encoding,sourceControl=control,sourcePreparation=prep,actualContextUsed={},pipelineMs=None,cleanupIpcMs=None,scope='Only formatter newly generated. Historical cleanup/selection reused; controls have no cleanup, no OCR or selection. No output rejection or source fallback.')
   for k in ['response','renderedBody','renderError','generationMs','residentRssBytes']:seed.pop(k,None)
   result.append(seed)
 return result
if __name__=='__main__':executor.execute(HERE,jobs(),dict(pipeline='work-scope-v3',sourceCases=110,historicalCases=94,newControls=16,newSelectorGenerations=0,newAsrGenerations=0,scope=json.loads((HERE/'protocol.json').read_text())['limits']))
