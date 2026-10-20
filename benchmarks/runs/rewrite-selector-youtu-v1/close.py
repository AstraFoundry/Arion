"""Publish personally reviewed actual Youtu responses with exact control bindings."""
from pathlib import Path
import json,hashlib,shutil,importlib.util
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-youtu-v1';W=Path(__file__).resolve().parent;P=R/'benchmarks/runs/rewrite-selector-youtu-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rh(x):return hashlib.sha256(json.dumps(x,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def save(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
assert read(W/'terminal-status.json')['exitCode']==0 and read(W/'run/manifest.json')['status']=='complete'
rows=list(map(json.loads,(W/'run/results.jsonl').read_text().splitlines()));assert len(rows)==202
spec=importlib.util.spec_from_file_location('youtu_close_report',H/'report.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};targets={r['id']:r for r in read(R/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json')};reviews=[];pairs=[]
for r in rows:
 assert r['renderError'] is None and r['response']['choices'][0]['finish_reason']=='stop'
 s=m.assessment(r,targets);ref=refs[r['arm'],r['id']];assert r['request']==ref['request'] and r['prepared']==ref['prepared']
 note='Personally read all202 actual Youtu raw outputs and edited sources, all12 native responses,29 unique failed source/context/output tuples and58 changed canonical-control pairs. '
 note+=('Unsupported meaning substitution: '+', '.join(e['original']+' -> '+e['replacement'] for e in r['edits'])+'. ' if s['harmful'] else 'Context-supported required spelling repair missed. ' if s['missed'] else 'Frozen spelling/literal/ambiguity/meaning rubric respected. ')
 note+='Allowed readCash normalization and unchanged spelling both receive credit; unsupported readCache does not. No semantic output rejection/source fallback, final-body/style, independent or actual16GB acceptance. Custom-license research only; production license restrictions unresolved.'
 reviews.append(dict(id=r['id'],variant=r['variant'],stage='selection',rowSha256=rh(r),faithful=not s['harmful'],sceneUseful=None,languagePreserved=True,choiceConforms=s['conforms'],harmfulChoice=s['harmful'],missedRepair=s['missed'],issues=['unsupported-substitution'] if s['harmful'] else ['required-repair-missed'] if s['missed'] else [],note=note))
 pairs.append(dict(id=r['id'],arm=r['arm'],rowSha256=rh(r),reference=r['comparisonReference'],sameRequest=True,samePrepared=True,changedChoice=r['selection']!=ref['selection'],priorChoice=ref['selection'],newChoice=r['selection'],priorConforms=m.assessment(ref,targets)['conforms'],newConforms=s['conforms']))
save(W/'manual-review-completion.json',dict(complete=True,selectorResponses=202,nativeResponses=12,ranges=[[0,21],[21,42],[42,60],[60,85],[85,101],[101,120],[120,128],[128,165],[165,172],[172,180],[180,202]],uniqueFailedTuples=29,changedCanonicalPairs=58,reviewer='Assistant, not independent blind release acceptance'))
shutil.copytree(W,P);D=P/'run';save(D/'review.json',dict(resultsSha256=sha(D/'results.jsonl'),method='All202 complete actual source/context/raw/edited outputs and12 probes personally read;29 unique failed tuples and58 changed canonical-control pairs separately read.',reviews=reviews))
passed=[a for a in ['baseline','balanced'] if all(j['choiceConforms'] for j in reviews if j['variant']==a+'-selection')];assert not passed
save(D/'selection.json',dict(qualified=False,selected=None,screenPassedArms=passed,productionActivated=False,freshExecuted=False,formatterExecuted=False,resultsSha256=sha(D/'results.jsonl'),reviewSha256=sha(D/'review.json'),reason='Youtu arms conform75/84 of101, harmful3/4, missed23/13, repaired24/34 of47. Same-request canonical controls conform92/94, harmful3/4, missed6/3.5 improved and32 regressed choices; reject both, no formatter/fresh expansion.'))
save(D/'paired-comparison.json',dict(records=pairs,limits='Same202 complete requests compared with prior actual canonical Qwen controls, not regenerated. Different architecture/tokenizer/template/quantization; no isolated parameter-count or quantization causality.58 differences include5 improvements,32 regressions and21 both-correct normalizations.'))
save(P/'path-mappings.json',dict(source='.workspace/rewrite-selector-youtu-v1',published='benchmarks/runs/rewrite-selector-youtu-v1',privateExcluded=['weights','runtime environments','native binaries'],legacyExecutorLabels='Immutable hybrid-preflight/selector-model-v1 labels retained; actual Youtu model/freeze/commands identify treatment.'))
for name in ['close.py','review-detail.py','qualify.py','analyze.py','launch.py']:shutil.copyfile(W/name,H/('original-'+name+'.txt'))
print('Published actual202 choices plus12 native responses, both configurations rejected.')
