"""Replay the full entity-policy and actual-Vision diagnostic without inference."""
import argparse,hashlib,json,pathlib
import run as study
ROOT=study.ROOT
HERE=study.HERE
RUN=ROOT/'benchmarks/runs/rewrite-entity-evidence-v1'
engine=study.engine
rh=lambda r:hashlib.sha256(json.dumps(r,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
ENTITIES={'ocr-email-name':'林语桐','ocr-work-code':'loadCache','ocr-work-term':'回归测试','ocr-prompt-code':'getProfile','ocr-chat-name':'程语楠'}
def check_frozen():
    frozen=engine.read(HERE/'freeze.json');mapping=engine.read(RUN/'path-mappings.json')
    for name,digest in frozen['files'].items():
        if name.startswith('.workspace/'):
            if mapping['externalArtifacts'].get(name)!=digest:raise ValueError('Changed external artifact identity')
        elif engine.sha(ROOT/mapping['sourceSnapshots'].get(name,name))!=digest:raise ValueError('Changed archived input: '+name)
    ocr=engine.read(RUN/'ocr/manifest.json')
    if ocr['status']!='complete' or ocr['observations']!=24 or ocr['observationsSha256']!=engine.sha(RUN/'ocr/observations.jsonl'):raise ValueError('Incomplete/changed Vision observations')
    if ocr['provenance']['binarySha256']!=mapping['externalArtifacts']['.workspace/rewrite-entity-evidence-v1/arion-ocr'] or ocr['provenance']['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('Unpinned OCR build')
    return frozen

def checked(directory,phase):
    frozen=check_frozen();manifest=engine.read(directory/'manifest.json');review=engine.read(directory/'review.json')
    if manifest['status']!='complete' or manifest['phase']!=phase or manifest['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Incomplete or changed generation')
    if manifest['freezeSha256']!=engine.sha(HERE/'freeze.json') or review['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed freeze/review')
    if manifest['provenance']['packagingBuild']!=frozen['build'] or manifest['provenance']['packagingWorkerSha256']!=frozen['build'][str((study.packaging.BUILD/'worker').relative_to(ROOT))]:raise ValueError('Changed common worker')
    if phase=='ocr' and manifest['ocrSha256']!=engine.sha(RUN/'ocr/observations.jsonl'):raise ValueError('Wrong OCR source')
    samples=study.samples_for(phase,RUN/'ocr/observations.jsonl' if phase=='ocr' else None);samples={r['id']:r for r in samples};expected={(v,i) for v in study.ARMS for i in samples}
    rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];keys=[(r['variant'],r['id']) for r in rows];judgments={(r['variant'],r['id']):r for r in review['reviews']}
    if len(keys)!=len(expected) or set(keys)!=expected or len(review['reviews'])!=len(expected) or set(judgments)!=expected:raise ValueError('Missing/duplicate observation or review')
    for row in rows:
        item=judgments[(row['variant'],row['id'])]
        if any(row[k]!=v for k,v in samples[row['id']].items()):raise ValueError('Changed source or actual OCR context')
        template=HERE/f"templates/{row['variant']}-{row['scene']}.json"
        if row['templateSha256']!=engine.sha(template) or row['effectiveModelScene']!=row['scene']:raise ValueError('Changed template/scene')
        if item['rowSha256']!=rh(row):raise ValueError('Changed reviewed output')
        if any(type(item.get(k)) is not bool for k in ['proposalFaithful','proposalSceneUseful','cleanupFaithful','cleanupSceneUseful','repairSucceeded']):raise ValueError('Missing judgment')
    return manifest,rows,judgments

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();refs={r['id']:r for r in engine.read(HERE/'expectations.json')};metrics={};md=['# Complete entity-policy / native Vision review','','Actual Vision OCR on authored image fixtures, then native local LLM inference. Assistant inspection, not independent ratings. Inputs are exposed development probes, not held-out natural ASR. The exact 程雨南→程语楠 alias appears in a demonstration; correcting it with empty OCR cannot establish contextual benefit.',''];manifests=[]
    for phase,relative in [('dev','dev'),('ocr','vision')]:
        manifest,rows,reviews=checked(RUN/relative,phase);manifests.append(manifest)
        for row in rows:
            j=reviews[(row['variant'],row['id'])];base=row.get('ocrCaseId',row['id']);ref=refs[base];key='/'.join([phase,row['cohort'],row['variant']]);m=metrics.setdefault(key,dict(cases=0,faithful=0,sceneReady=0,requiredGoldRepairs=0,goldRepairs=0,correctSourceControls=0,correctSourcePreserved=0,contextReplacements=0,unsupportedDemonstrationAlias=0,strictFaithfulDelivered=0))
            m['cases']+=1;m['faithful']+=j['proposalFaithful'];m['sceneReady']+=j['proposalFaithful'] and j['proposalSceneUseful'];m['requiredGoldRepairs']+=ref['mustRepair'];m['goldRepairs']+=ref['mustRepair'] and j['repairSucceeded'];m['correctSourceControls']+=row['cohort']=='correct-source-control';m['correctSourcePreserved']+=row['cohort']=='correct-source-control' and j['proposalFaithful'];m['contextReplacements']+='context_replaced_dictation' in j['issues'];m['unsupportedDemonstrationAlias']+='demonstration_alias_without_context_evidence' in j['issues'];accepted=row['strict']=='accepted' and row['envelope']=='accepted';m['strictFaithfulDelivered']+=j['proposalFaithful'] if accepted else j['cleanupFaithful']
            supported=ENTITIES.get(base) in row['context'].get('Screen','') if base in ENTITIES else None
            md += [f"## {phase} / {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']} (fixture assigned); template: {row['response']['template']}",'','**Source**','','```text',row['source'],'```','','**Complete reference**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw output**','','```text',row['response']['text'],'```','',j['note'],'',f"Faithful: {j['proposalFaithful']}; scene ready: {j['proposalSceneUseful']}; required spelling present in OCR: {supported}; strict: {row['strict']}",'','**Strict delivery**','','```text',row['response']['text'] if accepted else row['response']['cleanup']['text'],'```','',j['note'] if accepted else j['cleanupNote'],'']
    if manifests[0]['provenance']['modelFiles']!=manifests[1]['provenance']['modelFiles']:raise ValueError('Different model assets')
    decision=engine.read(RUN/'dev/decision.json')
    if decision['devResultsSha256']!=engine.sha(RUN/'dev/results.jsonl') or decision['devReviewSha256']!=engine.sha(RUN/'dev/review.json'):raise ValueError('Changed decision')
    summary=dict(generations=100,reviewedOutputs=100,actualVisionObservations=24,metrics=metrics,productionActivated=False,decision=decision,limitations=['Authored and exposed text/images, not natural or held-out generalization.','Same example alias can be reproduced without OCR; gold repair is not causal OCR benefit.','Python host reports Rosetta architecture; archived host evidence identifies Apple M4 and arm64 native executables.','No acoustic ASR, live window capture, native insertion or independent review acceptance.','Native OCR overlapped initial development model loading; request timings are not isolated release latency evidence.'])
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);engine.write(out/'summary.json',summary);(out/'review.md').write_text('\n'.join(md)+'\n');print(json.dumps(summary,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
