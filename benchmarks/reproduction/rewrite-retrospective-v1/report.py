"""Validate retrospective views against immutable historical inference rows."""
import hashlib,json,pathlib
ROOT=pathlib.Path(__file__).resolve().parents[3]
OLD=ROOT/'benchmarks/runs/scene-rewrite-alignment-v1'
OUT=ROOT/'benchmarks/runs/rewrite-retrospective-v1'
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
rh=lambda r:hashlib.sha256(json.dumps(r,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
def checked(phase):
    directory=OUT/phase;manifest=json.loads((directory/'manifest.json').read_text());review=json.loads((directory/'review.json').read_text())
    if manifest['status']!='complete' or manifest['newGenerations']!=0:raise ValueError('Not a completed retrospective')
    if manifest['resultsSha256']!=sha(directory/'results.jsonl') or review['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed retrospective')
    original={}
    for source in manifest['sources']:
        path=ROOT/source['path']
        if sha(path)!=source['sha256']:raise ValueError('Changed historical source')
        original.update({(r['generationArm'],r['id']):r for r in [json.loads(x) for x in path.read_text().splitlines()]})
    corpus=ROOT/'benchmarks/reproduction/scene-rewrite-alignment-v1/corpus.json'
    if manifest['corpusSha256']!=sha(corpus):raise ValueError('Changed source corpus')
    samples={r['id']:r for r in json.loads(corpus.read_text())['samples'] if r['split']==phase}
    rows=[json.loads(x) for x in (directory/'results.jsonl').read_text().splitlines()];reviews={(r['variant'],r['id']):r for r in review['reviews']};expected={(v,i) for v in ['production','scene'] for i in samples}
    if len(rows)!=48 or {(r['variant'],r['id']) for r in rows}!=expected or len(reviews)!=48 or len(review['reviews'])!=48:raise ValueError('Incomplete review')
    for row in rows:
        key=(row['variant'],row['id']);raw=original[key];sample=samples[row['id']]
        if row['originalRowSha256']!=rh(raw) or any(row[k]!=v for k,v in raw.items()):raise ValueError('Historical inference changed')
        if row['source']!=sample['source'] or row['scene']!=raw['requestedScene'] or row['scene']!=sample['scene']:raise ValueError('Historical request changed')
        if reviews[key]['rowSha256']!=rh(row):raise ValueError('Review binding changed')
    return rows,reviews

def main():
    summary={};lines=['# Retrospective manual review','','No new inference. Original selection and proxy scores remain unchanged. Assistant inspection under the current rewrite contract; not independent blind ratings.','']
    for phase in ['dev','test']:
        rows,reviews=checked(phase)
        for row in rows:
            item=reviews[(row['variant'],row['id'])];key=phase+'/'+row['variant'];m=summary.setdefault(key,dict(cases=0,faithful=0,sceneReady=0));m['cases']+=1;m['faithful']+=item['proposalFaithful'];m['sceneReady']+=item['proposalFaithful'] and item['proposalSceneUseful']
            lines += [f"## {phase} / {row['variant']} / {row['id']}",'',f"Scene: {row['scene']} → {row['effectiveModelScene']} (requested by fixture); template: {row['response']['template']}",'','**Source**','','```text',row['source'],'```','','**Raw output**','','```text',row['response']['text'],'```','',item['note'],'',f"Faithful: {item['proposalFaithful']}; scene ready: {item['proposalSceneUseful']}; strict: {row['strict']}",'','**Cleanup**','','```text',row['response']['cleanup']['text'],'```','',item['cleanupNote'],'']
    target=OUT/'report';target.mkdir(exist_ok=False);(target/'review.md').write_text('\n'.join(lines)+'\n');(target/'summary.json').write_text(json.dumps(dict(reviewedOriginalGenerations=96,newGenerations=0,metrics=summary),indent=2)+'\n');print(json.dumps(summary))
if __name__=='__main__':main()
