"""Replay complete assistant review of context correction and structural editing."""
import argparse
import hashlib
import importlib.util
import json
import pathlib
import sys

HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
sys.path.insert(0, str(HERE.parent))
from scene_report import scene_metadata, scene_markdown

def read(path):return json.loads(pathlib.Path(path).read_text())
def sha(path):return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
def rowhash(row):return hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()

def validate(directory,experiment):
    directory=pathlib.Path(directory);source=HERE.parent/experiment
    for name,digest in read(source/'freeze.json')['files'].items():
        if sha(source/name)!=digest:raise ValueError('Changed frozen experiment')
    manifest=read(directory/'manifest.json')
    if manifest['status']!='complete' or manifest['freezeSha256']!=sha(source/'freeze.json'):raise ValueError('Incomplete or changed experiment')
    if manifest['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed results')
    rows=[json.loads(line) for line in (directory/'results.jsonl').read_text().splitlines()]
    variants=['production','structure-v3','context-repair-v1'] if experiment=='context-repair-v1' else ['examples-v2','structure-v3']
    samples=read(source/('cases.json' if experiment=='context-repair-v1' else 'development.json'))
    samples={sample['id']:sample for sample in samples}
    expected={(v,id) for v in variants for id in samples}
    keys=[(r['variant'],r['id']) for r in rows]
    if len(keys)!=len(expected) or set(keys)!=expected or len(set(keys))!=len(keys):raise ValueError('Missing or duplicate generation')
    review=read(directory/'review.json')
    if review['resultsSha256']!=sha(directory/'results.jsonl'):raise ValueError('Changed reviewed results')
    indexed={}
    for item in review['reviews']:
        key=(item['variant'],item['id'])
        if key in indexed or key not in expected:raise ValueError('Duplicate or unknown review')
        if any(type(item[k]) is not bool for k in ['faithful','sceneUseful']) or not item['note'].strip():raise ValueError('Incomplete judgment')
        indexed[key]=item
    if set(indexed)!=expected:raise ValueError('Missing review')
    for row in rows:
        for key,value in samples[row['id']].items():
            if row[key]!=value:raise ValueError('Changed inference input')
        item=indexed[(row['variant'],row['id'])]
        if item['rowSha256']!=rowhash(row):raise ValueError('Changed reviewed output')
        if row['variant']=='structure-v3':template=HERE.parent/f"scene-structure-v3/{row['scene']}-zh.json"
        elif row['variant']=='examples-v2':template=HERE.parent/f"scene-rewrite-prompt-v2/templates/examples-{row['scene']}-zh.json"
        elif row['variant']=='context-repair-v1':template=HERE/f"{row['scene']}-zh.json"
        else:template=None
        if template and row['templateSha256']!=sha(template):raise ValueError('Changed template identity')
    return rows,indexed

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--context',type=pathlib.Path,required=True);parser.add_argument('--structure',type=pathlib.Path,required=True);parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args()
    cr,cv=validate(args.context,'context-repair-v1');sr,sv=validate(args.structure,'scene-structure-v3')
    expectations={x['id']:x for x in read(HERE/'expectations.json')}
    out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
    summary={};scene_details=[];md=['# Complete assistant review','','All cases are development diagnostics; judgments are not independent human ratings.','']
    for experiment,rows,reviews in [('structure',sr,sv),('context',cr,cv)]:
        for row in rows:
            item=reviews[(row['variant'],row['id'])]
            group=experiment+'/'+row['variant']+'/'+row.get('kind',row.get('cohort','unknown'))
            stat=summary.setdefault(group,dict(cases=0,faithful=0,sceneUsefulAndFaithful=0,strictAccepted=0,faithfulStrictRejected=0,unfaithfulStrictAccepted=0,envelopeFailures=0))
            accepted=row['strict']=='accepted' and row['envelope']=='accepted'
            stat['cases']+=1;stat['faithful']+=item['faithful'];stat['sceneUsefulAndFaithful']+=item['faithful'] and item['sceneUseful'] and row['envelope']=='accepted';stat['strictAccepted']+=accepted
            stat['faithfulStrictRejected']+=item['faithful'] and row['envelope']=='accepted' and not accepted;stat['unfaithfulStrictAccepted']+=not item['faithful'] and accepted;stat['envelopeFailures']+=row['envelope']!='accepted'
            metadata=scene_metadata(row,direct_scene_from_runner=experiment=='structure')
            scene_details.append(dict(experiment=experiment,id=row['id'],variant=row['variant'],**metadata))
            md += [f"## {experiment} / {row['variant']} / {row['id']}",''] + scene_markdown(metadata) + ['**Input**','','```text',row['source'],'```','','**Context**','','```json',json.dumps(row['context'],ensure_ascii=False,indent=2),'```']
            if experiment=='context':md += ['','**Evaluator-only intended text**','','```text',expectations[row['id']]['intendedText'],'```']
            md += ['','**Output**','','```text',row['response'].get('text',''),'```','',f"**Review:** faithful={item['faithful']}; sceneUseful={item['sceneUseful']}; envelope={row['envelope']}; strict={row['strict']}. {item['note']}",'']
    (out/'summary.json').write_text(json.dumps(summary,indent=2)+'\n');(out/'review.md').write_text('\n'.join(md)+'\n')
    (out/'scenes.json').write_text(json.dumps(scene_details,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(summary,indent=2))

if __name__=='__main__':main()
