"""Paired local generation, offline delivery-policy ablations and verifier calibration."""
import argparse
import collections
import hashlib
import json
import os
import pathlib
import random
import re
import select
import subprocess
import time

HERE = pathlib.Path(__file__).resolve().parent
ROOT = next(p for p in HERE.parents if (p/'models/catalog.json').exists())
BUILD = ROOT/'.workspace/scene-rewrite-alignment-v1/build'
RUNTIME = pathlib.Path('/Applications/Arion.app/Contents/Resources/llm')
MODEL = pathlib.Path.home()/'Library/Application Support/Arion/models/qwen3-1.7b-int4-cpu'
def read(p): return json.loads(pathlib.Path(p).read_text())
def sha(p):
    h=hashlib.sha256()
    with pathlib.Path(p).open('rb') as f:
        for b in iter(lambda:f.read(1024*1024),b''): h.update(b)
    return h.hexdigest()
def write(p,obj): pathlib.Path(p).write_text(json.dumps(obj,ensure_ascii=False,indent=2)+'\n')
def check_freeze():
    for name,h in read(HERE/'freeze.json')['files'].items():
        if sha(HERE/name)!=h: raise ValueError('Frozen source changed: '+name)
def provenance():
    entry=next(x for x in read(ROOT/'models/catalog.json')['rewriters'] if x['id']==MODEL.name)
    for name,pin in entry['files'].items():
        if sha(MODEL/name)!=pin['sha256']: raise ValueError('Model mismatch: '+name)
    return dict(model=entry,modelFiles={n:sha(MODEL/n) for n in entry['files']},
        binaries={str(p):sha(p) for p in [RUNTIME/'arion-llm',RUNTIME/'libonnxruntime-genai.dylib',RUNTIME/'libonnxruntime.dylib',BUILD/'worker',BUILD/'validate']},
        build=read(BUILD/'manifest.json'),freezeSha256=sha(HERE/'freeze.json'),
        gitHead=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),
        gitStatus=subprocess.check_output(['git','status','--short'],cwd=ROOT,text=True),
        hardware={k:subprocess.check_output(cmd,text=True).strip() for k,cmd in dict(cpu=['sysctl','-n','machdep.cpu.brand_string'],ram=['sysctl','-n','hw.memsize'],os=['sw_vers'],power=['pmset','-g','batt']).items()})

class Worker:
    def __init__(self,out,name,template=None,audit=False):
        self.seq=0
        env=dict(os.environ,ORT_LIB_PATH=str(RUNTIME))
        executable=RUNTIME/'arion-llm' if template is None else BUILD/'worker'
        if template: env['ARION_EXPERIMENT_TEMPLATE']=str(template)
        if audit: env['ARION_EXPERIMENT_AUDIT']='1'
        self.stderr=(out/(name+'-stderr.log')).open('x')
        self.p=subprocess.Popen([str(executable)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.stderr,text=True,bufsize=1,env=env)
        self.load=self.ask('load',timeout=90,runtime=str(RUNTIME),model=str(MODEL),threads=4,maxLength=4096,maxNewTokens=12 if audit else 384)
        write(out/(name+'-load.json'),self.load)
    def send(self,obj):
        self.p.stdin.write(json.dumps(obj,ensure_ascii=False)+'\n');self.p.stdin.flush()
    def ask(self,op,timeout=10,**kw):
        self.seq+=1;ident=self.seq
        self.send(dict(version=2,id=ident,op=op,**kw))
        timed_out=False
        if not select.select([self.p.stdout],[],[],timeout)[0]:
            timed_out=True
            self.send(dict(version=2,id=0,op='cancel',target=ident))
            if not select.select([self.p.stdout],[],[],10)[0]: raise TimeoutError('Worker did not acknowledge cancellation')
        line=self.p.stdout.readline()
        if not line: raise RuntimeError('Worker exited')
        result=json.loads(line)
        if result['id']!=ident or not result['ok']: raise RuntimeError(result)
        if timed_out:result['applicationTimeout']=True
        return result
    def close(self):
        self.ask('close');self.p.wait(timeout=10);self.stderr.close()

def strict_check(source,response,context):
    req=dict(operation='validate',mode='product',text=source,candidate=response.get('text',''),truncated=response.get('truncated',False),context=context)
    return json.loads(subprocess.check_output([str(BUILD/'validate')],input=json.dumps(req),text=True))['verdict']
def envelope(response,strict):
    if response.get('applicationTimeout'):return 'timeout'
    if response.get('cancelled'):return 'cancelled'
    if strict in ['empty','markup','truncated']:return strict
    if len(response.get('text',''))>2048:return 'output_limit'
    return 'accepted'
def audit(worker,source,candidate,scene,session):
    payload=json.dumps(dict(source=source,candidate=candidate,scene=scene),ensure_ascii=False)
    start=time.monotonic()
    result=worker.ask('rewrite',session=session,scene='chat',text=payload)
    verdict=result.get('text','').strip()
    if verdict not in ['PASS','FAIL'] or result.get('truncated') or result.get('cancelled') or result.get('applicationTimeout'):verdict='INVALID'
    return dict(response=result,verdict=verdict,wallMs=(time.monotonic()-start)*1000)
def fact_score(sample,text):
    failed=[p for p in sample['facts'] if not re.search(p,text,re.I|re.S)]
    lines=[l for l in text.splitlines() if l.strip()]
    layout=True
    if sample['scene'] in ['work','prompt']:layout=len(lines)>=2
    if sample['scene']=='email':layout=bool(re.search(r'\n\s*\n',text))
    return dict(factsPassed=not failed,failedPatterns=failed,layoutPassed=layout)
def metrics(samples,rows):
    byid={s['id']:s for s in samples};groups={}
    for generation in ['production','scene']:
        subset=[r for r in rows if r['generationArm']==generation]
        for policy in ['strict','envelope','semantic']:
            counts=collections.Counter();durations=[]
            for row in subset:
                sample=byid[row['id']];response=row['response'];accepted=row['delivery'][policy]=='accepted'
                raw=fact_score(sample,response.get('text',''))
                delivered=response['text'] if accepted else response['cleanup']['text']
                score=fact_score(sample,delivered)
                counts['cases']+=1;counts['accepted']+=accepted
                counts['rawFactsPassed']+=raw['factsPassed']
                counts['rawFactsAndLayoutPassed']+=raw['factsPassed'] and raw['layoutPassed']
                counts['deliveredFactsPassed']+=score['factsPassed']
                counts['deliveredFactsAndLayoutPassed']+=score['factsPassed'] and score['layoutPassed']
                counts['acceptedConstraintFailures']+=accepted and not raw['factsPassed']
                counts['rejectedFactPassingCandidates']+=(not accepted) and raw['factsPassed'] and row['envelope']=='accepted'
                ms=row['generationMs']+(row.get('audit',{}).get('wallMs',0) if policy=='semantic' else 0)
                durations.append(ms)
            durations.sort()
            groups[generation+'/'+policy]=dict(counts,finalP50Ms=durations[len(durations)//2],finalP95Ms=durations[min(len(durations)-1,int(.95*len(durations)))])
    return groups

def main():
    p=argparse.ArgumentParser();p.add_argument('--split',choices=['dev','test'],required=True);p.add_argument('--out',type=pathlib.Path,required=True);p.add_argument('--selection',type=pathlib.Path);a=p.parse_args()
    check_freeze()
    if (a.split=='test') != bool(a.selection):raise ValueError('Test requires frozen dev selection')
    if a.selection and read(a.selection)['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Selection freeze mismatch')
    out=a.out.resolve();out.mkdir(parents=True,exist_ok=False)
    manifest=dict(status='running',split=a.split,kind='synthetic-development-diagnostic',provenance=provenance(),selectionSha256=sha(a.selection) if a.selection else None)
    write(out/'manifest.json',manifest)
    samples=[s for s in read(HERE/'corpus.json')['samples'] if s['split']==a.split]
    rows=[];worker=None
    try:
        if a.split=='dev':
            worker=Worker(out,'calibration',HERE/'audit-template.json',audit=True)
            calibration=[]
            for pair in read(HERE/'calibration.json'):
                result=audit(worker,pair['source'],pair['candidate'],pair['scene'],pair['id'])
                calibration.append(dict(pair=pair,audit=result))
                print('calibration',pair['id'],pair['expected'],result['verdict'],flush=True)
                write(out/'calibration.json',calibration)
            worker.close();worker=None
        for arm in ['production','scene']:
            worker=Worker(out,arm,None if arm=='production' else HERE/'scene-template.json')
            jobs=list(samples);random.Random(1729).shuffle(jobs)
            with (out/(arm+'.jsonl')).open('x') as f:
                for sample in jobs:
                    # Match production scene mapping for the baseline; candidate receives the real scene.
                    scene=sample['scene']
                    if arm=='production':
                        enumerated=any(x in sample['source'].lower() and y in sample['source'].lower() for x,y in [('第一','第二'),('一是','二是'),('一个是','另一个是'),('首先','其次'),('first','second')])
                        scene='email' if scene=='email' else 'list' if scene in ['work','prompt'] and enumerated else 'chat'
                    start=time.monotonic()
                    response=worker.ask('rewrite',session=sample['id'],scene=scene,text=sample['source'])
                    duration=(time.monotonic()-start)*1000
                    strict=strict_check(sample['source'],response,sample['context'])
                    env=envelope(response,strict)
                    row=dict(id=sample['id'],generationArm=arm,requestedScene=sample['scene'],effectiveModelScene=scene,response=response,generationMs=duration,strict=strict,envelope=env,delivery={'strict':strict if env not in ['cancelled','timeout'] else env,'envelope':env})
                    rows.append(row);f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush()
                    print(a.split,arm,sample['id'],strict,env,round(duration),flush=True)
            worker.close();worker=None
        worker=Worker(out,'auditor',HERE/'audit-template.json',audit=True)
        with (out/'audited.jsonl').open('x') as f:
            for row in rows:
                sample=next(s for s in samples if s['id']==row['id'])
                if row['envelope']=='accepted':
                    row['audit']=audit(worker,sample['source'],row['response']['text'],sample['scene'],row['generationArm']+'-'+row['id'])
                    row['delivery']['semantic']='accepted' if row['audit']['verdict']=='PASS' else 'semantic_'+row['audit']['verdict'].lower()
                else:row['delivery']['semantic']=row['envelope']
                row['rawConstraintScore']=fact_score(sample,row['response'].get('text',''))
                f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush()
                print(a.split,'audit',row['generationArm'],row['id'],row['delivery']['semantic'],flush=True)
        worker.close();worker=None
        summary=metrics(samples,rows);write(out/'summary.json',summary)
        if a.split=='dev':
            calibration=read(out/'calibration.json');bad=sum(r['pair']['expected']=='FAIL' and r['audit']['verdict']=='PASS' for r in calibration);good=sum(r['pair']['expected']=='PASS' and r['audit']['verdict']=='PASS' for r in calibration)
            base=summary['production/strict'];candidate=summary['scene/semantic']
            passed=bad==0 and good/12>=.95 and candidate['deliveredFactsAndLayoutPassed']-base['deliveredFactsAndLayoutPassed']>=4 and candidate['acceptedConstraintFailures']<=base['acceptedConstraintFailures'] and candidate['finalP95Ms']<=1500
            selection=dict(selected='scene/semantic' if passed else None,decision='candidate' if passed else 'no-promotion',freezeSha256=sha(HERE/'freeze.json'),calibrationHarmfulAccepted=bad,calibrationFaithfulAccepted=good,devSummarySha256=sha(out/'summary.json'),reason='All frozen gates required; test runs the predeclared factorial for diagnosis even after no-promotion.')
            write(out/'selection.json',selection)
        manifest.update(status='complete',cases=len(samples),generations=len(rows),auditedSha256=sha(out/'audited.jsonl'),summarySha256=sha(out/'summary.json'));write(out/'manifest.json',manifest)
    except Exception as e:
        if worker:
            worker.p.kill();worker.p.wait()
        manifest.update(status='failed',error=type(e).__name__+': '+str(e));write(out/'manifest.json',manifest);raise
    print(json.dumps(summary,indent=2),flush=True)
if __name__=='__main__':main()
