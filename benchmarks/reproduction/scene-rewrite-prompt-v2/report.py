"""Model-free replay of the paired prompt study and complete assistant review."""
import argparse
import csv
import importlib.util
import json
import math
import pathlib
import sys

HERE=pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
from scene_report import scene_metadata, scene_markdown
spec=importlib.util.spec_from_file_location('prompt_study',HERE/'run.py')
study=importlib.util.module_from_spec(spec);spec.loader.exec_module(study)
engine=study.engine
ROOT=study.ROOT

def checked_run(directory,phase):
    directory=pathlib.Path(directory);manifest=engine.read(directory/'manifest.json')
    if manifest['status']!='complete' or manifest['phase']!=phase:raise ValueError('Incomplete or wrong-phase run')
    if manifest['freezeSha256']!=engine.sha(HERE/'freeze.json'):raise ValueError('Changed experiment freeze')
    if manifest['resultsSha256']!=engine.sha(directory/'results.jsonl'):raise ValueError('Changed raw observations')
    rows=[json.loads(l) for l in (directory/'results.jsonl').read_text().splitlines()]
    if phase=='dev':
        samples=engine.read(HERE/'development.json');variants=['focused','examples']
    else:
        samples=engine.read(HERE/'regression.json')+engine.read(HERE/'fresh.json');variants=['production','examples']
    expected={(v,s['id']) for v in variants for s in samples}
    keys=[(r['variant'],r['id']) for r in rows]
    if len(keys)!=len(expected) or len(set(keys))!=len(keys) or set(keys)!=expected:raise ValueError('Missing, duplicated or unknown generation')
    source={s['id']:s for s in samples}
    for row in rows:
        sample=source[row['id']]
        for key in ['source','scene','language','cohort','context']:
            if row[key]!=sample[key]:raise ValueError('Changed inference input: '+key)
        if row['variant']!='production':
            template=HERE/f"templates/{row['variant']}-{row['scene']}-{row['language']}.json"
            if row['templateSha256']!=engine.sha(template):raise ValueError('Changed prompt identity')
        if row['envelope']!=engine.envelope(row['response'],row['strict']):raise ValueError('Changed envelope verdict')
    return manifest,rows

def checked_review(path,rows,results_path):
    review=engine.read(path)
    if review['resultsSha256']!=engine.sha(results_path):raise ValueError('Review is for different observations')
    expected={(r['variant'],r['id']):r for r in rows};seen=set();bykey={}
    for item in review['reviews']:
        key=(item['variant'],item['id'])
        if key not in expected or key in seen:raise ValueError('Unknown or duplicate review')
        seen.add(key);row=expected[key]
        import hashlib
        h=hashlib.sha256(json.dumps(row,sort_keys=True,ensure_ascii=False).encode()).hexdigest()
        if h!=item['rowSha256']:raise ValueError('Review source/output mismatch')
        if any(type(item[k]) is not bool for k in ['faithful','sceneUseful','envelopeValid']):raise ValueError('Missing review verdict')
        if item['envelopeValid']!=(row['envelope']=='accepted') or not item['note'].strip():raise ValueError('Invalid review evidence')
        bykey[key]=item
    if seen!=set(expected):raise ValueError('Incomplete review coverage')
    return bykey

def summarize(rows,reviews):
    groups={}
    for variant in sorted({r['variant'] for r in rows}):
        for cohort in sorted({r['cohort'] for r in rows if r['variant']==variant}):
            subset=[r for r in rows if r['variant']==variant and r['cohort']==cohort]
            group=dict(cases=len(subset),reviewedFaithful=0,reviewedSceneReady=0,strictAccepted=0,strictRejectedFaithful=0,strictAcceptedUnfaithful=0,envelopeFailures=0)
            durations=[];plain=[];context=[]
            for row in subset:
                review=reviews[(row['variant'],row['id'])];faithful=review['faithful'];valid=review['envelopeValid'];accepted=row['strict']=='accepted' and valid
                group['reviewedFaithful']+=faithful
                group['reviewedSceneReady']+=faithful and valid and review['sceneUseful']
                group['strictAccepted']+=accepted
                group['strictRejectedFaithful']+=faithful and valid and not accepted
                group['strictAcceptedUnfaithful']+=accepted and not faithful
                group['envelopeFailures']+=not valid
                durations.append(row['generationMs']);(context if row['context'] else plain).append(row['generationMs'])
            def percentile(values,q):return sorted(values)[max(0,math.ceil(q*len(values))-1)] if values else None
            group.update(p50Ms=percentile(durations,.5),p95Ms=percentile(durations,.95),withoutContextP95Ms=percentile(plain,.95),withContextP95Ms=percentile(context,.95))
            groups[variant+'/'+cohort]=group
    return groups

def main():
    p=argparse.ArgumentParser();p.add_argument('--dev',type=pathlib.Path,required=True);p.add_argument('--evaluation',type=pathlib.Path,required=True);p.add_argument('--out',type=pathlib.Path,required=True);a=p.parse_args()
    study.check_freeze();engine.check_freeze()
    dm,dr=checked_run(a.dev,'dev');em,er=checked_run(a.evaluation,'evaluation')
    for key in ['modelFiles','binaries','build']:
        if dm['provenance'][key]!=em['provenance'][key]:raise ValueError('Unpaired artifact identities')
    if em['selectionSha256']!=engine.sha(a.dev/'selection.json'):raise ValueError('Changed development selection')
    selection=engine.read(a.dev/'selection.json')
    if selection['devResultsSha256']!=engine.sha(a.dev/'results.jsonl') or selection['devReviewSha256']!=engine.sha(a.dev/'review.json'):raise ValueError('Changed development review')
    devreview=checked_review(a.dev/'review.json',dr,a.dev/'results.jsonl');evaluationreview=checked_review(a.evaluation/'review.json',er,a.evaluation/'results.jsonl')
    reviews=dict(devreview);reviews.update(evaluationreview)
    # IDs overlap between the two phases only through different variants/cohorts, not generation keys.
    rows=dr+er;summary=summarize(rows,reviews)
    fresh=[r for r in er if r['cohort']=='fresh-test']
    new_harm=[];fixed=[]
    for id in sorted({r['id'] for r in fresh}):
        before=reviews[('production',id)]['faithful'];after=reviews[('examples',id)]['faithful']
        if before and not after:new_harm.append(id)
        if not before and after:fixed.append(id)
    result=dict(kind='complete-assistant-review-not-independent-human-ratings',generations=len(rows),reviewCoverage=len(reviews),summary=summary,newFreshMaterialErrors=new_harm,fixedFreshMaterialErrors=fixed,selected='examples',productionActivated=False,limitations=['Scene/language are declared fixture metadata, not evaluated routing classifiers.','Single-assistant inspection, not independent blind review or semantic accuracy.','Shared host; per-template loads recorded separately; live scene/language switching and native insertion not evaluated.','Existing strict validator is unchanged; reviewed good rewrites can still be rejected.'])
    out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);engine.write(out/'summary.json',result)
    with (out/'cases.csv').open('x',newline='') as f:
        writer=csv.writer(f);writer.writerow(['variant','id','cohort','scene','modelScene','templateVersion','language','faithful','sceneUseful','envelope','strict','milliseconds','issues','source','output','review'])
        for row in rows:
            review=reviews[(row['variant'],row['id'])]
            writer.writerow([row['variant'],row['id'],row['cohort'],row['scene'],scene_metadata(row)['effectiveModelScene'],scene_metadata(row)['templateVersion'],row['language'],review['faithful'],review['sceneUseful'],row['envelope'],row['strict'],round(row['generationMs'],2),','.join(review['issues']),row['source'],row['response'].get('text',''),review['note']])
    md=['# Focused scene prompt v2: inspected outputs','','Every generation below was compared with its original source by the assistant. These are inspectable judgments, not independent human ratings.','']
    scene_details=[]
    for row in rows:
        review=reviews[(row['variant'],row['id'])]
        metadata=scene_metadata(row)
        scene_details.append(dict(id=row['id'],variant=row['variant'],**metadata))
        md += [f"## {row['variant']} / {row['id']}",''] + scene_markdown(metadata) + ['**Source**','','```text',row['source'],'```','','**Output**','','```text',row['response'].get('text',''),'```','',f"**Review:** faithful={review['faithful']}; sceneUseful={review['sceneUseful']}; strict={row['strict']}. {review['note']}",'']
    (out/'review.md').write_text('\n'.join(md)+'\n')
    (out/'scenes.json').write_text(json.dumps(scene_details,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False,indent=2))
if __name__=='__main__':main()
