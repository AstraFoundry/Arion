import importlib.util,json,hashlib
from pathlib import Path
spec=importlib.util.spec_from_file_location('r','benchmarks/reproduction/rewrite-stage-preservation-v1/report.py');r=importlib.util.module_from_spec(spec);spec.loader.exec_module(r)
p=Path('.workspace/rewrite-stage-preservation');study=Path('benchmarks/reproduction/rewrite-stage-preservation-v1');cases=study/'final-validation.json';fixtures=r.load_json(cases);ids=[c['id'] for c in fixtures];derived=p/'final-validation-baseline-derived';derived.mkdir(exist_ok=False)
sources=[(p/'fresh-contrast-matrix/baseline.jsonl',fixtures[:16]),(p/'final-intent-baseline-2/observations.jsonl',fixtures[16:])];lineage=[];rawlines=[]
for source,subset in sources:
 rows=[json.loads(x) for x in source.read_bytes().splitlines() if x.strip()]
 assert rows[-1]['kind']=='complete'
 chosen={c['id']:c for c in subset};seen=set();mapping=[]
 for raw in source.read_bytes().splitlines(keepends=True):
  row=json.loads(raw)
  if row['kind']!='observation' or row['id'] not in chosen:continue
  key=(row['id'],row['phase']);assert key not in seen;seen.add(key);case=chosen[row['id']]
  assert row['request']['Text']==case[row['phase']] and row['request']['Scene']==case['scene'] and row['request']['Context']['Screen']==case['screen']
  rawlines.append(raw if raw.endswith(b'\n') else raw+b'\n');mapping.append({'id':row['id'],'phase':row['phase'],'rawLineSha256':r.digest(raw)})
 assert seen=={(c['id'],phase) for c in subset for phase in ['preview','final']}
 lineage.append({'source':str(source.resolve()),'sha256':r.digest(source.read_bytes()),'observations':mapping})
header={'kind':'start','origin':'verbatim-existing-observation-composition','cases':str(cases.resolve()),'casesSha256':r.digest(cases.read_bytes()),'sources':lineage,'limitations':'Composite of two separately completed native baseline runs. No synthetic load event/timing is added; stage and phase timings stay their original values. Sequential nonrandom arms and host drift prohibit attributing speed differences solely to treatment.'}
with (derived/'observations.jsonl').open('xb') as f:
 f.write(json.dumps(header,ensure_ascii=False).encode()+b'\n')
 for line in rawlines:f.write(line)
 f.write(b'{"kind":"complete","origin":"composition-coverage-complete"}\n')
(derived/'lineage.json').write_text(json.dumps(header,ensure_ascii=False,indent=2)+'\n')
for idx,(source,_) in enumerate(sources):
 target=derived/f'original-source-{idx+1}.jsonl';target.write_bytes(source.read_bytes())
arms={'baseline-composite':r.load_arm(derived/'observations.jsonl',ids),'candidate-v5':r.load_arm(p/'final-validation-candidate/observations.jsonl',ids)};review=r.review_template(cases,arms)
review['reviewer']={'kind':'separate-assistant-output-review','independence':'Separate task from production implementation; personally inspected all 96 native observation outputs and their stage inputs/choices. Not independent human or blind. Previously exposed first16 contrasts are regression cases, additional8 were frozen before chosen-source generation; no held-out natural-speech claim.','rubric':'Current source intent and literal spelling outrank previous recognition; restore only supported same-product references. Preserve phase-specific negation/number/literal constraints. Exact source casing is not a semantic error by itself.','baselineProvenance':'First16 original fresh baseline observations plus new8 completed baseline-2 observations copied verbatim; see derived lineage source hashes. No model run or output was fabricated by composition.'}
old=r.load_json(p/'fresh-contrast-matrix/assistant-review.json');oldnotes={(x['id'],x['phase']):x for x in old['records'] if x['arm']=='baseline'};positive={1,3,5,7,16}
for x in review['records']:
 arm,i,phase=x['arm'],x['id'],x['phase'];final=phase=='final';x.update(fidelity='correct',entityCorrect=True,firstObservedDamage='none-observed')
 if i.startswith('fresh-prior-'):
  n=int(i.split('-')[2]);oldrow=oldnotes[(i,phase)]
  if arm=='baseline-composite':
   for k in ['fidelity','entityCorrect','firstObservedDamage','note']:x[k]=oldrow[k]
  elif not final:x['note']=oldrow['note']
  elif n in positive:
   case=next(c for c in fixtures if c['id']==i);x['note']='Selector e0=replace correctly restores '+case['checks'][0]+' under current same-product continuation and independent screen evidence. Original selector/formatter prompts are used; formatter preserves the exact name, action, and current constraints.'
  elif n==15:x['note']='Ambiguous earlier product matches are mechanically forced to keep; no model selector call is made. Formatter preserves lum board, the status check, and lack-of-information statement. This completes successfully, unlike the rejected V3 timeout.'
  elif n in (13,14):x['note']=oldrow['note']
  else:x['note']=oldrow['note']+' Chosen-source input qualification excludes the conflicting earlier-name hypothesis before sampling, and the native trace has no selector call for it.'
 else:
  n=int(i.rsplit('-',1)[1])
  if not final:x['note']='Correct source product name and check request are preserved exactly once. Final-only constraints are not yet present and are not required.'
  elif n in (2,4,6,8):
   if arm=='baseline-composite':x.update(fidelity='incorrect',entityCorrect=False,firstObservedDamage='unrecovered-final-source-spelling');x['note']='Current source refers to the same earlier, screen-supported product, but baseline fails to restore its canonical name and leaves the two-word variant. '+('It inserts article the before ced bridge without resolving Cedarbridge. ' if n==4 else '')+('Unrelated alpha beta literal remains correct. ' if n==6 else '')+('250 millisecond timeout remains correct. ' if n==8 else '')+'This is missing supported recovery, not posterior corruption of an already-correct selector output.'
   else:
    target={2:'Amberfield',4:'Cedarbridge',6:'Maplepath',8:'Oceanframe'}[n];x['note']='Selector e0=replace correctly restores '+target+'; formatter preserves it once and retains current intent. '+({2:'The same-project continuation and launch-log request survive.',4:'The same-application continuation and log export survive.',6:'The separate next-sentence literal alpha beta remains unchanged with its internal space, so preserving that unrelated literal does not globally disable a supported earlier-name repair.',8:'Timeout remains exactly 250 milliseconds and the explicit same-product reference survives.'}[n])
  elif n==1:x['note']='Current source explicitly identifies amb field as a phrase and requires the two words to remain separate. Output preserves that spelling and constraint; earlier Amberfield is not substituted.'
  elif n==3:x['note']='Intentional phrase ced bridge and its internal space are preserved; earlier Cedarbridge is not substituted.'
  elif n==5:x['note']='Current source explicitly corrects previous recognition and makes map path authoritative. Output preserves the two-word current spelling, overriding stale earlier Maplepath.'
  elif n==7:x['note']='Current source says previous product recognition was mistaken; output retains oce frame rather than importing earlier Oceanframe.'
  else:raise ValueError(i)
review_path=p/'final-validation-review.json';review_path.write_text(json.dumps(review,ensure_ascii=False,indent=2)+'\n')
summary={'reviewer':'Separate assistant, not independent human/blind evaluation','reviewSha256':r.digest(review_path.read_bytes()),'scope':'48 phases per arm from authored current-intent contrasts. Baseline is a documented verbatim composite. No global accuracy or release-quality claim.','arms':{}}
for arm in arms:
 rows=[x for x in review['records'] if x['arm']==arm];summary['arms'][arm]={'phases':len(rows),**{s:sum(x['fidelity']==s for x in rows) for s in ['correct','incorrect','not-assessable']},'supportedRepairFinals':{'correct':sum(x['fidelity']=='correct' for x in rows if x['phase']=='final' and ((x['id'].startswith('fresh') and int(x['id'].split('-')[2]) in positive) or (x['id'].startswith('final') and int(x['id'].rsplit('-',1)[1]) in (2,4,6,8)))),'denominator':9},'preservationFinals':{'correct':sum(x['fidelity']=='correct' for x in rows if x['phase']=='final' and not ((x['id'].startswith('fresh') and int(x['id'].split('-')[2]) in positive) or (x['id'].startswith('final') and int(x['id'].rsplit('-',1)[1]) in (2,4,6,8)))),'denominator':15}}
(p/'final-validation-semantic-summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n');print(json.dumps(summary['arms'],ensure_ascii=False))
