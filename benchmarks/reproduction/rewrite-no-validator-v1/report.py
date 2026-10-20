"""Check complete proposal/cleanup review and export every text-policy delivery."""
import argparse
import hashlib
import json
import math
import pathlib
import sys
import run as study
from policies import POLICIES,delivery
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent))
from scene_report import scene_metadata,scene_markdown
HERE=study.HERE
engine=study.engine

def rowhash(row):return hashlib.sha256(json.dumps(row,ensure_ascii=False,sort_keys=True).encode()).hexdigest()

def checked(directory,phase,selected_variant=None):
    directory=pathlib.Path(directory);manifest=engine.read(directory/'manifest.json')
    if manifest['status']!='complete' or manifest['phase']!=phase:raise ValueError('Incomplete or wrong-phase run')
    if manifest['freezeSha256']!=engine.sha(HERE/'freeze.json') or manifest['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed experiment or raw results')
    samples=engine.read(HERE/'development.json') if phase=='dev' else engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json')
    if phase=='evaluation' and selected_variant not in ['minimal','anchored']:raise ValueError('Missing frozen selected variant')
    variants=['minimal','anchored'] if phase=='dev' else ['previous',selected_variant]
    expected={(v,s['id']) for v in variants for s in samples};samples={s['id']:s for s in samples}
    rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()]
    keys=[(r['variant'],r['id']) for r in rows]
    if len(keys)!=len(expected) or set(keys)!=expected or len(set(keys))!=len(keys):raise ValueError('Missing or duplicate generation')
    review=engine.read(directory/'review.json')
    if review['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Review binds different results')
    reviews={}
    for item in review['reviews']:
        key=(item['variant'],item['id'])
        if key not in expected or key in reviews:raise ValueError('Unknown or duplicate review')
        for field in ['proposalFaithful','proposalSceneUseful','cleanupFaithful','cleanupSceneUseful']:
            if type(item[field]) is not bool:raise ValueError('Missing judgment')
        if not item['note'].strip() or not item['cleanupNote'].strip():raise ValueError('Missing inspection notes')
        reviews[key]=item
    if set(reviews)!=expected:raise ValueError('Incomplete review')
    for row in rows:
        for key,value in samples[row['id']].items():
            if row[key]!=value:raise ValueError('Changed inference input')
        if row['effectiveModelScene']!=row['scene'] or row['templateSha256']!=engine.sha(study.template_path(row['variant'],row['scene'])):raise ValueError('Changed scene or template')
        if row['envelope']!=engine.envelope(row['response'],row['strict']):raise ValueError('Changed envelope verdict')
        if reviews[(row['variant'],row['id'])]['rowSha256']!=rowhash(row):raise ValueError('Changed reviewed source/output')
    return manifest,rows,reviews

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--dev',type=pathlib.Path,required=True);parser.add_argument('--evaluation',type=pathlib.Path,required=True);parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args()
    study.check_freeze()
    selection=engine.read(args.dev/'selection.json')
    dm,dr,dv=checked(args.dev,'dev');em,er,ev=checked(args.evaluation,'evaluation',selection['selected'])
    for key in ['modelFiles','binaries','build']:
        if dm['provenance'][key]!=em['provenance'][key]:raise ValueError('Unpaired model/build')
    selection=engine.read(args.dev/'selection.json')
    if em['selectionSha256']!=engine.sha(args.dev/'selection.json') or selection['devResultsSha256']!=engine.sha(args.dev/'results.jsonl') or selection['devReviewSha256']!=engine.sha(args.dev/'review.json'):raise ValueError('Changed frozen selection')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    references={r['id']:r for r in engine.read(HERE/'expectations.json')}
    records=[];summary={};scenes=[];md=['# Complete prompt and validator ablation review','','All outputs and completed cleanup were inspected by the assistant. No independent human ratings or native insertion are claimed.','']
    for phase,rows,reviews in [('development',dr,dv),('evaluation',er,ev)]:
        for row in rows:
            review=reviews[(row['variant'],row['id'])];metadata=scene_metadata(row)
            scenes.append(dict(phase=phase,variant=row['variant'],id=row['id'],**metadata))
            md += [f"## {phase} / {row['variant']} / {row['id']}",'']+scene_markdown(metadata)
            md += ['**Evaluator-only intended message**','','```text',references[row['id']]['intendedText'],'```','','**Input**','','```text',row['source'],'```','','**Context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw proposal**','','```text',row['response'].get('text',''),'```','',f"**Proposal review:** faithful={review['proposalFaithful']}; sceneUseful={review['proposalSceneUseful']}; strict={row['strict']}; envelope={row['envelope']}. {review['note']}",'']
            for policy in POLICIES:
                result=delivery(row,policy);origin=result['origin']
                faithful=review['proposalFaithful'] if origin=='proposal' else review['cleanupFaithful'] if origin=='cleanup' else False
                useful=review['proposalSceneUseful'] if origin=='proposal' else review['cleanupSceneUseful'] if origin=='cleanup' else False
                envelope_valid=row['envelope']=='accepted' if origin=='proposal' else origin=='cleanup'
                record=dict(phase=phase,cohort=row['cohort'],id=row['id'],variant=row['variant'],policy=policy,**metadata,**result,faithful=faithful,sceneReady=faithful and useful and envelope_valid,invalidProposalDelivered=origin=='proposal' and not envelope_valid,missedRequiredRepair=not faithful and references[row['id']]['mustRepair'] and (origin=='cleanup' or 'missed_supported_correction' in review['issues']),proposalIssues=review['issues'],review=review['note'] if origin=='proposal' else review['cleanupNote'])
                records.append(record)
                key='/'.join([phase,row['cohort'],row['variant'],policy]);stat=summary.setdefault(key,dict(cases=0,proposalsDelivered=0,cleanupDelivered=0,suppressed=0,faithfulDelivered=0,sceneReadyDelivered=0,unfaithfulDelivered=0,invalidProposalsDelivered=0,missedRequiredRepairs=0,faithfulProposalsRejected=0))
                stat['cases']+=1;stat['proposalsDelivered']+=origin=='proposal';stat['cleanupDelivered']+=origin=='cleanup';stat['suppressed']+=origin=='suppressed';stat['faithfulDelivered']+=faithful;stat['sceneReadyDelivered']+=record['sceneReady'];stat['unfaithfulDelivered']+=not faithful and origin!='suppressed';stat['invalidProposalsDelivered']+=record['invalidProposalDelivered'];stat['missedRequiredRepairs']+=record['missedRequiredRepair'];stat['faithfulProposalsRejected']+=not result['accepted'] and review['proposalFaithful'] and row['envelope']=='accepted'
                md += [f"**Delivered with {policy}:** origin={origin}; outcome={result['outcome']}; faithful={faithful}; sceneReady={record['sceneReady']}",'','```text',result['text'] if result['text'] is not None else '[no delivery]','```','',record['review'],'']
    timings={}
    for variant in sorted({r['variant'] for r in er}):
        values=[r for r in er if r['variant']==variant]
        timings[variant]={}
        for field in ['generationMs','strictCheckMs']:
            durations=sorted(r[field] for r in values)
            timings[variant][field]=dict(p50=durations[math.ceil(.5*len(durations))-1],p95=durations[math.ceil(.95*len(durations))-1])
    fresh_records={(r['variant'],r['id'],r['policy']):r for r in records if r['phase']=='evaluation' and r['cohort']=='fresh-test'}
    changes=[]
    for variant,id in sorted({(r['variant'],r['id']) for r in fresh_records.values()}):
        before=fresh_records[(variant,id,'strict')];after=fresh_records[(variant,id,'none')]
        if before['origin']!=after['origin']:
            changes.append(dict(variant=variant,id=id,beforeOrigin=before['origin'],afterOrigin=after['origin'],beforeFaithful=before['faithful'],afterFaithful=after['faithful'],beforeSceneReady=before['sceneReady'],afterSceneReady=after['sceneReady'],invalidProposalDelivered=after['invalidProposalDelivered'],issues=after['proposalIssues']))
    report=dict(strictToNoneChanges=changes,timings=timings,generations=len(dr)+len(er),reviewedGenerations=len(dv)+len(ev),policyDeliveries=len(records),selected=selection['selected'],productionActivated=False,summary=summary,limitations=['Authored Chinese ASR-like/OCR-like text; no executed recognition/OCR or native app-scene matching.','Single-assistant judgments; fresh cases include known semantic challenge types and are not independent new speakers/families.','None bypasses every text gate but cannot override runtime cancellation/deadline/session ownership.','Previous is the frozen prior-candidate composite, not installed production.','StrictCheckMs includes a diagnostic Go subprocess and is not production validator overhead.'])
    engine.write(out/'summary.json',report);engine.write(out/'scenes.json',scenes)
    with (out/'deliveries.jsonl').open('x') as stream:
        for record in records:stream.write(json.dumps(record,ensure_ascii=False)+'\n')
    (out/'review.md').write_text('\n'.join(md)+'\n')
    indexed={(r['variant'],r['id']):r for r in er}
    ungated=['# Complete evaluation deliveries without a text validator','','All 96 evaluation outputs below bypass text validation. Runtime cancellation/deadline protection remains. All judgments are assistant review, not independent ratings.','']
    for record in records:
        if record['phase']!='evaluation' or record['policy']!='none':continue
        row=indexed[(record['variant'],record['id'])]
        ungated += [f"## {record['variant']} / {record['id']}",'']+scene_markdown(record)
        ungated += ['**Source**','','```text',row['source'],'```','','**Context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Delivered without text validation**','','```text',record['text'] if record['text'] is not None else '[no delivery]','```','',f"**Review:** faithful={record['faithful']}; sceneReady={record['sceneReady']}; origin={record['origin']}; original strict={row['strict']}. {record['review']}",'']
    (out/'without-validator.md').write_text('\n'.join(ungated)+'\n');print(json.dumps(report,indent=2))
if __name__=='__main__':main()
