"""Replay real native typed decisions with explicit token-budget and review evidence."""
import argparse,collections,hashlib,importlib.util,json,math,pathlib,statistics,sys
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2];RUN=ROOT/'benchmarks/runs/rewrite-d1-evidence-v1'
GROUPS=[('transcript','rewrite-selector-d1-v1',''),('structured','rewrite-selector-d1-layout-v1','structured'),('sentence','rewrite-selector-d1-layout-v1','sentence'),('compact-structured','rewrite-selector-d1-compact-v1','structured'),('compact-sentence','rewrite-selector-d1-compact-v1','sentence'),('policy','rewrite-selector-d1-policy-v1','')]
def read(path):return json.loads(path.read_text())
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def rh(value):return hashlib.sha256(json.dumps(value,ensure_ascii=False,sort_keys=True).encode()).hexdigest()
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);value=importlib.util.module_from_spec(spec);before=list(sys.path);sys.path.insert(0,str(path.parent))
 try:spec.loader.exec_module(value)
 finally:sys.path[:]=before
 return value

def assessment(row,source,evaluator):
 answers={k:v['choice'] for k,v in row['response']['answers'].items()};bad={k:v for k,v in answers.items() if v not in evaluator['allowedChoices'][k]}
 h=source['prepared']['hypotheses'][0];raw=source['prepared']['source'].encode();changed=(raw[:h['startUtf8']]+h['candidateSpellings'][0].encode()+raw[h['endUtf8']:]).decode() if answers[h['id']]=='replace' else source['prepared']['source']
 return dict(choiceConforms=not bad,harmfulChoice=any(v=='replace' for v in bad.values()),missedRepair=any(v=='keep' for v in bad.values()),required=len(evaluator['requiredRepairs']),repaired=sum(answers[k]=='replace' for k in evaluator['requiredRepairs']),derivedSpellingText=changed)

def collect(review_required=True):
 references={(r['arm'],r['id']):r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())};rubric={e['id']:e for e in read(ROOT/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json')}
 budget_root=ROOT/'benchmarks/runs/rewrite-selector-d1-policy-v1/budget';bs=read(budget_root/'summary.json');budgets={(b['group'],b['id']):b for b in map(json.loads,(budget_root/'requests.jsonl').read_text().splitlines())}
 if bs['status']!='complete' or bs['modelGenerations']!=0 or bs['requestsSha256']!=sha(budget_root/'requests.jsonl') or len(budgets)!=36 or bs['priorInferenceTokenMatches']!=60 or bs['priorNativeLoggedPromptMatches']!=50 or bs['policyRequestsWithoutTruncation']!=6:raise ValueError('Changed native budget evidence')
 native_calls=list(map(json.loads,(budget_root/'native-calls.jsonl').read_text().splitlines()))
 if any(c['endpoint'] not in ['/tokenize','/detokenize'] or json.loads(c['rawResponse'])!=c['response'] for c in native_calls):raise ValueError('Invented zero-generation token evidence')
 token_calls={json.dumps(c['request'],sort_keys=True):c['response']['tokens'] for c in native_calls if c['endpoint']=='/tokenize'}
 detoken_calls={tuple(c['request']['tokens']):c['response']['content'] for c in native_calls if c['endpoint']=='/detokenize'}
 for b in budgets.values():
  used=[]
  for piece in b['pieces']:
   key=json.dumps(dict(content=piece['text'],add_special=False,parse_special=True,with_pieces=False),sort_keys=True);expected=token_calls[key];limit={'state':None,'question':96,'option':45,'other':None}[piece['kind']];retained=expected if limit is None else expected[:limit]
   if piece['tokens']!=expected or piece['limit']!=limit or piece['usedTokens']!=retained or piece['truncated']!=(retained!=expected):raise ValueError('Changed native trained piece clipping')
   used+=retained
  if b['effectiveTokens']!=used or b['inputTokens']!=len(used) or b['effectivePrompt']!=detoken_calls[tuple(used)] or b['truncatedPieces']!=sum(p['truncated'] for p in b['pieces']):raise ValueError('Changed effective native input')
 if bs['priorTruncatedRequests']!=sum(b['truncatedPieces']>0 for b in budgets.values() if b['group']!='policy') or bs['maximumPolicyInputTokens']!=max(b['inputTokens'] for b in budgets.values() if b['group']=='policy'):raise ValueError('Hidden native truncation or changed complete policy budget')
 results=[];groups={};sources=[]
 for name,study,kind in GROUPS:
  h=ROOT/'benchmarks/reproduction'/study;fpath=h/kind/'freeze.json' if kind else h/'freeze.json';f=read(fpath);p=ROOT/'benchmarks/runs'/study;directory=p/'preflight'/kind if kind else p/'preflight'
  inputs=read(h/kind/'probe-inputs.json' if kind else h/'probe-inputs.json')['records'];contract=module('native_contract_'+name.replace('-','_'),h/('request_contract.py' if name=='transcript' else 'layout.py'))
  contract.validate(inputs,kind) if kind else contract.validate(inputs)
  rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()));manifest=read(directory/'manifest.json')
  if manifest['status']!='complete' or len(rows)!=12 or manifest['typedRequests']!=12 or manifest['modelResponses']!=12 or manifest['generatedTokens']!=0 or manifest['resultsSha256']!=sha(directory/'results.jsonl') or manifest['freezeSha256']!=sha(fpath) or manifest['model']!=f['model']:raise ValueError('Incomplete actual native decisions')
  if f['model']['parameters']!=380732161 or f['model']['familyParameters']!=587161089:raise ValueError('Changed model eligibility')
  summaries=collections.Counter();native_prompts={}
  for backend,gpu,placement in [('metal',99,b'offloaded 19/19 layers to GPU'),('cpu',0,b'offloaded 0/19 layers to GPU')]:
   log=(directory/backend/'server.log').read_bytes();worker=next(w for w in manifest['workers'] if w['gpuLayersRequested']==gpu)
   props=read(directory/backend/'props.json')
   if placement not in log or worker['exitCode']!=0 or props['build_info']!='b11734-f3fb2fcdb' or '--no-warmup' not in worker['command'] or worker['command'][worker['command'].index('-ngl')+1]!=str(gpu):raise ValueError('Missing actual native runtime placement')
   observed=[]
   for line in log.splitlines():
    if b'launching slot : ' in line:
     slot=json.loads(line.split(b'launching slot : ',1)[1])
     if slot.get('prompt'):observed.append(slot['prompt'])
   if len(observed)!=5:raise ValueError('Changed actual prompt-log scope')
   native_prompts[gpu]=observed
  for i,row in enumerate(rows):
   item=inputs[i%6];source=references['baseline',row['id']];backend='metal' if i<6 else 'cpu';expected_gpu=99 if i<6 else 0;ref=row['sourceReference']
   if row['id']!=item['id'] or row['scene']!=source['scene'] or row['stage']!='typed-decision' or row['gpuLayersRequested']!=expected_gpu or row['request']!=item['request'] or row['sourceReference']!=item['sourceReference'] or ref['rowSha256']!=rh(source):raise ValueError('Changed typed input/source identity')
   request_path=directory/backend/(str(i%6)+'.request.json');response_path=directory/backend/(str(i%6)+'.response.json')
   if sha(request_path)!=row['requestSha256'] or read(request_path)!=row['request'] or sha(response_path)!=row['rawResponseSha256'] or read(response_path)!=row['response'] or row['httpStatus']!=200:raise ValueError('Changed exact native HTTP bytes')
   response=row['response']
   if set(response['answers'])!=set(row['request']['questions']) or response['usage']['output_tokens']!=0 or row['generatedTokens']!=0:raise ValueError('Invented generation or answer ID')
   for identity,answer in response['answers'].items():
    probabilities=answer['probabilities']
    if answer['type']!='choice' or set(probabilities)!=set(row['request']['questions'][identity]['criteria']) or not all(isinstance(v,(int,float)) and math.isfinite(v) and 0<=v<=1 for v in probabilities.values()) or abs(sum(probabilities.values())-1)>1e-10 or answer['choice']!=max(probabilities,key=probabilities.get):raise ValueError('Changed actual typed choice/probabilities')
   b=budgets[study+('/'+kind if kind else '') if name!='policy' else 'policy',row['id']]
   if b['request']!=row['request'] or b['requestSha256']!=rh(row['request']) or b['inputTokens']!=response['usage']['input_tokens'] or b['modelGenerations']!=0 or len(b['effectiveTokens'])!=b['inputTokens']:raise ValueError('Native effective budget mismatch')
   if i%6<5 and native_prompts[expected_gpu][i%6]!=b['effectivePrompt']:raise ValueError('Effective prompt differs from actual log')
   if name=='policy' and b['truncatedPieces']!=0:raise ValueError('Clipped corrected policy')
   a=assessment(row,source,rubric[row['id']]);summaries.update(cases=1,conforming=a['choiceConforms'],harmful=a['harmfulChoice'],missed=a['missedRepair'],repaired=a['repaired'],required=a['required'],clippedRequests=b['truncatedPieces']>0)
   results.append(dict(group=name,study=study,kind=kind,row=row,rowSha256=rh(row),source=source,assessment=a,budget=b,rawResponsePath=str(response_path.relative_to(ROOT)),actualRuntimeCommand=next(w['command'] for w in manifest['workers'] if w['gpuLayersRequested']==expected_gpu)))
  groups[name]=dict(summaries,backendChoiceAgreement=sum(rows[i]['response']['answers']['e0']['choice']==rows[i+6]['response']['answers']['e0']['choice'] for i in range(6)),maximumProbabilityDelta=max(abs(rows[i]['response']['answers']['e0']['probabilities']['replace']-rows[i+6]['response']['answers']['e0']['probabilities']['replace']) for i in range(6)),maximumInputTokens=max(r['response']['usage']['input_tokens'] for r in rows),metalRequestMedianMs=statistics.median(r['requestMs'] for r in rows[:6]),cpuRequestMedianMs=statistics.median(r['requestMs'] for r in rows[6:]),qualified=False)
  sources+=[dict(path=str(x.relative_to(ROOT)),sha256=sha(x)) for x in [fpath,directory/'results.jsonl',directory/'manifest.json']]
 if len(results)!=72:raise ValueError('Incomplete six-layout scope')
 if review_required:
  review=read(RUN/'review.json');idx={(r['group'],r['gpu'],r['id']):r for r in review['reviews']}
  if len(idx)!=72:raise ValueError('Missing personal typed review')
  for item in results:
   row=item['row'];record=idx[item['group'],row['gpuLayersRequested'],row['id']]
   if record['rowSha256']!=item['rowSha256'] or any(record[k]!=v for k,v in item['assessment'].items()) or not record['personallyRead'] or not record['note']:raise ValueError('Changed source-bound review')
   item['review']=record
  decision=read(RUN/'selection.json')
  if decision!=dict(qualified=False,selected=None,productionActivated=False,mainComparisonExecuted=False,reason='No full-policy, no-truncation configuration passes all six probes on both backends. Compact sentence has12/12 correct but clipped trusted instructions; corrected full policy6/12 correct. Preserve all72 native answers; no further expansion or release credit.'):raise ValueError('Unsupported native candidate qualification')
 summary=dict(modelResponses=72,generatedTokens=0,finalBodies=0,independentSources=6,groups=groups,nativeBudget=bs,allChoicesBackendAgree=all(g['backendChoiceAgreement']==6 for g in groups.values()),sources=sources,limits='Six exposed repeated development sources, actual native typed outputs; no synthesized chat completions, confidence threshold, semantic rejection or source fallback. No new ASR/Vision/formatter/full-chain/native insertion/independent/actual16GB acceptance. Backend probabilities differ; no numeric parity or calibration guarantee.')
 return summary,results

def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();summary,items=collect();args.out.mkdir(parents=True,exist_ok=False)
 (args.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n');lines=['# Native d1 decision packaging review','','Actual typed answers retained; derived spelling text is audit illustration only, not generated or delivered prose.']
 for item in items:
  row=item['row'];lines += ['',f"## {item['group']} / {row['gpuLayersRequested']} / {row['id']} / {row['scene']}",'','```json',json.dumps(dict(source=item['source']['source'],context=item['source']['context'],actualNativeRequest=row['request'],actualNativeResponse=row['response'],derivedAudit=item['assessment'],effectiveNativePrompt=item['budget']['effectivePrompt'],clippedPieces=item['budget']['truncatedPieces']),ensure_ascii=False,indent=2),'```']
 (args.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
