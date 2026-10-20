"""Policy-driven native-probe verification; historical v2 stays immutable for archive replay."""
import json

def checked(directory,*,policy):
    read,sha,rh=policy.read,policy.sha,policy.rowhash
    root,here=policy.root,policy.here
    rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()))
    manifest,review,qualification=[read(directory/name) for name in ['manifest.json','review.json','qualification.json']]
    freeze,plan=read(here/'freeze.json'),read(here/'preflight-plan.json')
    expected=dict(status='complete',plannedAttempts=12,attempts=12,completedResponses=12,errors=0,model=freeze['model'],runtimeFiles=freeze['runtimeFiles'],freezeSha256=sha(here/'freeze.json'),newGoPreparations=0,newAsrGenerations=0,newOcrGenerations=0,formatterGenerations=0)
    if len(rows)!=12 or any(manifest[k]!=v for k,v in expected.items()):raise ValueError('Incomplete actual native probe coverage')
    if any(v!=sha(directory/'results.jsonl') for v in [manifest['resultsSha256'],review['resultsSha256'],qualification['resultsSha256']]) or qualification['freezeSha256']!=sha(here/'freeze.json'):raise ValueError('Changed native response bytes or qualification')
    for path,digest in freeze['files'].items():
        if sha(root/path)!=digest:raise ValueError('Changed frozen native source: '+path)
    refs={(r['arm'],r['id']):r for r in map(json.loads,(root/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
    jobs={j['id']:j for j in plan['jobs']};labels={j['id']:j for j in review['reviews']};workers={w['placement']:w for w in manifest['workers']}
    if set(workers)!={99,0} or len(manifest['workers'])!=2 or len(labels)!=12:raise ValueError('Missing unique native workers/reviews')
    embedded=read(here/'sources/gguf-header-inspection.json')['metadata']['tokenizer.chat_template'];native=read(policy.template_path)
    expected_template=policy.native_template(embedded)
    if native['metadata']['tokenizer.chat_template']!=expected_template:raise ValueError('Wrong exact native template transformation')
    seen=set()
    for row in rows:
        key=row['placement'],row['job'];worker=workers[row['placement']];ref=refs[row['sourceReference']['arm'],row['sourceReference']['id']];job=jobs[row['job']]
        request=json.loads(json.dumps(ref['request']));request['cache_prompt']=row['job'] in ['other-source','cache-on-return']
        if row['job'].startswith('free-'):request.pop('response_format')
        if key in seen or row['request']!=request or row['request']!=job['request'] or row['sourceReference']!=job['sourceReference'] or row['sourceReference']['rowSha256']!=rh(ref) or row['command']!=worker['command']:raise ValueError('Changed exact native source/request')
        cmd=row['command']
        for flag,value in [('--model',str(root/freeze['modelPath'])),('--n-gpu-layers',str(row['placement'])),('--threads','4'),('--threads-batch','4'),('--ctx-size','4096'),('--parallel','1'),('--reasoning','off')]:
            if cmd[cmd.index(flag)+1]!=value:raise ValueError('Wrong actual native settings')
        log=(directory/(worker['name']+'-server.log')).read_text();props=read(directory/(worker['name']+'-props.json'))
        expected_offload=policy.offload_lines[row['placement']]
        if props['chat_template']!=expected_template or expected_offload not in log or 'init: chat template, thinking = 0' not in log or policy.generation_prompt_log not in log:raise ValueError('Missing actual native template/layers/nonthinking prefix')
        if row['httpError'] is not None or row['response']['choices'][0]['finish_reason']!='stop':raise ValueError('Incomplete native response')
        choice=json.loads(row['response']['choices'][0]['message']['content']);correct=choice=={'e0':'replace' if row['sourceReference']['id']=='history-0095' else 'keep'};label=labels[row['id']]
        if set(choice)!={'e0'} or choice['e0'] not in ['keep','replace'] or label['rowSha256']!=rh(row) or label['protocolConforms'] is not True or label['probeConforms']!=correct or not label['note']:raise ValueError('Changed native task/protocol review')
        seen.add(key)
    if seen!={(p,j) for p in [99,0] for j in jobs}:raise ValueError('Missing unique native attempts')
    parity={str(p):len({r['response']['choices'][0]['message']['content'] for r in rows if r['placement']==p and r['job'] in ['cache-off-first','cache-on-return','cache-off-return']})==1 for p in [99,0]}
    if qualification['cacheRawParity']!=parity or qualification['qualifiedForSelector']!=parity['99'] or qualification['productionQualified']:raise ValueError('Unsupported native/full-release qualification')
    view=directory.parent/'runtime-view';viewed=list(map(json.loads,(view/'results.jsonl').read_text().splitlines()));vm,vj=read(view/'manifest.json'),read(view/'review.json')
    expected_rows=[policy.runtime_row(row,refs[row['sourceReference']['arm'],row['sourceReference']['id']]) for row in rows]
    if viewed!=expected_rows or vm['status']!='complete' or vm['resultsSha256']!=sha(view/'results.jsonl') or vj['resultsSha256']!=sha(view/'results.jsonl') or vm['rawResultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed derived native view')
    if vj['reviews']!=[dict(id=r['id'],variant=r['variant'],rowSha256=rh(r),probeConforms=labels[r['id']]['probeConforms'],note=labels[r['id']]['note']) for r in viewed]:raise ValueError('Changed native view review')
    return rows,labels,qualification
