"""Inspectable intended-message scoring and exact delivery-policy replay for OCR repair."""
import argparse
import hashlib
import importlib.util
import json
import math
import pathlib
import sys
import run as study
sys.path.insert(0,str(study.HERE.parent))
from scene_report import scene_metadata,scene_markdown
spec=importlib.util.spec_from_file_location('recorded_delivery_policies',study.HERE.parent/'rewrite-no-validator-v1/policies.py')
policy_module=importlib.util.module_from_spec(spec);spec.loader.exec_module(policy_module)
POLICIES=policy_module.POLICIES
delivery=policy_module.delivery
engine=study.engine
HERE=study.HERE

def rowhash(row):return hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()

def checked(directory,phase,selected=None):
    directory=pathlib.Path(directory);manifest=engine.read(directory/'manifest.json')
    if manifest['status']!='complete' or manifest['phase']!=phase:raise ValueError('Incomplete or wrong phase')
    if manifest['freezeSha256']!=engine.sha(HERE/'freeze.json') or manifest['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed experiment or observations')
    samples=engine.read(HERE/'development.json') if phase=='dev' else engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json')
    if phase=='evaluation' and selected not in ['aligned','staged']:raise ValueError('Missing frozen selection')
    variants=['aligned','staged'] if phase=='dev' else ['previous',selected]
    expected={(v,s['id']) for v in variants for s in samples};samples={s['id']:s for s in samples}
    rows=[json.loads(line) for line in (directory/'results.jsonl').read_text().splitlines()]
    keys=[(r['variant'],r['id']) for r in rows]
    if len(keys)!=len(expected) or set(keys)!=expected or len(set(keys))!=len(keys):raise ValueError('Missing or duplicate generation')
    review=engine.read(directory/'review.json')
    if review['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed reviewed results')
    reviews={}
    for item in review['reviews']:
        key=(item['variant'],item['id'])
        if key in reviews or key not in expected:raise ValueError('Duplicate or unknown review')
        if any(type(item[field]) is not bool for field in ['proposalFaithful','proposalSceneUseful','cleanupFaithful','cleanupSceneUseful','repairSucceeded']):raise ValueError('Missing review judgment')
        if not item['note'].strip() or not item['cleanupNote'].strip():raise ValueError('Missing inspection note')
        reviews[key]=item
    if set(reviews)!=expected:raise ValueError('Incomplete review')
    for row in rows:
        for key,value in samples[row['id']].items():
            if row[key]!=value:raise ValueError('Changed inference input')
        if row['effectiveModelScene']!=row['scene'] or row['templateSha256']!=engine.sha(study.template_path(row['variant'],row['scene'])):raise ValueError('Changed scene/template')
        if row['envelope']!=engine.envelope(row['response'],row['strict']):raise ValueError('Changed envelope verdict')
        if reviews[(row['variant'],row['id'])]['rowSha256']!=rowhash(row):raise ValueError('Changed reviewed output')
    return manifest,rows,reviews

def checked_parity(directory):
    directory=pathlib.Path(directory);manifest=engine.read(directory/'manifest.json');plan=engine.read(HERE/'parity-plan.json')
    if manifest['status']!='complete' or manifest['planSha256']!=engine.sha(HERE/'parity-plan.json') or manifest['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed or incomplete parity observations')
    if plan['runnerSha256']!=engine.sha(HERE/'parity_probe.py'):raise ValueError('Changed parity runner')
    rows=[json.loads(line) for line in (directory/'results.jsonl').read_text().splitlines()]
    keys=[(r['arm'],r['id']) for r in rows];expected={(a,id) for a in plan['arms'] for id in plan['caseIds']}
    if len(keys)!=len(expected) or set(keys)!=expected or len(set(keys))!=len(keys):raise ValueError('Missing or duplicate parity request')
    review=engine.read(directory/'review.json')
    if review['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed parity review binding')
    judgments={(r['arm'],r['id']):r for r in review['reviews']}
    if len(judgments)!=len(review['reviews']) or set(judgments)!=expected:raise ValueError('Incomplete parity review')
    samples={s['id']:s for s in engine.read(HERE/'regression.json')}
    for row in rows:
        if any(row[key]!=samples[row['id']][key] for key in ['source','context']):raise ValueError('Changed parity inference input')
        item=judgments[(row['arm'],row['id'])]
        if item['rowSha256']!=rowhash(row) or type(item['faithful']) is not bool or not item['note'].strip():raise ValueError('Changed parity reviewed output')
        if row['templateSha256']!=engine.sha(study.template_path('aligned','work')):raise ValueError('Changed parity template')
    mismatches=[]
    for id in plan['caseIds']:
        outputs={r['arm']:r['response'].get('text','') for r in rows if r['id']==id}
        if len(set(outputs.values()))>1:mismatches.append(dict(id=id,outputs=outputs))
    if manifest['mismatches']!=mismatches:raise ValueError('Incorrect parity comparison')
    return manifest,rows,judgments,mismatches

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--dev',type=pathlib.Path,required=True);parser.add_argument('--evaluation',type=pathlib.Path,required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--parity',type=pathlib.Path,required=True);args=parser.parse_args()
    study.check_freeze();selection=engine.read(args.dev/'selection.json')
    dm,dr,dv=checked(args.dev,'dev');em,er,ev=checked(args.evaluation,'evaluation',selection['selected'])
    for key in ['modelFiles','binaries','build']:
        if dm['provenance'][key]!=em['provenance'][key]:raise ValueError('Unpaired model/runtime/build')
    for path,key in [(args.dev/'results.jsonl','devResultsSha256'),(args.dev/'review.json','devReviewSha256')]:
        if selection[key]!=engine.sha(path):raise ValueError('Changed selected development evidence')
    if em['selectionSha256']!=engine.sha(args.dev/'selection.json'):raise ValueError('Changed development selection')
    pm,pr,pv,mismatches=checked_parity(args.parity)
    for key in ['modelFiles','binaries','build']:
        if pm['provenance'][key]!=em['provenance'][key]:raise ValueError('Unpaired parity assets')
    refs={r['id']:r for r in engine.read(HERE/'expectations.json')}
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    raw_summary={};delivered_summary={};records=[];scenes=[]
    md=['# Complete OCR alignment prompt review','','All outputs and cleanup values were inspected by the assistant. Authored text only; no independent human or native ASR/OCR acceptance is claimed.','']
    for phase,rows,reviews in [('development',dr,dv),('evaluation',er,ev)]:
        for row in rows:
            item=reviews[(row['variant'],row['id'])];ref=refs[row['id']];metadata=scene_metadata(row)
            scenes.append(dict(phase=phase,id=row['id'],variant=row['variant'],**metadata))
            key='/'.join([phase,row['cohort'],row['variant']]);stat=raw_summary.setdefault(key,dict(cases=0,faithful=0,sceneReady=0,requiredRepairs=0,repairSucceeded=0,harmfulContextChanges=0,omittedConstraints=0,demonstrationLeakage=0,correctTextDamaged=0,envelopeFailures=0))
            stat['cases']+=1;stat['faithful']+=item['proposalFaithful'];stat['sceneReady']+=item['proposalFaithful'] and item['proposalSceneUseful'] and row['envelope']=='accepted';stat['requiredRepairs']+=ref['mustRepair'];stat['repairSucceeded']+=ref['mustRepair'] and item['repairSucceeded'];stat['harmfulContextChanges']+=any(issue in item['issues'] for issue in ['context_instruction_followed','wrong_ocr_overwrite','ambiguous_context_guess','context_fact_leakage','context_replaced_dictation']);stat['demonstrationLeakage']+='demonstration_leakage' in item['issues'];stat['correctTextDamaged']+=row.get('kind')=='already-correct' and not item['proposalFaithful'];stat['omittedConstraints']+='omitted_constraint' in item['issues'];stat['envelopeFailures']+=row['envelope']!='accepted'
            md += [f"## {phase} / {row['variant']} / {row['id']}",'']+scene_markdown(metadata)
            md += ['**Evaluator-only intended message**','','```text',ref['intendedText'],'```','','**Source**','','```text',row['source'],'```','','**Context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw proposal**','','```text',row['response'].get('text',''),'```','',f"**Review:** faithful={item['proposalFaithful']}; sceneUseful={item['proposalSceneUseful']}; repairSucceeded={item['repairSucceeded']}; strict={row['strict']}; envelope={row['envelope']}. {item['note']}",'']
            for policy in POLICIES:
                result=delivery(row,policy);origin=result['origin']
                faithful=item['proposalFaithful'] if origin=='proposal' else item['cleanupFaithful'] if origin=='cleanup' else False
                useful=item['proposalSceneUseful'] if origin=='proposal' else item['cleanupSceneUseful'] if origin=='cleanup' else False
                valid=row['envelope']=='accepted' if origin=='proposal' else origin=='cleanup'
                repaired=ref['mustRepair'] and origin=='proposal' and item['repairSucceeded']
                record=dict(phase=phase,id=row['id'],variant=row['variant'],cohort=row['cohort'],policy=policy,**metadata,**result,faithful=faithful,sceneReady=faithful and useful and valid,repairSucceeded=repaired,issues=item['issues'],note=item['note'] if origin=='proposal' else item['cleanupNote'])
                records.append(record);dkey=key+'/'+policy;d=delivered_summary.setdefault(dkey,dict(cases=0,proposalsDelivered=0,faithfulDelivered=0,sceneReadyDelivered=0,repairSucceeded=0,invalidProposalsDelivered=0,faithfulProposalsRejected=0))
                d['cases']+=1;d['proposalsDelivered']+=origin=='proposal';d['faithfulDelivered']+=faithful;d['sceneReadyDelivered']+=record['sceneReady'];d['repairSucceeded']+=repaired;d['invalidProposalsDelivered']+=origin=='proposal' and not valid;d['faithfulProposalsRejected']+=origin!='proposal' and item['proposalFaithful'] and row['envelope']=='accepted'
                md += [f"**Delivered with {policy}:** origin={origin}; outcome={result['outcome']}; faithful={faithful}; sceneReady={record['sceneReady']}",'','```text',result['text'] if result['text'] is not None else '[no delivery]','```','',record['note'],'']
    timing={}
    for variant in sorted({r['variant'] for r in er}):
        values=sorted(r['generationMs'] for r in er if r['variant']==variant)
        timing[variant]=dict(p50Ms=values[math.ceil(.5*len(values))-1],p95Ms=values[math.ceil(.95*len(values))-1])
    development_by_key={(r['variant'],r['id']):r for r in dr}
    evaluation_by_key={(r['variant'],r['id']):r for r in er}
    shared=sorted(set(development_by_key)&set(evaluation_by_key));shared_mismatches=[]
    for key in shared:
        before=development_by_key[key];after=evaluation_by_key[key]
        if any(before[field]!=after[field] for field in ['source','context','scene','templateSha256']):raise ValueError('Nonidentical input in cross-phase comparison')
        if before['response'].get('text','')!=after['response'].get('text',''):shared_mismatches.append(dict(variant=key[0],id=key[1]))
    result=dict(crossPhaseIdenticalInputs=len(shared),crossPhaseOutputMismatches=shared_mismatches,parityGenerations=len(pr),parityOutputMismatches=mismatches,generations=len(dr)+len(er),reviewedGenerations=len(dv)+len(ev),policyDeliveries=len(records),selected=selection['selected'],productionActivated=False,rawSummary=raw_summary,deliveredSummary=delivered_summary,timing=timing,limitations=['Authored text, not executed ASR or screenshot OCR; scenes are fixture-assigned.','Single-assistant review, not independent blinded ratings.','Development and regression overlap explicitly; new cases share known challenge types.','Simultaneous policy/example edits versus old prompt are not an isolated causal attribution.','None bypasses text validation, not runtime/session/deadline protection; no native lifecycle test.'])
    engine.write(out/'summary.json',result);engine.write(out/'scenes.json',scenes)
    with (out/'deliveries.jsonl').open('x') as stream:
        for record in records:stream.write(json.dumps(record,ensure_ascii=False)+'\n')
    (out/'review.md').write_text('\n'.join(md)+'\n')
    raw={(r['variant'],r['id']):r for r in er};ungated=['# Every evaluation delivery without text validation','','Local text-only outputs; no independent ratings or native insertion.','']
    for r in records:
        if r['phase']!='evaluation' or r['policy']!='none':continue
        row=raw[(r['variant'],r['id'])]
        ungated += [f"## {r['variant']} / {r['id']}",'']+scene_markdown(r)+['**Source**','','```text',row['source'],'```','','**Context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Delivered**','','```text',r['text'] if r['text'] is not None else '[no delivery]','```','',f"**Review:** faithful={r['faithful']}; sceneReady={r['sceneReady']}; repairSucceeded={r['repairSucceeded']}. {r['note']}",'']
    (out/'without-validator.md').write_text('\n'.join(ungated)+'\n')
    parity_md=['# Cold and request-order parity diagnostic','','Same aligned work template, same four inputs; the cold, forward and reverse arms have '+str(len(mismatches))+' output mismatches. This bounded check does not establish all runtime parity.','']
    for row in pr:
        item=pv[(row['arm'],row['id'])]
        parity_md += [f"## {row['arm']} / {row['id']}",'','**Source**','','```text',row['source'],'```','','**Context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Output**','','```text',row['response'].get('text',''),'```','',item['note'],'']
    (out/'parity.md').write_text('\n'.join(parity_md)+'\n');print(json.dumps(result,indent=2))
if __name__=='__main__':main()
