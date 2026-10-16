import pathlib,subprocess,sys,json,hashlib
root=pathlib.Path.cwd();r=root/'.workspace/research/doubao-alignment'
assert json.loads((r/'final-state.json').read_text())['status']=='complete'
state=r/'matched-state.json'
with state.open('x') as f:json.dump({'status':'running'},f)
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
manifest={'baseManifestSha256':sha(r/'final-manifest.json'),'binaries':{n:sha(r/n) for n in ['baseline0-worker','baseline0-bench','baseline0-audio','final-worker','final-bench','final-audio']},'buildInfo':{n:subprocess.check_output(['go','version','-m',str(r/n)],text=True).replace(str(root),'$REPOSITORY') for n in ['baseline0-worker','final-worker','baseline0-bench','final-bench','baseline0-audio','final-audio']},'baselineOverlay':{n:sha(r/n) for n in ['baseline0-main.go','baseline-engine.go','baseline0-overlay.json','baseline-bench-overlay.json','baseline-rewrite.go','baseline-fidelity.go','baseline-live.go']}}
with (r/'matched-manifest.json').open('x') as f:json.dump(manifest,f,indent=2)
try:
 for split in ['dev','alignment','regression','heldout']:
  subprocess.run([sys.executable,str(r/'run-text.py'),split,'matched-'+split,'baseline0'],check=True)
  old=r/(('final-heldout-baseline.jsonl') if split=='heldout' else f'{split}-baseline.jsonl')
  new=r/f'matched-{split}-baseline0.jsonl'
  a,b=[list(map(json.loads,p.read_text().splitlines())) for p in [old,new]]
  assert len(a)==len(b)
  for x,y in zip(a,b):
   assert all(x[k]==y[k] for k in ['caseId','source','proposal','verdict','text','truncated']), (split,x['caseId'],'CGO changed outputs')
  print(split,'baseline CGO parity confirmed',flush=True)
 subprocess.run([sys.executable,str(r/'run-audio.py'),'baseline0'],check=True)
 state.write_text(json.dumps({'status':'complete'})+'\n')
except BaseException as e:
 state.write_text(json.dumps({'status':'failed','error':str(e)})+'\n');raise
