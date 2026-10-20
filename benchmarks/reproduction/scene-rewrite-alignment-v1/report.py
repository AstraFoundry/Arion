"""Replay completed study rows without models; reject missing or modified evidence."""
import argparse
import collections
import csv
import hashlib
import json
import pathlib
import random
import sys

import study

HERE=pathlib.Path(__file__).resolve().parent
ROOT=study.ROOT
def load_run(path,split):
    path=pathlib.Path(path)
    manifest=study.read(path/'manifest.json')
    if manifest['status']!='complete' or manifest['split']!=split:raise ValueError('Incomplete or wrong-split run')
    if manifest['provenance']['freezeSha256']!=study.sha(HERE/'freeze.json'):raise ValueError('Frozen input mismatch')
    if study.sha(path/'audited.jsonl')!=manifest['auditedSha256']:raise ValueError('Changed audited observations')
    if study.sha(path/'summary.json')!=manifest['summarySha256']:raise ValueError('Changed original summary')
    rows=[json.loads(l) for l in (path/'audited.jsonl').read_text().splitlines()]
    samples=[s for s in study.read(HERE/'corpus.json')['samples'] if s['split']==split]
    expected={(s['id'],arm) for s in samples for arm in ['production','scene']}
    actual=[(r['id'],r['generationArm']) for r in rows]
    if len(actual)!=len(expected) or len(set(actual))!=len(actual) or set(actual)!=expected:raise ValueError('Incomplete, duplicate or unknown observations')
    byid={s['id']:s for s in samples}
    for row in rows:
        if row['rawConstraintScore']!=study.fact_score(byid[row['id']],row['response'].get('text','')):raise ValueError('Changed constraint score')
        if row['envelope']!=study.envelope(row['response'],row['strict']):raise ValueError('Changed envelope decision')
        if row['delivery']['envelope']!=row['envelope']:raise ValueError('Changed delivery decision')
        semantic=row['envelope']
        if semantic=='accepted':
            a=row['audit'];raw=a['response'].get('text','').strip()
            verdict=raw if raw in ['PASS','FAIL'] and not any(a['response'].get(k) for k in ['truncated','cancelled','applicationTimeout']) else 'INVALID'
            if a['verdict']!=verdict:raise ValueError('Changed audit verdict')
            semantic='accepted' if verdict=='PASS' else 'semantic_'+verdict.lower()
        if row['delivery']['semantic']!=semantic:raise ValueError('Changed semantic delivery')
    summary=study.metrics(samples,rows)
    if summary!=study.read(path/'summary.json'):raise ValueError('Scorer replay differs')
    return manifest,samples,rows,summary

def main():
    p=argparse.ArgumentParser();p.add_argument('--dev',type=pathlib.Path,required=True);p.add_argument('--test',type=pathlib.Path,required=True);p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args()
    study.check_freeze()
    dev,ds,dr,dm=load_run(a.dev,'dev');test,ts,tr,tm=load_run(a.test,'test')
    for field in ['modelFiles','binaries','build','freezeSha256']:
        if dev['provenance'][field]!=test['provenance'][field]:raise ValueError('Mismatched paired artifact: '+field)
    if test['selectionSha256']!=study.sha(a.dev/'selection.json'):raise ValueError('Test selection identity mismatch')
    selection=study.read(a.dev/'selection.json')
    if selection['devSummarySha256']!=study.sha(a.dev/'summary.json'):raise ValueError('Changed development selection')
    calibration=study.read(a.dev/'calibration.json');pairs={p['id']:p for p in study.read(HERE/'calibration.json')}
    if len(calibration)!=len(pairs) or {r['pair']['id'] for r in calibration}!=set(pairs):raise ValueError('Incomplete calibration')
    for row in calibration:
        if row['pair']!=pairs[row['pair']['id']]:raise ValueError('Changed calibration input')
    safe=sum(r['pair']['expected']=='PASS' and r['audit']['verdict']=='PASS' for r in calibration)
    harmful=sum(r['pair']['expected']=='FAIL' and r['audit']['verdict']=='PASS' for r in calibration)
    if safe!=selection['calibrationFaithfulAccepted'] or harmful!=selection['calibrationHarmfulAccepted']:raise ValueError('Changed calibration selection')
    out=a.out.resolve();out.mkdir(parents=True,exist_ok=False)
    result=dict(kind='synthetic-development-proxies',calibration=dict(faithful=12,faithfulAccepted=safe,harmful=12,harmfulAccepted=harmful),dev=dm,test=tm,selection=selection,productionChange=False,releaseAcceptance='pending independent human and native evaluation')
    gates={
        'completeGenerationCoverage':len(dr)==48 and len(tr)==48,
        'calibrationRejectsEveryHarmfulPair':harmful==0,
        'calibrationFaithfulAcceptanceAtLeast95Percent':safe/12>=.95,
        'devAtLeastFourAdditionalDeliveredConstraintLayoutPasses':dm['scene/semantic']['deliveredFactsAndLayoutPassed']-dm['production/strict']['deliveredFactsAndLayoutPassed']>=4,
        'testAtLeastFourAdditionalDeliveredConstraintLayoutPasses':tm['scene/semantic']['deliveredFactsAndLayoutPassed']-tm['production/strict']['deliveredFactsAndLayoutPassed']>=4,
        'devNoAdditionalAcceptedConstraintFailures':dm['scene/semantic']['acceptedConstraintFailures']<=dm['production/strict']['acceptedConstraintFailures'],
        'testNoAdditionalAcceptedConstraintFailures':tm['scene/semantic']['acceptedConstraintFailures']<=tm['production/strict']['acceptedConstraintFailures'],
        'devWarmFinalP95Under1500Ms':dm['scene/semantic']['finalP95Ms']<=1500,
        'testWarmFinalP95Under1500Ms':tm['scene/semantic']['finalP95Ms']<=1500,
    }
    result['diagnosticGates']=gates
    result['diagnosticAccepted']=all(gates.values())
    result['actualGenerationCalls']=len(dr)+len(tr)
    result['actualVerifierCalls']=len(calibration)+sum('audit' in r for r in dr+tr)
    study.write(out/'summary.json',result)
    rows=dr+tr;samples=ds+ts;byid={s['id']:s for s in samples}
    with (out/'cases.csv').open('x',newline='') as f:
        writer=csv.writer(f);writer.writerow(['split','id','generation','scene','language','strict','envelope','semantic','factsPassed','layoutPassed','generationMs','auditMs','source','proposal'])
        for row in rows:
            s=byid[row['id']];score=row['rawConstraintScore']
            writer.writerow([s['split'],s['id'],row['generationArm'],s['scene'],s['language'],row['strict'],row['envelope'],row['delivery']['semantic'],score['factsPassed'],score['layoutPassed'],row['generationMs'],row.get('audit',{}).get('wallMs',0),s['source'],row['response'].get('text','')])
    # Concrete blinded tasks, without claiming ratings or exposing model/arm identity.
    tasks=[];mapping=[]
    for row in rows:
        s=byid[row['id']]
        for policy in ['strict','envelope','semantic']:
            accepted=row['delivery'][policy]=='accepted';candidate=row['response']['text'] if accepted else row['response']['cleanup']['text']
            tasks.append(dict(source=s['source'],scene=s['scene'],candidate=candidate))
            mapping.append(dict(id=row['id'],generation=row['generationArm'],delivery=policy))
    order=list(range(len(tasks)));random.Random(1729).shuffle(order)
    private=[]
    with (out/'blind-tasks.jsonl').open('x') as f:
        for n,i in enumerate(order):
            taskid=hashlib.sha256(('scene-alignment-review-'+str(n)).encode()).hexdigest()[:16]
            f.write(json.dumps(dict(taskId=taskid,**tasks[i]),ensure_ascii=False)+'\n');private.append(dict(taskId=taskid,**mapping[i]))
    study.write(out/'mapping-private.json',private)
    study.write(out/'review-rubric.json',dict(status='unrated',minimumIndependentRaters=3,dimensions=['meaning 1-5','scene appropriateness 1-5','fluency 1-5','unsupported facts yes/no','lost requirements yes/no','language changed yes/no','answered or executed dictation yes/no'],note='Do not share mapping-private.json with raters. These synthetic tasks do not replace a reviewed natural-speech corpus.'))
    lines=['# Scene rewrite alignment experiment','', 'Synthetic text development study; automatic counts below are constraint/layout proxies, not semantic accuracy. No production change was adopted.','',f'Calibration: {safe}/12 faithful pairs accepted; {harmful}/12 harmful pairs incorrectly accepted.','']
    for split,summary in [('Development',dm),('Frozen test',tm)]:
        lines += ['## '+split,'','| Arm | Accepted | Delivered facts + layout | Accepted constraint failures | Rejected fact-passing outputs | Final p95 ms |','| --- | ---: | ---: | ---: | ---: | ---: |']
        for arm,m in summary.items():lines.append(f"| {arm} | {m['accepted']}/24 | {m['deliveredFactsAndLayoutPassed']}/24 | {m['acceptedConstraintFailures']} | {m['rejectedFactPassingCandidates']} | {m['finalP95Ms']:.0f} |")
        lines += ['']
    lines += ['## Decision','','The development-only decision is '+selection['decision']+'. The local verifier failed its faithful-rewrite gate. Relaxing to envelope-only validation is a diagnostic arm, not a safe adoption path. Test was not used for retuning.','', 'All generated and rejected outputs, timings, calibration, provenance and original failed observations are retained. Native speech, ASR contention, screen context, cold-load distributions and independent human review are open release gates.','', 'The fact patterns can miss added claims, actor/condition scope or answered questions. Inspect cases.csv and complete blind review before making semantic claims. Merely inserting headings or satisfying required words is insufficient.']
    lines += ['', '## Frozen diagnostic gates', '', '| Gate | Result |', '| --- | --- |']
    for name,value in gates.items():lines.append('| '+name+' | '+('PASS' if value else 'FAIL')+' |')
    (out/'report.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(result,indent=2))
if __name__=='__main__':main()
