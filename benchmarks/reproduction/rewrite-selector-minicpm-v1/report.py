"""Validate unchanged MiniCPM choices, native probes and disclosed failed orchestration."""
import argparse
import importlib.util
import json
import pathlib
import sys
import types
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
RUN=ROOT/'benchmarks/runs/rewrite-selector-minicpm-v1/comparison-run'
PREFLIGHT=RUN.parent/'preflight'
def module(name,path):
    spec=importlib.util.spec_from_file_location(name,path)
    value=importlib.util.module_from_spec(spec);spec.loader.exec_module(value);return value
before=list(sys.path);sys.path.insert(0,str(HERE.parent))
try:
    prior=module('minicpm_comparison_report',HERE.parent/'rewrite-selector-model-cpu-v1/report.py')
    shared=module('minicpm_shared_model_checks',HERE.parent/'gguf_model_screen_evidence.py')
finally:sys.path[:]=before
read,sha,rowhash,assessment=prior.read,prior.sha,prior.rowhash,prior.assessment
ARMS=['baseline','balanced']
prior.HERE,prior.RUN=HERE,RUN
base_checked=prior.checked
policy=types.SimpleNamespace(read=read,sha=sha,rowhash=rowhash,root=ROOT,here=HERE,arms=ARMS,pipeline=prior.pipeline,ownership=prior.ownership,assessment=assessment,template_path=HERE/'sources/native-template-policy.json',gpu_layers=99,offload_line='offloaded 25/25 layers to GPU')
def validate(rows,manifest,review,outcomes,decision,directory=RUN):
    result=shared.validate(rows,manifest,review,outcomes,decision,directory,policy=policy)
    for w in manifest['workers']:
        log=(directory/(w['name']+'-server.log')).read_text()
        if 'init: chat template, thinking = 0' not in log or '"generation_prompt":"<|im_start|>assistant\\n<think>\\n\\n</think>\\n\\n"' not in log:
            raise ValueError('Missing actual nonthinking template prefix')
    return result
prior.validate=validate

def runtime_row(row,reference):
    return dict(id=row['id'],variant=('metal99-' if row['placement']==99 else 'cpu-layer0-')+row['job'],cohort='native-runtime-compatibility-probe',modelId='MiniCPM5-1B',scene=reference['scene'],effectiveModelScene='spelling-selection-runtime-probe',templateVersion=reference['templateVersion']+'-runtime-probe',source=reference['source'],context=reference['context'],request=row['request'],response=row['response'],generationMs=row['generationMs'],residentRssBytes=row['residentRssBytes'],stage='runtime-probe',rawReference=dict(path='benchmarks/runs/rewrite-selector-minicpm-v1/preflight/results.jsonl',rowSha256=rowhash(row)),expectedDiagnosticChoice={'e0':'replace' if row['sourceReference']['id']=='history-0095' else 'keep'},actualNativeCommand=row['command'])

def preflight_checked(directory=PREFLIGHT):
    manifest,review,qualification=[read(directory/name) for name in ['manifest.json','review.json','qualification.json']]
    rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()))
    freeze,plan,addendum=[read(HERE/name) for name in ['freeze.json','preflight-plan.json','comparison-addendum.json']]
    for path,digest in addendum['nativeSourceEvidence'].items():
        if sha(HERE/path)!=digest:raise ValueError('Changed pinned native lexer source')
    if addendum['originalFreezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed initial native freeze')
    embedded=read(HERE/'sources/gguf-header-inspection.json')['metadata']['tokenizer.chat_template']
    template_policy=read(policy.template_path)
    if not embedded.endswith('\n') or template_policy['metadata']['tokenizer.chat_template']!=embedded[:-1] or template_policy['embeddedTemplateSha256']!=__import__('hashlib').sha256(embedded.encode()).hexdigest():
        raise ValueError('Incorrect native trailing-LF normalization')
    expected=dict(status='complete',plannedAttempts=12,attempts=12,completedResponses=12,errors=0,model=freeze['model'],runtimeFiles=freeze['runtimeFiles'],freezeSha256=sha(HERE/'freeze.json'),newGoPreparations=0,newAsrGenerations=0,newOcrGenerations=0,formatterGenerations=0)
    if any(manifest[k]!=v for k,v in expected.items()) or len(rows)!=12:raise ValueError('Incomplete native evidence')
    if any(x!=sha(directory/'results.jsonl') for x in [manifest['resultsSha256'],review['resultsSha256'],qualification['resultsSha256']]) or qualification['freezeSha256']!=sha(HERE/'freeze.json'):raise ValueError('Changed native response bytes')
    for path,digest in freeze['files'].items():
        if sha(ROOT/path)!=digest:raise ValueError('Changed frozen native source')
    refs={(r['arm'],r['id']):r for r in map(json.loads,(ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
    jobs={j['id']:j for j in plan['jobs']};labels={j['id']:j for j in review['reviews']};workers={w['placement']:w for w in manifest['workers']}
    if set(workers)!={99,0} or len(manifest['workers'])!=2 or len(labels)!=12:raise ValueError('Missing unique runtime workers or reviews')
    seen=set()
    for row in rows:
        key=(row['placement'],row['job']);job=jobs[row['job']];ref=refs[row['sourceReference']['arm'],row['sourceReference']['id']];worker=workers[row['placement']]
        req=json.loads(json.dumps(ref['request']));req['cache_prompt']=row['job'] in ['other-source','cache-on-return']
        if row['job'].startswith('free-'):req.pop('response_format')
        if key in seen or row['request']!=job['request'] or row['request']!=req or row['sourceReference']!=job['sourceReference'] or row['sourceReference']['rowSha256']!=rowhash(ref) or row['command']!=worker['command']:raise ValueError('Changed runtime source or request')
        cmd=row['command']
        for flag,value in [('--model',str(ROOT/freeze['modelPath'])),('--n-gpu-layers',str(row['placement'])),('--threads','4'),('--threads-batch','4'),('--ctx-size','4096'),('--parallel','1'),('--reasoning','off')]:
            if cmd[cmd.index(flag)+1]!=value:raise ValueError('Changed actual runtime settings')
        props=read(directory/(worker['name']+'-props.json'));log=(directory/(worker['name']+'-server.log')).read_text()
        if props['chat_template']!=embedded[:-1] or 'offloaded '+('25/25' if row['placement']==99 else '0/25')+' layers to GPU' not in log or 'init: chat template, thinking = 0' not in log:raise ValueError('Wrong actual template, placement or thinking mode')
        if row['httpError'] is not None or row['response']['choices'][0]['finish_reason']!='stop':raise ValueError('Incomplete native response')
        choice=json.loads(row['response']['choices'][0]['message']['content']);item=labels[row['id']];correct=choice=={'e0':'replace' if row['sourceReference']['id']=='history-0095' else 'keep'}
        if set(choice)!={'e0'} or choice['e0'] not in ['keep','replace'] or item['rowSha256']!=rowhash(row) or item['protocolConforms'] is not True or item['probeConforms']!=correct or not item['note']:raise ValueError('Incorrect native task/protocol review')
        seen.add(key)
    if seen!={(p,j) for p in [99,0] for j in jobs}:raise ValueError('Missing runtime attempts')
    parity={str(p):len({r['response']['choices'][0]['message']['content'] for r in rows if r['placement']==p and r['job'] in ['cache-off-first','cache-on-return','cache-off-return']})==1 for p in [99,0]}
    if qualification['cacheRawParity']!=parity or qualification['qualifiedForSelector']!=parity['99'] or qualification['productionQualified']:raise ValueError('Unsupported runtime qualification')
    view=directory.parent/'runtime-view';viewed=list(map(json.loads,(view/'results.jsonl').read_text().splitlines()));vm,vj=read(view/'manifest.json'),read(view/'review.json')
    if viewed!=[runtime_row(r,refs[r['sourceReference']['arm'],r['sourceReference']['id']]) for r in rows] or vm['resultsSha256']!=sha(view/'results.jsonl') or vj['resultsSha256']!=sha(view/'results.jsonl') or vm['rawResultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed native derived view')
    if vj['reviews']!=[dict(id=r['id'],variant=r['variant'],rowSha256=rowhash(r),probeConforms=labels[r['id']]['probeConforms'],note=labels[r['id']]['note']) for r in viewed]:raise ValueError('Changed native view judgments')
    aborted=directory.parent/'aborted-before-qualification';closure=read(aborted/'closure.json')
    if read(aborted/'terminal-status.json')['exitCode']!=-2 or (aborted/'run/results.jsonl').read_bytes() or closure['nativeLogCompletions']!=1 or closure['capturedResponses']!=0 or closure['productionQualified'] or 'Parsed message: {"role":"assistant","content":"{\\"e0\\":\\"keep\\"}"}' not in (aborted/'run/selector-model-v1-baseline-work-server.log').read_text():raise ValueError('Hidden aborted native attempt')
    return rows,labels,qualification

def checked(directory=RUN):
    preflight_checked();return base_checked(directory)
prior.checked=checked

def collect():
    probes,labels,qualification=preflight_checked();summary,rows,reviews=prior.collect()
    summary['preflight']=dict(attempts=12,completedResponses=12,protocolConforming=12,taskConforming=sum(j['probeConforms'] for j in labels.values()),errors=0,qualification=qualification,finalBodies=0)
    summary['abortedAttempt']=read(RUN.parent/'aborted-before-qualification/closure.json')
    return summary,rows,reviews

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();summary,rows,labels=collect();args.out.mkdir(parents=True,exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
    lines=['# MiniCPM5-1B native and unchanged selector comparison','','All actual outputs personally reviewed. Exposed development data; no independent release, new ASR/Vision or final body evidence. One interrupted native-log completion is retained separately.','']
    for r in rows:
        lines += [f"## {r['arm']} / {r['id']} / {r['scene']}",'','```json',json.dumps({k:r[k] for k in ['source','context','request','selection','editedSource','edits']},ensure_ascii=False,indent=2),'```','','```text',r['response']['choices'][0]['message']['content'],'```','',labels[r['variant'],r['id']]['note'],'']
    for r in preflight_checked()[0]:lines += [f"## Runtime / {r['id']}",'','```json',json.dumps(r,ensure_ascii=False,indent=2),'```','']
    lines += ['## Interrupted orchestration attempt','','```json',json.dumps(read(RUN.parent/'aborted-before-qualification/closure.json'),ensure_ascii=False,indent=2),'```']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n');print(json.dumps(summary,ensure_ascii=False))
if __name__=='__main__':main()
