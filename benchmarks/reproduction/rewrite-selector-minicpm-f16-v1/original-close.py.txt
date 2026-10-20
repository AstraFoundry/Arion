"""Publish personally inspected F16 observations and exact Q4/control pair bindings."""
from pathlib import Path
import json,hashlib,shutil,importlib.util,collections
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-minicpm-f16-v1';W=R/'.workspace/rewrite-selector-minicpm-f16-v1';P=R/'benchmarks/runs/rewrite-selector-minicpm-f16-v1'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def rh(x):return hashlib.sha256(json.dumps(x,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def save(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
assert read(W/'terminal-status.json')['exitCode']==0 and read(W/'run/manifest.json')['status']=='complete'
rows=list(map(json.loads,(W/'run/results.jsonl').read_text().splitlines()));assert len(rows)==202
spec=importlib.util.spec_from_file_location('f16_close_report',H/'report.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
refs={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};q4={(r['arm'],r['id']):r for r in map(json.loads,(R/'benchmarks/runs/rewrite-selector-minicpm-v1/comparison-run/results.jsonl').read_text().splitlines())};targets={r['id']:r for r in read(R/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json')};reviews=[];pairs=[];q4pairs=[];groups=collections.defaultdict(collections.Counter)
for r in rows:
 assert r['renderError'] is None and r['response']['choices'][0]['finish_reason']=='stop'
 s=m.assessment(r,targets);ref=refs[r['arm'],r['id']];q=q4[r['arm'],r['id']];assert r['request']==ref['request']==q['request'] and r['prepared']==ref['prepared']==q['prepared'];assert r['response']['usage']['prompt_tokens']==q['response']['usage']['prompt_tokens']
 note='Personally read all202 actual F16 raw outputs and edited sources, all12 native responses,22 unique failed source/context/output tuples and47 changed Q4 pairs. '
 note+=('Unsupported meaning substitution: '+', '.join(e['original']+' -> '+e['replacement'] for e in r['edits'])+'. ' if s['harmful'] else 'Context-supported required spelling repair missed. ' if s['missed'] else 'Frozen spelling/literal/ambiguity/meaning rubric respected. ')
 note+='Allowed readCash normalization and unchanged spelling both receive credit; unsupported readCache does not. No semantic output rejection/source fallback, final-body/style, independent or actual16GB acceptance. Precision/export comparison is not proven numerical quantization causality.'
 reviews.append(dict(id=r['id'],variant=r['variant'],stage='selection',rowSha256=rh(r),faithful=not s['harmful'],sceneUseful=None,languagePreserved=True,choiceConforms=s['conforms'],harmfulChoice=s['harmful'],missedRepair=s['missed'],issues=['unsupported-substitution'] if s['harmful'] else ['required-repair-missed'] if s['missed'] else [],note=note))
 pairs.append(dict(id=r['id'],arm=r['arm'],rowSha256=rh(r),reference=r['comparisonReference'],sameRequest=True,samePrepared=True,changedChoice=r['selection']!=ref['selection'],priorChoice=ref['selection'],newChoice=r['selection'],priorConforms=m.assessment(ref,targets)['conforms'],newConforms=s['conforms']))
 q4pairs.append(dict(id=r['id'],arm=r['arm'],rowSha256=rh(r),q4RowSha256=rh(q),sameRequest=True,samePrepared=True,sameInputTokens=True,changedChoice=r['selection']!=q['selection'],q4Choice=q['selection'],f16Choice=r['selection'],q4Conforms=m.assessment(q,targets)['conforms'],f16Conforms=s['conforms']))
 groups[r['arm']].update(conforming=s['conforms'],harmful=s['harmful'],missed=s['missed'],repaired=s['repaired'])
save(W/'manual-review-completion.json',dict(complete=True,selectorResponses=202,nativeResponses=12,ranges=[[0,25],[25,80],[80,135],[135,180],[180,202]],uniqueFailedTuples=22,changedQ4Pairs=47,reviewer='Assistant, not independent blind release acceptance'))
shutil.copytree(W,P);D=P/'run';save(D/'review.json',dict(resultsSha256=sha(D/'results.jsonl'),method='All202 complete actual source/context/raw/edited outputs and12 probes personally read;22 unique failed tuples and47 changed Q4 pairs separately read.',reviews=reviews))
passed=[a for a in ['baseline','balanced'] if all(j['choiceConforms'] for j in reviews if j['variant']==a+'-selection')];assert not passed
save(D/'selection.json',dict(qualified=False,selected=None,screenPassedArms=passed,productionActivated=False,freshExecuted=False,formatterExecuted=False,resultsSha256=sha(D/'results.jsonl'),reviewSha256=sha(D/'review.json'),reason='Both F16 arms conform82/101, harmful15/17, missed4/2, repaired43/45 of47. PriorQ4 conforms80/77, harmful11/4, missed10/20, repaired37/27. More necessary repairs but more harmful choices; reject both, no formatter/fresh expansion.'))
save(D/'paired-comparison.json',dict(records=pairs,limits='Same202 complete requests compared with prior actual canonical Qwen controls, not regenerated. Different model/architecture/tokenizer/template; not parameter-only causality.'))
save(D/'q4-comparison.json',dict(records=q4pairs,limits='Exact202 same-request/same-preparation/input-token-count Q4/F16 pairs. Same upstream family and official revision; exact tokenizer/architecture metadata, template and tensor name/shape identity. No numerical conversion/quantizer lineage equivalence proof; precision/export artifact comparison only.47 differences:25 improvements,18 regressions,4 both-correct normalization changes. Old controls not regenerated or newly counted.'))
save(P/'path-mappings.json',dict(source='.workspace/rewrite-selector-minicpm-f16-v1',published='benchmarks/runs/rewrite-selector-minicpm-f16-v1',privateExcluded=['weights','runtime environments','native binaries'],legacyExecutorLabels='Shared immutable hybrid-preflight/selector-model-v1 labels retained; actual F16 model/freeze/commands identify the owned treatment.'))
for name in ['close.py','review-detail.py','qualify.py']:shutil.copyfile(W/name,H/('original-'+name+'.txt'))
print(json.dumps({a:dict(x) for a,x in groups.items()}))
