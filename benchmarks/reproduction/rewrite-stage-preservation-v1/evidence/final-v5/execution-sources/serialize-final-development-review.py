import importlib.util,json
from pathlib import Path
spec=importlib.util.spec_from_file_location('r','benchmarks/reproduction/rewrite-stage-preservation-v1/report.py');r=importlib.util.module_from_spec(spec);spec.loader.exec_module(r)
p=Path('.workspace/rewrite-stage-preservation');study=Path('benchmarks/reproduction/rewrite-stage-preservation-v1');cases=study/'development.json';ids=[c['id'] for c in r.load_json(cases)];oldreview=r.load_json(p/'matrix-1/development-review.json');oldnotes={(x['arm'],x['id'],x['phase']):x for x in oldreview['records']}
names=['baseline','no-prior','no-screen','candidate'];arms={a:r.load_arm(p/f'final-development-{a}'/'observations.jsonl',ids) for a in names};review=r.review_template(cases,arms)
review['reviewer']={'kind':'separate-assistant-output-review','independence':'Separate task from production implementation. Personally inspected all 64 current outputs and exact native stage inputs/choices, then compared with former treatment outputs. Not independent human, not blind, not held out.','sourcePolicy':'Chosen V5 uses original selector/formatter prompts plus lexical current-intent hypothesis qualification and mechanical ambiguity handling; no V1/V3 instruction treatment is inferred from equal outputs.','rubric':oldreview['reviewer']['rubric']}
summary={'reviewer':'Separate assistant, not independent human/blind evaluation','arms':{},'formerTreatmentParity':{}}
for a in names:
 previous=r.load_arm(p/f'matrix-1/development-{a}/observations.jsonl',ids)
 equal=sum(o['result'].get('text')==previous['observations'][key]['result'].get('text') for key,o in arms[a]['observations'].items())
 summary['formerTreatmentParity'][a]={'textEqualPhases':equal,'denominator':16,'formerArmSha256':previous['sha256'],'currentArmSha256':arms[a]['sha256'],'scope':'Observed output equality only; source/treatment identities and timings remain distinct.'}
for row in review['records']:
 a,i,phase=row['arm'],row['id'],row['phase'];old=oldnotes[(a,i,phase)]
 for key in ['fidelity','entityCorrect','firstObservedDamage','sourceFidelity','note']:row[key]=old[key]
 current=arms[a]['observations'][(i,phase)]
 # Exact outputs were re-read above; require parity before carrying the rationale.
 previous=r.load_arm(p/f'matrix-1/development-{a}/observations.jsonl',ids)['observations'][(i,phase)]
 assert current['result']['text']==previous['result']['text'] and not current['error'] and not current['result']['cancelled']
review_path=p/'final-development-review.json';review_path.write_text(json.dumps(review,ensure_ascii=False,indent=2)+'\n');summary['reviewSha256']=r.digest(review_path.read_bytes())
for a in names:
 rows=[x for x in review['records'] if x['arm']==a];pairs={}
 for x in rows:pairs.setdefault(x['id'],{})[x['phase']]=x
 eligible=[i for i,v in pairs.items() if v['preview']['entityCorrect']];lost=[i for i,v in pairs.items() if v['preview']['entityCorrect'] and not v['final']['entityCorrect']]
 summary['arms'][a]={'phases':len(rows),**{s:sum(x['fidelity']==s for x in rows) for s in ['correct','incorrect','not-assessable']},'entityCorrectPreviewPairs':len(eligible),'previewToFinalEntityRegressionPairs':lost}
(p/'final-development-semantic-summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n');print(json.dumps(summary['arms'],ensure_ascii=False))
