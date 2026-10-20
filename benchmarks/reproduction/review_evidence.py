"""Shared integrity checks and deterministic exports for reviewed model diagnostics."""
import hashlib,json,math,pathlib

def read(path):return json.loads(pathlib.Path(path).read_text())
def sha(path):return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
def rowhash(row):return hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
def checked_rows(directory,samples,variants,template_for,model_for):
    directory=pathlib.Path(directory);manifest=read(directory/'manifest.json');review=read(directory/'review.json')
    if manifest['status']!='complete' or manifest['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Incomplete/changed generation')
    if review['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed review source')
    indexed={s['id']:s for s in samples}
    if len(indexed)!=len(samples):raise ValueError('Duplicate sample')
    expected={(v,i) for v in variants for i in indexed};rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];keys=[(r['variant'],r['id']) for r in rows];judgments={(r['variant'],r['id']):r for r in review['reviews']}
    if len(keys)!=len(expected) or set(keys)!=expected or len(review['reviews'])!=len(expected) or set(judgments)!=expected:raise ValueError('Incomplete/duplicate observation or review')
    if manifest.get('generations')!=len(expected):raise ValueError('Wrong declared generation count')
    for row in rows:
        item=judgments[(row['variant'],row['id'])]
        if any(row[k]!=v for k,v in indexed[row['id']].items()):raise ValueError('Changed request')
        if row['modelFiles']!=model_for(row):raise ValueError('Changed actual model identity')
        template=template_for(row)
        if row['effectiveModelScene']!=row['scene'] or row['templateSha256']!=sha(template) or row['response']['template']!=read(template)['version']:raise ValueError('Changed scene/template')
        if item['rowSha256']!=rowhash(row):raise ValueError('Changed reviewed output')
        if any(type(item.get(k)) is not bool for k in ['proposalFaithful','proposalSceneUseful','cleanupFaithful','cleanupSceneUseful','repairSucceeded']):raise ValueError('Missing judgment')
        if not item['note'].strip() or not item['cleanupNote'].strip():raise ValueError('Missing inspection note')
    return rows,judgments

def summarize(rows,reviews,refs):
    groups={}
    for row in rows:
        item=reviews[(row['variant'],row['id'])];ref=refs[row['id']];key=row['variant'];m=groups.setdefault(key,dict(cases=0,rawFaithful=0,rawSceneReady=0,requiredRepairs=0,repairs=0,strictFaithfulDelivered=0))
        accepted=row['strict']=='accepted' and row['envelope']=='accepted'
        m['cases']+=1;m['rawFaithful']+=item['proposalFaithful'];m['rawSceneReady']+=item['proposalFaithful'] and item['proposalSceneUseful'];m['requiredRepairs']+=ref['mustRepair'];m['repairs']+=ref['mustRepair'] and item['repairSucceeded'];m['strictFaithfulDelivered']+=item['proposalFaithful'] if accepted else item['cleanupFaithful']
    for variant,m in groups.items():
        data=[r for r in rows if r['variant']==variant]
        def q(values,p):return sorted(values)[math.ceil(p*len(values))-1]
        m.update(requestP50Ms=q([r['generationMs'] for r in data],.5),requestP95Ms=q([r['generationMs'] for r in data],.95),prefillP50Ms=q([r['response']['prefillMs'] for r in data],.5),decodeP50Ms=q([r['response']['decodeMs'] for r in data],.5),peakWorkerRssBytes=max(r['response']['peakRssBytes'] for r in data))
    return groups

def markdown(rows,reviews,title):
    lines=['# '+title,'','Complete raw/strict/cleanup inspection by assistant, not independent human ratings. Scene assignment is explicit fixture input.','']
    for row in rows:
        item=reviews[(row['variant'],row['id'])];accepted=row['strict']=='accepted' and row['envelope']=='accepted'
        lines += [f"## {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']}; template: {row['response']['template']}",'','**Source**','','```text',row['source'],'```','','**Full reference**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```','','**Raw output**','','```text',row['response']['text'],'```','',item['note'],'',f"Faithful: {item['proposalFaithful']}; scene useful: {item['proposalSceneUseful']}; strict: {row['strict']}",'','**Cleanup**','','```text',row['response']['cleanup']['text'],'```','',item['cleanupNote'],'','**Strict delivery**','','```text',row['response']['text'] if accepted else row['response']['cleanup']['text'],'```','',item['note'] if accepted else item['cleanupNote'],'']
    return '\n'.join(lines)+'\n'

def check_runtime(provenance,expected,worker):
    if provenance['packagingWorkerSha256']!=expected[worker]:raise ValueError('Wrong native worker')
    for name,digest in expected.items():
        if name!=worker and provenance['binaries'].get(name)!=digest:raise ValueError('Wrong runtime/validator')
