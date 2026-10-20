"""Validate and replay the bounded checkpoint/export screen without inference."""
import argparse,hashlib,json,math,pathlib
import run as study
import sys
sys.path.insert(0,str(study.HERE.parent))
import review_evidence as evidence
ROOT=study.ROOT;HERE=study.HERE;RUN=ROOT/'benchmarks/runs/rewrite-model-screen-v1';engine=study.engine
rh=lambda r:hashlib.sha256(json.dumps(r,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
def checked(directory):
    frozen=engine.read(HERE/'freeze.json');mapping=engine.read(RUN/'path-mappings.json')
    for name,digest in frozen['files'].items():
        if engine.sha(ROOT/mapping.get(name,name))!=digest:raise ValueError('Changed frozen input: '+name)
    manifest=engine.read(directory/'manifest.json');review=engine.read(directory/'review.json')
    if manifest['status']!='complete' or manifest['generations']!=16 or manifest['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Incomplete/changed generation')
    if manifest['freezeSha256']!=engine.sha(HERE/'freeze.json') or review['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed freeze/review')
    if manifest['provenance']['modelFiles']!=frozen['modelFiles'] or manifest['provenance']['packagingBuild']!=frozen['build']:raise ValueError('Wrong model/build')
    worker=str(study.packaging.BUILD/'worker')
    if manifest['provenance']['packagingWorkerSha256']!=frozen['runtimeFiles'][worker]:raise ValueError('Wrong native worker')
    for name,digest in frozen['runtimeFiles'].items():
        if name!=worker and manifest['provenance']['binaries'].get(name)!=digest:raise ValueError('Wrong runtime/validator')
    return evidence.checked_rows(directory,engine.read(HERE/'cases.json'),study.VARIANTS,lambda row:HERE/f"templates/{row['scene']}.json",lambda row:frozen['modelFiles'][row['variant']])

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();rows,reviews=checked(RUN/'development');refs={r['id']:r for r in engine.read(HERE/'expectations.json')};summary={};lines=['# Complete checkpoint/export screen review','','Same native worker, package, templates and budgets; different checkpoint/export bundles. Every proposal and cleanup personally inspected by assistant. Eight exposed probes, not independent review or release latency evidence.','']
    for row in rows:
        item=reviews[(row['variant'],row['id'])];m=summary.setdefault(row['variant'],dict(cases=0,rawFaithful=0,rawSceneReady=0,requiredRepairs=0,repairs=0,strictFaithfulDelivered=0));ref=refs[row['id']];accepted=row['strict']=='accepted' and row['envelope']=='accepted'
        m['cases']+=1;m['rawFaithful']+=item['proposalFaithful'];m['rawSceneReady']+=item['proposalFaithful'] and item['proposalSceneUseful'];m['requiredRepairs']+=ref['mustRepair'];m['repairs']+=ref['mustRepair'] and item['repairSucceeded'];m['strictFaithfulDelivered']+=item['proposalFaithful'] if accepted else item['cleanupFaithful']
        lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']} (fixture assigned); template: {row['response']['template']}",'','**Source**','','```text',row['source'],'```','','**Full reference**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw output**','','```text',row['response']['text'],'```','',item['note'],'',f"Faithful: {item['proposalFaithful']}; scene ready: {item['proposalSceneUseful']}; strict: {row['strict']}",'','**Strict delivery**','','```text',row['response']['text'] if accepted else row['response']['cleanup']['text'],'```','',item['note'] if accepted else item['cleanupNote'],'']
    for variant,m in summary.items():
        data=[r for r in rows if r['variant']==variant]
        def quantile(values,p):return sorted(values)[math.ceil(p*len(values))-1]
        m.update(requestP50Ms=quantile([r['generationMs'] for r in data],.5),requestP95Ms=quantile([r['generationMs'] for r in data],.95),prefillP50Ms=quantile([r['response']['prefillMs'] for r in data],.5),decodeP50Ms=quantile([r['response']['decodeMs'] for r in data],.5),peakWorkerRssBytes=max(r['response']['peakRssBytes'] for r in data))
    result=dict(generations=16,reviewedOutputs=16,summary=summary,productionActivated=False,decision=engine.read(RUN/'development/decision.json'),scope='Different checkpoints and mixed export configurations; not a size-only or quantization-only causal comparison. Timing is weight-resident request time with a precomputed fixed prefix, not stop-to-insertion. Per-worker lifetime RSS excludes other processes/device counters.')
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False);engine.write(out/'summary.json',result);(out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
