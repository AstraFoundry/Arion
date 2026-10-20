"""Validate the failed development candidates and retain every reviewed result."""
import hashlib
import json
import pathlib
import run as study
HERE=study.HERE
ROOT=study.ROOT
sha=study.engine.sha
rowhash=lambda r:hashlib.sha256(json.dumps(r,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
def collect():
    study.check_freeze();directory=ROOT/'benchmarks/runs/rewrite-prompt-isolation-v1/dev'
    manifest=study.engine.read(directory/'manifest.json');review=study.engine.read(directory/'review.json');decision=study.engine.read(directory/'decision.json')
    if manifest['status']!='complete' or manifest['resultsSha256']!=sha(directory/'results.jsonl') or review['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed run/review')
    if manifest['freezeSha256']!=sha(HERE/'freeze.json') or decision['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed freeze')
    if decision['devResultsSha256']!=sha(directory/'results.jsonl') or decision['devReviewSha256']!=sha(directory/'review.json'):raise ValueError('Changed decision source')
    samples={s['id']:s for s in study.engine.read(HERE/'development.json')};expected={(a,i) for a in study.ARMS for i in samples}
    rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];judgments={(r['variant'],r['id']):r for r in review['reviews']}
    keys=[(r['variant'],r['id']) for r in rows]
    if set(keys)!=expected or len(keys)!=len(expected) or set(judgments)!=expected or len(review['reviews'])!=len(expected):raise ValueError('Incomplete generation/review')
    refs={r['id']:r for r in study.engine.read(HERE/'expectations.json')};metrics={}
    for row in rows:
        item=judgments[(row['variant'],row['id'])]
        if item['rowSha256']!=rowhash(row) or any(row[k]!=v for k,v in samples[row['id']].items()):raise ValueError('Changed source/output')
        if row['effectiveModelScene']!=row['scene'] or row['templateSha256']!=sha(HERE/f"templates/{row['variant']}-{row['scene']}.json"):raise ValueError('Changed scene/template')
        if any(type(item.get(k)) is not bool for k in ['proposalFaithful','proposalSceneUseful','cleanupFaithful','cleanupSceneUseful','repairSucceeded']):raise ValueError('Missing judgment')
        m=metrics.setdefault(row['variant'],dict(cases=0,faithful=0,sceneReady=0,requiredRepairs=0,repairs=0))
        m['cases']+=1;m['faithful']+=item['proposalFaithful'];m['sceneReady']+=item['proposalFaithful'] and item['proposalSceneUseful'];m['requiredRepairs']+=refs[row['id']]['mustRepair'];m['repairs']+=refs[row['id']]['mustRepair'] and item['repairSucceeded']
    return rows,judgments,dict(generations=len(rows),reviewedOutputs=len(judgments),metrics=metrics,decision=decision,productionActivated=False)
def main():
    rows,reviews,summary=collect();out=ROOT/'benchmarks/runs/rewrite-prompt-isolation-v1/report';out.mkdir(exist_ok=False)
    study.engine.write(out/'summary.json',summary)
    lines=['# Complete failed-candidate development review','','Assistant inspection of all raw outputs and cleanup; fixture-assigned scenes. No evaluation or production promotion.','']
    for r in rows:
        j=reviews[(r['variant'],r['id'])];lines += [f"## {r['variant']} / {r['id']}",'',f"Scene: {r['scene']} → {r['effectiveModelScene']} (fixture assigned); template: {r['response']['template']}",'','**Source**','','```text',r['source'],'```','','**Reference**','','```json',json.dumps(r['context'],ensure_ascii=False,indent=2),'```','','**Raw output**','','```text',r['response'].get('text',''),'```','',j['note'],'',f"Faithful: {j['proposalFaithful']}; scene useful: {j['proposalSceneUseful']}; strict: {r['strict']}",'','**Cleanup**','','```text',r['response']['cleanup']['text'],'```','',j['cleanupNote'],'']
    (out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
