"""Publish all personally inspected actual responses and the interrupted attempt."""
from pathlib import Path
import json,hashlib,shutil,importlib.util
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-minicpm-v1';W=R/'.workspace/rewrite-selector-minicpm-v1';P=R/'benchmarks/runs/rewrite-selector-minicpm-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rh(x):return hashlib.sha256(json.dumps(x,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def save(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
assert read(W/'comparison-terminal-status.json')['exitCode']==0
rows=list(map(json.loads,(W/'comparison-run/results.jsonl').read_text().splitlines()));assert len(rows)==202
spec=importlib.util.spec_from_file_location('minicpm_close',H/'report.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};targets={r['id']:r for r in read(R/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json')}
reviews=[];pairs=[]
for r in rows:
 assert r['renderError'] is None and r['response']['choices'][0]['finish_reason']=='stop'
 s=m.assessment(r,targets);q=refs[r['arm'],r['id']];assert r['request']==q['request'] and r['prepared']==q['prepared']
 note='Personally read all202 actual raw choices and edited sources, all12 native responses and the recovered interrupted native-log completion; all32 unique failed source/context/output tuples and55 changed pairs separately read. '
 if s['harmful']:note+='Unsupported meaning substitution: '+', '.join(e['original']+' -> '+e['replacement'] for e in r['edits'])+'. '
 elif s['missed']:note+='Required context-supported repair missed; source retains '+', '.join(k+'='+r['selection'][k] for k in targets[r['id']]['requiredRepairs'])+'. '
 else:note+='Choice respects frozen contextual repair, literal meaning and ambiguity rubric. '
 note+='Allowed readCash normalization and unchanged spelling are both accepted; unsupported readCache is not. No final-body/style, independent blind or minimum16GB acceptance. No semantic output rejection/source fallback.'
 reviews.append(dict(id=r['id'],variant=r['variant'],stage='selection',rowSha256=rh(r),faithful=not s['harmful'],sceneUseful=None,languagePreserved=True,choiceConforms=s['conforms'],harmfulChoice=s['harmful'],missedRepair=s['missed'],issues=['unsupported-substitution'] if s['harmful'] else ['required-repair-missed'] if s['missed'] else [],note=note))
 pairs.append(dict(id=r['id'],arm=r['arm'],rowSha256=rh(r),reference=r['comparisonReference'],sameRequest=True,samePrepared=True,changedChoice=r['selection']!=q['selection'],priorChoice=q['selection'],newChoice=r['selection'],priorConforms=m.assessment(q,targets)['conforms'],newConforms=s['conforms']))
save(W/'manual-review-completion.json',dict(complete=True,selectorResponses=202,nativeResponses=12,interruptedLogCompletions=1,ranges=[[0,60],[60,115],[115,165],[165,202]],uniqueFailedTuples=32,changedPairs=55,reviewer='Assistant, not independent or blind release review'))
shutil.copytree(W,P);D=P/'comparison-run'
save(D/'review.json',dict(resultsSha256=sha(D/'results.jsonl'),method='All202 actual source/context/raw/edited outputs personally read, plus12 native responses and1 interrupted log completion;32 failed tuples and55 changed pairs separately read. Not independent acceptance.',reviews=reviews))
save(D/'selection.json',dict(qualified=False,selected=None,screenPassedArms=[],productionActivated=False,freshExecuted=False,formatterExecuted=False,resultsSha256=sha(D/'results.jsonl'),reviewSha256=sha(D/'review.json'),reason='MiniCPM baseline/balanced conform80/77 versus Qwen92/94, harmful11/4 versus3/4, missed10/20 versus6/3, required repairs37/27 of47 versus41/44. Reject both; no final-body or fresh expansion.'))
save(D/'paired-comparison.json',dict(records=pairs,limits='202 unchanged requests and prepared candidates; priorQwen controls reused, not regenerated or newly counted. Model/architecture/tokenizer/template change together.55 choice differences:6 improve,35 regress,14 acceptable normalization differences. Not independent causal or release evidence.'))
probes=list(map(json.loads,(P/'preflight/results.jsonl').read_text().splitlines()));prs=[];vs=[];vrs=[]
for r in probes:
 expected={'e0':'replace' if r['sourceReference']['id']=='history-0095' else 'keep'};correct=json.loads(r['response']['choices'][0]['message']['content'])==expected;assert not correct
 note='Personally read full raw response/source/context. Valid JSON and normal stop; '+('missed supported load cash -> loadCache repair. ' if expected['e0']=='replace' else 'unsupported read cash -> readCache contradicts explicit cash/not-cache meaning. ')+('Schema removed and error persists. ' if r['job'].startswith('free-') else '')+'Runtime/cache compatibility only; semantic task failed, no final body.'
 prs.append(dict(id=r['id'],rowSha256=rh(r),protocolConforms=True,probeConforms=correct,note=note));v=m.runtime_row(r,refs[r['sourceReference']['arm'],r['sourceReference']['id']]);vs.append(v);vrs.append(dict(id=v['id'],variant=v['variant'],rowSha256=rh(v),probeConforms=correct,note=note))
save(P/'preflight/review.json',dict(resultsSha256=sha(P/'preflight/results.jsonl'),reviews=prs));V=P/'runtime-view';V.mkdir();(V/'results.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in vs));save(V/'manifest.json',dict(status='complete',generations=12,resultsSha256=sha(V/'results.jsonl'),rawResultsSha256=sha(P/'preflight/results.jsonl'),scope='Derived view of12 existing native probe responses, zero additional inference. Interrupted1 native-log completion separate, no fabricated response envelope.'));save(V/'review.json',dict(resultsSha256=sha(V/'results.jsonl'),reviews=vrs))
save(P/'path-mappings.json',dict(source='.workspace/rewrite-selector-minicpm-v1',published='benchmarks/runs/rewrite-selector-minicpm-v1',privateExcluded=['weights','runtime environments','native binaries'],completedRun='comparison-run',abortedAttempt='aborted-before-qualification',legacyExecutorLabels='Shared immutable executor names hybrid-preflight and selector-model-v1 retained; owned model/freeze/commands identify actual MiniCPM.'))
for source,name in [(Path(__file__),'original-close.py.txt'),(W/'qualify.py','original-qualify-failed.py.txt'),(W/'qualify-v2.py','original-qualify-v2.py.txt'),(W/'review-detail.py','original-review-detail.py.txt')]:shutil.copyfile(source,H/name)
print('Published214 complete responses plus1 reviewed native-log completion separately.')
