"""Review actual F16 choices and native probes against prior same-request Q4."""
import argparse,importlib.util,json,pathlib,sys,types
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
RUN=ROOT/'benchmarks/runs/rewrite-selector-minicpm-f16-v1/run'
PREFLIGHT=RUN.parent/'preflight'
def module(name,path):
    spec=importlib.util.spec_from_file_location(name,path);value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value
before=list(sys.path);sys.path.insert(0,str(HERE.parent))
try:
    prior=module('minicpm_f16_prior_report',HERE.parent/'rewrite-selector-model-cpu-v1/report.py')
    shared=module('minicpm_f16_shared_model',HERE.parent/'gguf_model_screen_evidence.py')
    native=module('minicpm_f16_native',HERE.parent/'gguf_native_probe_evidence_v2.py')
finally:sys.path[:]=before
read,sha,rowhash,assessment=prior.read,prior.sha,prior.rowhash,prior.assessment
ARMS=['baseline','balanced'];prior.HERE,prior.RUN=HERE,RUN;base_checked=prior.checked
policy=types.SimpleNamespace(read=read,sha=sha,rowhash=rowhash,root=ROOT,here=HERE,arms=ARMS,pipeline=prior.pipeline,ownership=prior.ownership,assessment=assessment,template_path=HERE/'sources/native-template-policy.json',gpu_layers=99,offload_line='offloaded 25/25 layers to GPU')
def validate(rows,manifest,review,outcomes,decision,directory=RUN):
    return shared.validate(rows,manifest,review,outcomes,decision,directory,policy=policy)
prior.validate=validate

def runtime_row(row,ref):
    return dict(id=row['id'],variant=('metal99-' if row['placement']==99 else 'cpu-layer0-')+row['job'],cohort='native-runtime-compatibility-probe',modelId='MiniCPM5-1B-F16',scene=ref['scene'],effectiveModelScene='spelling-selection-runtime-probe',templateVersion=ref['templateVersion']+'-runtime-probe',source=ref['source'],context=ref['context'],request=row['request'],response=row['response'],generationMs=row['generationMs'],residentRssBytes=row['residentRssBytes'],stage='runtime-probe',rawReference=dict(path='benchmarks/runs/rewrite-selector-minicpm-f16-v1/preflight/results.jsonl',rowSha256=rowhash(row)),expectedDiagnosticChoice={'e0':'replace' if row['sourceReference']['id']=='history-0095' else 'keep'},actualNativeCommand=row['command'])
policy.runtime_row=runtime_row

def preflight_checked(directory=PREFLIGHT):return native.checked(directory,policy=policy)
def checked(directory=RUN):
    preflight_checked();rows,labels,targets=base_checked(directory)
    refs={(r['arm'],r['id']):r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-minicpm-v1/comparison-run/results.jsonl').read_text().splitlines())}
    pairs=read(directory/'q4-comparison.json')['records'];expected=[]
    for row in rows:
        ref=refs[row['arm'],row['id']]
        if row['request']!=ref['request'] or row['prepared']!=ref['prepared'] or row['response']['usage']['prompt_tokens']!=ref['response']['usage']['prompt_tokens']:raise ValueError('Changed precision-comparison input or token count')
        expected.append(dict(id=row['id'],arm=row['arm'],rowSha256=rowhash(row),q4RowSha256=rowhash(ref),sameRequest=True,samePrepared=True,sameInputTokens=True,changedChoice=row['selection']!=ref['selection'],q4Choice=ref['selection'],f16Choice=row['selection'],q4Conforms=assessment(ref,targets)['conforms'],f16Conforms=assessment(row,targets)['conforms']))
    if pairs!=expected:raise ValueError('Changed exact Q4/F16 paired evidence')
    return rows,labels,targets
prior.checked=checked

def collect():
    probes,probe_labels,qualification=preflight_checked();summary,rows,labels=prior.collect()
    pairs=read(RUN/'q4-comparison.json')['records'];summary['q4Comparison']=dict(sameRequests=202,changedChoices=sum(p['changedChoice'] for p in pairs),improvements=sum(not p['q4Conforms'] and p['f16Conforms'] for p in pairs),regressions=sum(p['q4Conforms'] and not p['f16Conforms'] for p in pairs),limits=read(RUN/'q4-comparison.json')['limits'])
    summary['preflight']=dict(attempts=12,completedResponses=12,protocolConforming=12,taskConforming=sum(j['probeConforms'] for j in probe_labels.values()),errors=0,qualification=qualification,finalBodies=0)
    return summary,rows,labels

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();summary,rows,labels=collect();args.out.mkdir(parents=True,exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
    lines=['# Official MiniCPM5-1B F16 export comparison','','All actual choices and native probe responses personally reviewed. Precision/export artifact comparison; numerical conversion lineage and independent release acceptance unproven.','']
    for r in rows:lines += [f"## {r['arm']} / {r['id']} / {r['scene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','request','selection','editedSource','edits']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',labels[r['variant'],r['id']]['note'],'']
    for r in preflight_checked()[0]:lines += [f"## Native / {r['id']}",'','```json',json.dumps(r,ensure_ascii=False,indent=2),'```','']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
