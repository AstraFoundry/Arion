"""Verify actual Youtu native probes and unchanged exposed selector requests."""
import argparse,importlib.util,json,pathlib,sys,types
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
RUN=ROOT/'benchmarks/runs/rewrite-selector-youtu-v1/run'
PREFLIGHT=RUN.parent/'preflight'
def module(name,path):
    spec=importlib.util.spec_from_file_location(name,path);value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value
before=list(sys.path);sys.path.insert(0,str(HERE.parent))
try:
    prior=module('youtu_prior_report',HERE.parent/'rewrite-selector-model-cpu-v1/report.py')
    shared=module('youtu_shared_model',HERE.parent/'gguf_model_screen_evidence.py')
    native=module('youtu_native',HERE.parent/'gguf_native_probe_evidence_v3.py')
finally:sys.path[:]=before
read,sha,rowhash,assessment=prior.read,prior.sha,prior.rowhash,prior.assessment
ARMS=['baseline','balanced'];prior.HERE,prior.RUN=HERE,RUN;base_checked=prior.checked
policy=types.SimpleNamespace(read=read,sha=sha,rowhash=rowhash,root=ROOT,here=HERE,arms=ARMS,pipeline=prior.pipeline,ownership=prior.ownership,assessment=assessment,template_path=HERE/'sources/native-template-policy.json',gpu_layers=99,offload_line='offloaded 33/33 layers to GPU',native_template=lambda embedded:embedded,offload_lines={99:'offloaded 33/33 layers to GPU',0:'offloaded 0/33 layers to GPU'},generation_prompt_log='"generation_prompt":"<|Assistant|><think>\\n\\n</think>\\n\\n"')
def validate(rows,manifest,review,outcomes,decision,directory=RUN):
    return shared.validate(rows,manifest,review,outcomes,decision,directory,policy=policy)
prior.validate=validate

def runtime_row(row,ref):
    return dict(id=row['id'],variant=('metal99-' if row['placement']==99 else 'cpu-layer0-')+row['job'],cohort='native-runtime-compatibility-probe',modelId='Youtu-LLM-2B-Q8_0',scene=ref['scene'],effectiveModelScene='spelling-selection-runtime-probe',templateVersion=ref['templateVersion']+'-runtime-probe',source=ref['source'],context=ref['context'],request=row['request'],response=row['response'],generationMs=row['generationMs'],residentRssBytes=row['residentRssBytes'],stage='runtime-probe',rawReference=dict(path='benchmarks/runs/rewrite-selector-youtu-v1/preflight/results.jsonl',rowSha256=rowhash(row)),expectedDiagnosticChoice={'e0':'replace' if row['sourceReference']['id']=='history-0095' else 'keep'},actualNativeCommand=row['command'])
policy.runtime_row=runtime_row

def preflight_checked(directory=PREFLIGHT):return native.checked(directory,policy=policy)
def checked(directory=RUN):
    for path,digest in read(HERE/'comparison-sources.json')['files'].items():
        if sha(ROOT/path)!=digest:raise ValueError('Changed supplemental comparison source: '+path)
    preflight_checked();return base_checked(directory)
prior.checked=checked

def collect():
    probes,labels,qualification=preflight_checked();summary,rows,reviews=prior.collect()
    summary['preflight']=dict(attempts=12,completedResponses=12,protocolConforming=12,taskConforming=sum(j['probeConforms'] for j in labels.values()),errors=0,qualification=qualification,finalBodies=0)
    return summary,rows,reviews

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();summary,rows,labels=collect();args.out.mkdir(parents=True,exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
    lines=['# Youtu-LLM-2B Q8 native and selector development comparison','','All actual responses personally reviewed. Research-only custom license; no final bodies, independent release or actual16GB acceptance.','']
    for r in rows:lines += [f"## {r['arm']} / {r['id']} / {r['scene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','request','selection','editedSource','edits']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',labels[r['variant'],r['id']]['note'],'']
    for r in preflight_checked()[0]:lines += [f"## Native / {r['id']}",'','```json',json.dumps(r,ensure_ascii=False,indent=2),'```','']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
