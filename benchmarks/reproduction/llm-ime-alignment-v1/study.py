"""Reference-isolated generic LLM candidate-ranking pilot and report replay."""
import argparse, hashlib, json, math, os, platform, random, subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
SOURCE = ROOT / 'benchmarks/runs/rime-llm-reranker-v1/candidates/raw'
CORPUS = ROOT / 'benchmarks/corpora/rime-llm-reranker-v1/corpus.json'

def read(p): return json.loads(Path(p).read_text())
def sha(p): return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def write(p, obj): Path(p).write_text(json.dumps(obj, ensure_ascii=False, indent=2)+'\n')
def lines(p): return [json.loads(s) for s in Path(p).read_text().splitlines()]
def quantile(xs, q):
    xs=sorted(xs)
    if not xs:return None
    at=(len(xs)-1)*q;lo=int(at);hi=min(lo+1,len(xs)-1)
    return xs[lo]+(xs[hi]-xs[lo])*(at-lo)

def source_requests():
    corpus=read(CORPUS);requests=[];sources={str(CORPUS.relative_to(ROOT)):sha(CORPUS)}
    for case in corpus['cases']:
        p=SOURCE/(case['id']+'.json');raw=read(p)
        sources[str(p.relative_to(ROOT))]=sha(p)
        words=raw['beforeConfirmation']['candidates'][:27]
        for arm in ['context','no-context']:
            requests.append({'session_id':case['id']+'/'+arm,'generation':1,'revision':1,'input':case['input'],'before':case['context'] if arm=='context' else '', 'candidates':[{'id':f'c{i}','text':t,'source':'native-rime-snapshot'} for i,t in enumerate(words)]})
    random.Random(42).shuffle(requests)
    warmups=[{'session_id':f'warmup-{i}','generation':1,'revision':1,'input':'maomi','before':'院子里有一只','candidates':[{'id':'c0','text':'猫咪','source':'authored-warmup'},{'id':'c1','text':'茂密','source':'authored-warmup'}]} for i in range(2)]
    return warmups+requests,sources

def prepare(out):
    out=Path(out);out.mkdir(parents=True,exist_ok=False)
    requests,sources=source_requests()
    (out/'requests.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False)+'\n' for r in requests))
    write(out/'provenance.json',{'corpus':'rime-llm-reranker-v1','split':'reused-authored-exploratory','labels':'Evaluator only; not independently reviewed or held out. Existing prior study outputs were known.','candidateBoundary':'Frozen actual native Rime first-page snapshots; no consumed-Pinyin spans available. Equal Unicode scalar length gate is a proxy.','design':'paired-context-ablation','seed':42,'warmups':2,'sources':sources})

def run(args):
    work=Path(args.work).resolve();out=work/'inference'
    cmd=[args.driver,'-requests',str(work/'requests.jsonl'),'-out',str(out),'-timeout',args.timeout]
    if args.provider:
        cmd+=['-provider',args.provider,'-model-id',args.model_id,'-base',args.base,'-format',args.format]
    else:
        cmd+=['-worker',args.worker,'-runtime',args.runtime,'-model',args.model,'-model-sha256',args.model_sha256]
    if args.ungated:cmd+=['-ungated']
    metadata={'platform':platform.platform(),'machine':platform.machine(),'gitRevision':subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),'gitStatus':subprocess.check_output(['git','status','--short'],cwd=ROOT,text=True),'timer':'Go time.Now/time.Since monotonic wall clock','concurrency':1,'nativeUI':False,'ASRContention':False,'configuration':{'timeout':args.timeout,'threads':4,'maxLength':4096,'maxNewTokens':256,'rankMaxTokens':48,'slotErasedBetweenRequests':not bool(args.provider)},'sources':{str(p.relative_to(ROOT)):sha(p) for base in ['packages/core/domain/keyboard_ranking.go','apps/desktop/infrastructure/llm/ranking/ranking.go','apps/desktop/infrastructure/llm/gguf/completion.go','apps/desktop/cmd/arion-ime-rank-eval/main.go'] for p in [ROOT/base]}}
    write(work/'environment.json',metadata)
    with (work/'driver.log').open('x') as f: subprocess.run(cmd,cwd=ROOT,stdout=f,stderr=subprocess.STDOUT,check=True)

def report(work):
    work=Path(work);reqs=lines(work/'requests.jsonl');obs=lines(work/'inference/observations.jsonl');status=read(work/'inference/status.json');prov=read(work/'provenance.json')
    if status['status']!='complete' or status['count']!=len(reqs) or len(obs)!=len(reqs) or status['requestsSha256']!=sha(work/'requests.jsonl'):raise ValueError('Incomplete or changed run')
    for p,h in prov['sources'].items():
        if sha(ROOT/p)!=h:raise ValueError('Changed source fixture')
    if reqs!=source_requests()[0]:raise ValueError('Changed or label-contaminated producer requests')
    cases={x['id']:x for x in read(CORPUS)['cases']};groups={a:[] for a in ['context','no-context']};warm=[]
    seen=set()
    for req,o in zip(reqs,obs):
        if req!=o['request'] or req['session_id'] in seen:raise ValueError('Request mismatch or duplicate')
        seen.add(req['session_id'])
        if req['session_id'].startswith('warmup-'):warm.append(o);continue
        caseid,arm=req['session_id'].split('/');case=cases[caseid];cs=req['candidates'];ids=[c['id'] for c in cs]
        before=[c['text'] for c in cs];after=before[:];eligible=[c for c in cs if len(c['text'])==len(cs[0]['text'])]
        if not o['error']:
            rs=o['result']
            if any(c.get('score') is not None and (not isinstance(c['score'],(float,int)) or not math.isfinite(c['score'])) for c in rs['candidates']):raise ValueError('Nonfinite score')
            order=[c['id'] for c in rs['candidates']]
            if len(order)!=len(ids) or len(set(order))!=len(order) or set(order)!=set(ids) or rs['session_id']!=req['session_id'] or rs['revision']!=req['revision'] or rs['generation']!=req['generation']:raise ValueError('Fabricated/stale rank result')
            texts={c['id']:c['text'] for c in cs};after=[texts[i] for i in order]
            if len(after[0])!=len(before[0]):raise ValueError('Incompatible promotion')
        acceptable=case['acceptable'];base=before[0] in acceptable;ranked=after[0] in acceptable
        rr=lambda words:next((1/(i+1) for i,w in enumerate(words) if w in acceptable),0)
        groups[arm].append({'id':caseid,'group':case['group'],'input':case['input'],'before':before[0],'after':after[0],'baselineCorrect':base,'rankedCorrect':ranked,'rescue':not base and ranked,'breakage':base and not ranked,'oracle':any(t in acceptable for t in before),'eligibleOracle':any(c['text'] in acceptable for c in eligible),'modelCalled':bool(o['completion']['chatRequest']['messages']),'decision':o['result'].get('decision','model' if not o['error'] else 'error'),'baselineRR':rr(before),'rankedRR':rr(after),'wallMs':o['wallMs'],'error':o['error']})
    result={'design':'paired-context-ablation; exploratory reused authored corpus','status':'complete','arms':{},'warmups':[{'wallMs':o['wallMs'],'error':o['error']} for o in warm],'limitations':['No independent annotation review or held-out quality estimate.','No current learned-pair baseline: an empty personal store preserves producer order; no test-derived memory was created.','No rendered UI, realtime typing, cloud/subscription inference or concurrent ASR measured.','Errors retain displayed producer order in metrics and remain explicit errors, never successful inference.','Candidate-span gate uses equal scalar length; actual consumed Pinyin coverage is unavailable.']}
    for arm,rows in groups.items():
        if len(rows)!=len(cases) or {r['id'] for r in rows}!=set(cases):raise ValueError('Unpaired or missing cases')
        result['arms'][arm]={'n':len(rows),**{k:sum(bool(r[k]) for r in rows) for k in ['baselineCorrect','rankedCorrect','rescue','breakage','oracle','eligibleOracle']},'errors':sum(bool(r['error']) for r in rows),'modelCalls':sum(r['modelCalled'] for r in rows),'modelWallMs':{f'p{int(q*100)}':quantile([r['wallMs'] for r in rows if r['modelCalled']],q) for q in [.5,.95,.99]},'baselineMRR':sum(r['baselineRR'] for r in rows)/len(rows),'rankedMRR':sum(r['rankedRR'] for r in rows)/len(rows),'wallMs':{f'p{int(q*100)}':quantile([r['wallMs'] for r in rows],q) for q in [.5,.95,.99]},'cases':rows}
    return result

def main():
    p=argparse.ArgumentParser();p.add_argument('op',choices=['prepare','run','report','verify']);p.add_argument('--work',required=True);p.add_argument('--out');p.add_argument('--driver');p.add_argument('--worker');p.add_argument('--runtime');p.add_argument('--model');p.add_argument('--model-sha256');p.add_argument('--timeout',default='1500ms');p.add_argument('--provider');p.add_argument('--model-id');p.add_argument('--base',default='');p.add_argument('--format',default='');p.add_argument('--ungated',action='store_true');a=p.parse_args()
    if a.op=='prepare':prepare(a.work)
    elif a.op=='run':run(a)
    elif a.op=='report':
        r=report(a.work)
        if Path(a.out).exists():raise FileExistsError(a.out)
        write(a.out,r);print(json.dumps({k:{x:y for x,y in v.items() if x!='cases'} for k,v in r['arms'].items()},indent=2))
    elif a.op=='verify':
        r=report(a.work)
        if r!=read(a.out):raise ValueError('Report replay mismatch')
        print('Exact report replay passed')
if __name__=='__main__':main()

# Separate event replay; inference still receives reference-free snapshots only.
def typing_schedule():
    cases=read(CORPUS)['cases']
    selected=[c for c in cases if c['group']=='context'][:16]+[c for c in cases if c['group'] in {'abbreviation','partial','long-input'}][:24:4]
    events=[];at=0
    for case in selected:
        raw=read(SOURCE/(case['id']+'.json'))
        for i,step in enumerate(raw['steps']):
            words=step['snapshot']['candidates']
            if not words:raise ValueError('Missing native prefix candidates')
            events.append({'atMs':at,'request':{'session_id':case['id'],'generation':1,'revision':i+1,'input':raw['input'][:i+1],'before':case['context'],'candidates':[{'id':f'c{j}','text':w,'source':'native-rime-snapshot'} for j,w in enumerate(words[:27])]}})
            at+=100
        at+=1000
    return events

def schedule_report(schedule, observations):
    events=read(schedule);rows=lines(observations)
    if events!=typing_schedule():raise ValueError('Changed scheduled producer requests')
    submitted=[r['data'] for r in rows if r['kind']=='submit']
    if submitted!=events or len([r for r in rows if r['kind']=='complete'])!=1:raise ValueError('Incomplete schedule')
    current=None;answers={};accepted=0;late=[];stale=0;seen=set();inference=[]
    for row in rows:
        if not math.isfinite(row['atMs']):raise ValueError('Invalid event time')
        if row['kind']=='submit':
            current=row['data']['request'];late.append(row['atMs']-row['data']['atMs'])
        if row['kind']=='inference':inference.append(row['data'])
        if row['kind']=='outcome':
            out=row['data'];req=out['request'];key=(req['session_id'],req['revision'])
            if key in seen:raise ValueError('Duplicate schedule outcome')
            seen.add(key);answers[key]=out
            if out['canApply']:
                accepted+=1
                if req!=current or out['error']:raise ValueError('Stale or failed scheduled outcome accepted')
                ids=[c['id'] for c in req['candidates']];order=[c['id'] for c in out['result']['candidates']]
                if len(order)!=len(ids) or set(order)!=set(ids) or len(set(order))!=len(order):raise ValueError('Invalid scheduled permutation')
    if len(seen)!=len(events) or stale:raise ValueError('Missing outcomes or stale accepted result')
    last={e['request']['session_id']:e['request'] for e in events};cases={c['id']:c for c in read(CORPUS)['cases']}
    finals=[]
    for caseid,req in last.items():
        out=answers[(caseid,req['revision'])];words={c['id']:c['text'] for c in req['candidates']};before=req['candidates'][0]['text'];after=words[out['result']['candidates'][0]['id']] if out['canApply'] else before;gold=cases[caseid]['acceptable']
        finals.append({'id':caseid,'before':before,'after':after,'baselineCorrect':before in gold,'rankedCorrect':after in gold,'enhancementAccepted':out['canApply'],'error':out['error']})
    return {'status':'complete','snapshots':len(events),'fixtures':len(last),'inferenceAttempts':len(inference),'cancelledOrFailedInference':sum(bool(r['error']) for r in inference),'totalInferenceWallMs':sum(r['wallMs'] for r in inference),'acceptedProposals':accepted,'invalidatedOutcomes':sum(bool(a['error']) for a in answers.values()),'staleAccepted':stale,'submissionLatenessMs':{'p95':quantile(late,.95),'max':max(late)},'finalBaselineCorrect':sum(r['baselineCorrect'] for r in finals),'finalRankedCorrect':sum(r['rankedCorrect'] for r in finals),'finalCases':finals,'boundary':'Real monotonic Go scheduling plus real local inference over frozen engine snapshots; no OS key events, rendered UI or ASR contention.'}
