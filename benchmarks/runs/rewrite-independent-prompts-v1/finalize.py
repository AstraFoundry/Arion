import pathlib,json,hashlib,shutil
ROOT=pathlib.Path('/Users/steven/code/Arion');W=ROOT/'.workspace/rewrite-independent-prompts-v1';H=ROOT/'benchmarks/reproduction/rewrite-independent-prompts-v1';R=ROOT/'benchmarks/runs/rewrite-independent-prompts-v1'
manifest=json.load(open(W/'isolated-authoring-manifest.json'));assert manifest['status']=='complete' and manifest['exitCode']==0
export=json.load(open(W/'isolated-session-export.json'));info=export['info'];assert info['id']==manifest['sessionID'];assert info['location']['directory']==manifest['explicitLocation'];assert '/code/Arion' not in info['location']['directory'];assert info['model']['providerID']=='opencode-go' and info['model']['id']=='gemini-4-argon-contributor'
assert info['permissions']==[{'action':'*','resource':'*','effect':'deny'}]
events=[json.loads(z) for z in (W/'isolated-events.jsonl').read_text().splitlines()]
assert not any(z['type'] in ['tool_use','error'] for z in events)
raw=''.join(z.get('part',{}).get('text','') for z in events if z['type']=='text');candidate=json.loads(raw)
assert set(candidate)=={'selection','format','designNotes'}
assert candidate['selection']['selectionExamples']==[]
fmt=candidate['format'];assert set(fmt['format'])=={'work','email','prompt','chat'};assert set(fmt['routeInstructions'])=={'han','latin','mixed'}
for scene,recipe in fmt['format'].items():
 assert recipe['examples']==[] and recipe['instruction'] and recipe['instructionHan']
assert fmt['formatSystem'] and fmt['formatSystemHan'] and candidate['selection']['selectionSystem']
for d in [H,R]:d.mkdir(parents=True,exist_ok=True)
# Preserve model-authored prompt strings verbatim. Formatting JSON is the only transformation.
(H/'candidate.json').write_text(raw+'\n')
for name in ['selection','format']:(H/(name+'.json')).write_text(json.dumps(candidate[name],ensure_ascii=False,indent=2)+'\n')
for name in ['isolated-author-brief.txt','isolated-config.json','explicit-session-create.json','isolated-authoring-manifest.json','isolated-events.jsonl','isolated-stderr.log','isolated-session-export.json','author-isolated.py','finalize.py']:
 source=W/name
 if source.exists():(R/name).write_bytes(source.read_bytes())
# Earlier implicit-location drafts are disqualified and never used as input
# to the accepted independent session. Keep their original observations.
D=R/'discarded-implicit-location';D.mkdir(exist_ok=True)
for name in ['authoring-manifest.json','opencode-events.jsonl','opencode-stderr.log','session-export.json','revision-events.jsonl','revision-stderr.log','revision-status.json']:
 source=W/name
 if source.exists():(D/name).write_bytes(source.read_bytes())
(D/'reason.json').write_text(json.dumps({'disposition':'discarded','reason':'Initial CLI launches did not bind an isolated session location; their session pointed to Arion. Neither draft nor revision was supplied to the accepted fresh session.'},indent=2)+'\n')
A=R/'failed-custom-agent';A.mkdir(exist_ok=True)
for source in (W/'attempt-1').iterdir():
 if source.is_file():(A/source.name).write_bytes(source.read_bytes())
summary={'authorModel':info['model'],'sessionID':info['id'],'explicitLocation':info['location']['directory'],'toolEvents':0,'examples':0,'providedMaterial':'Only the archived product interface/specification and general consistency requirements; no old prompts, benchmark cases, answers, metrics or earlier drafts.','promptStringsEditedAfterModelOutput':False,'localQwenEvaluationRuns':0,'productionActivated':False,'scope':'Independent instruction-only candidate; no empirically established quality, budget fit or guarantee of generalization.'}
(R/'summary.json').write_text(json.dumps(summary,indent=2)+'\n')
print(json.dumps(summary,ensure_ascii=False))
