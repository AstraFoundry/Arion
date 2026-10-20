"""Validate and replay the packaging study with complete proposal and cleanup review."""
import argparse
import hashlib
import json
import math
import pathlib
import sys
import run as study
sys.path.insert(0,str(study.HERE.parent))
from scene_report import scene_metadata,scene_markdown
engine=study.engine
HERE=study.HERE

def rowhash(row):return hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()

def checked(directory,phase,selected=None):
    directory=pathlib.Path(directory);manifest=engine.read(directory/'manifest.json')
    if manifest['status']!='complete' or manifest['phase']!=phase:raise ValueError('Incomplete or wrong phase')
    if manifest['freezeSha256']!=engine.sha(HERE/'freeze.json') or manifest['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed frozen/raw data')
    frozen=engine.read(HERE/'freeze.json')
    if manifest['provenance']['packagingBuild']!=frozen['build']:raise ValueError('Unpinned common build')
    if manifest['provenance']['packagingWorkerSha256']!=frozen['build'][str((study.BUILD/'worker').relative_to(study.ROOT))]:raise ValueError('Unpinned worker binary')
    samples=engine.read(HERE/'development.json') if phase=='dev' else engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json')
    variants=list(study.FORMATS) if phase=='dev' else ['xml',selected]
    expected={(v,s['id']) for v in variants for s in samples};samples={s['id']:s for s in samples}
    rows=[json.loads(line) for line in (directory/'results.jsonl').read_text().splitlines()]
    keys=[(r['variant'],r['id']) for r in rows]
    if len(keys)!=len(expected) or set(keys)!=expected or len(set(keys))!=len(keys):raise ValueError('Missing/duplicate generation')
    review=engine.read(directory/'review.json')
    if review['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed review binding')
    indexed={}
    for item in review['reviews']:
        key=(item['variant'],item['id'])
        if key in indexed or key not in expected:raise ValueError('Duplicate/unknown review')
        if any(type(item[f]) is not bool for f in ['proposalFaithful','proposalSceneUseful','cleanupFaithful','cleanupSceneUseful','repairSucceeded']):raise ValueError('Missing review judgment')
        if not item['note'].strip() or not item['cleanupNote'].strip():raise ValueError('Missing inspection note')
        indexed[key]=item
    if set(indexed)!=expected:raise ValueError('Incomplete review')
    for row in rows:
        if any(row[k]!=v for k,v in samples[row['id']].items()):raise ValueError('Changed request')
        if row['effectiveModelScene']!=row['scene'] or row['templateSha256']!=engine.sha(HERE/f"templates/{row['scene']}.json"):raise ValueError('Changed scene/template')
        if indexed[(row['variant'],row['id'])]['rowSha256']!=rowhash(row):raise ValueError('Changed reviewed output')
    return manifest,rows,indexed

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--dev',type=pathlib.Path,required=True);parser.add_argument('--evaluation',type=pathlib.Path,required=True);parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();study.check_freeze()
    selection=engine.read(args.dev/'selection.json');dm,dr,dv=checked(args.dev,'dev');em,er,ev=checked(args.evaluation,'evaluation',selection['selected'])
    if em['selectionSha256']!=engine.sha(args.dev/'selection.json') or selection['devResultsSha256']!=engine.sha(args.dev/'results.jsonl') or selection['devReviewSha256']!=engine.sha(args.dev/'review.json'):raise ValueError('Changed frozen selection')
    for key in ['modelFiles','packagingBuild','packagingWorkerSha256']:
        if dm['provenance'][key]!=em['provenance'][key]:raise ValueError('Unpaired model/worker')
    refs={r['id']:r for r in engine.read(HERE/'expectations.json')};summary={};scenes=[];md=['# Complete packaging comparison review','','Same model, template and absolute decoding budget; different input serialization. All judgments are assistant inspection, not independent human ratings.','']
    for phase,rows,reviews in [('development',dr,dv),('evaluation',er,ev)]:
        for row in rows:
            item=reviews[(row['variant'],row['id'])];ref=refs[row['id']];meta=scene_metadata(row);scenes.append(dict(phase=phase,id=row['id'],variant=row['variant'],**meta))
            key='/'.join([phase,row['cohort'],row['variant']]);s=summary.setdefault(key,dict(cases=0,rawFaithful=0,rawSceneReady=0,requiredRepairs=0,repairSucceeded=0,harmfulContextChanges=0,demonstrationLeakage=0,strictFaithfulDelivered=0,strictSceneReadyDelivered=0,faithfulProposalsRejected=0))
            accepted=row['strict']=='accepted' and row['envelope']=='accepted';faithful=item['proposalFaithful'];useful=item['proposalSceneUseful'];delivered=row['response'].get('text','') if accepted else row['response']['cleanup']['text'];df=faithful if accepted else item['cleanupFaithful'];du=useful if accepted else item['cleanupSceneUseful']
            s['cases']+=1;s['rawFaithful']+=faithful;s['rawSceneReady']+=faithful and useful and row['envelope']=='accepted';s['requiredRepairs']+=ref['mustRepair'];s['repairSucceeded']+=ref['mustRepair'] and item['repairSucceeded'];s['harmfulContextChanges']+=any(i in item['issues'] for i in ['context_replaced_dictation','context_instruction_followed','context_fact_leakage','ambiguous_context_guess']);s['demonstrationLeakage']+='demonstration_leakage' in item['issues'];s['strictFaithfulDelivered']+=df;s['strictSceneReadyDelivered']+=df and du;s['faithfulProposalsRejected']+=faithful and not accepted
            md += [f"## {phase} / {row['variant']} / {row['id']}",'']+scene_markdown(meta)+['**Source**','','```text',row['source'],'```','','**Reference**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw output**','','```text',row['response'].get('text',''),'```','',f"**Review:** faithful={faithful}; sceneUseful={useful}; repairSucceeded={item['repairSucceeded']}; strict={row['strict']}. {item['note']}",'','**Strict delivery**','','```text',delivered,'```','',item['note'] if accepted else item['cleanupNote'],'']
    timings={}
    for variant in ['xml',selection['selected']]:
        values=sorted(r['generationMs'] for r in er if r['variant']==variant);timings[variant]=dict(p50Ms=values[math.ceil(.5*len(values))-1],p95Ms=values[math.ceil(.95*len(values))-1])
    result=dict(generations=len(dr)+len(er),reviewedOutputs=len(dv)+len(ev),selected=selection['selected'],summary=summary,timing=timings,productionActivated=False,limitations=['Authored Chinese text; no real ASR/OCR/native app-scene test.','Shared-host timing with an experimental absolute budget; not a production latency comparison.','Assistant inspection, not independent human ratings.'])
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);engine.write(out/'summary.json',result);engine.write(out/'scenes.json',scenes);(out/'review.md').write_text('\n'.join(md)+'\n');print(json.dumps(result,indent=2))
if __name__=='__main__':main()
