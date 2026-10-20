import subprocess,pathlib,json
root=pathlib.Path.cwd();base=root/'.workspace/rewrite-stage-preservation';study=root/'benchmarks/reproduction/rewrite-stage-preservation-v1'
commands=[('final-intent-baseline-2','baseline','final-intent-controls.json')]+[('final-development-'+arm,arm,'development.json') for arm in ['baseline','no-prior','no-screen','candidate']]
progress=[]
for name,arm,corpus in commands:
 command=['node','scripts/rewrite-stage-command.mjs','run','--arm',arm,'--cases',str(study/corpus),'--out',str(base/name),'--runtime','/Applications/Arion.app/Contents/Resources/llm','--model','/Users/steven/Library/Application Support/Arion/models/qwen3.5-2b-q4-k-m-metal']
 print('Starting '+name,flush=True);result=subprocess.run(command,cwd=root);progress.append({'name':name,'command':command,'exitCode':result.returncode});(base/'final-followups-2-progress.json').write_text(json.dumps(progress,indent=2)+'\n')
 if result.returncode:raise SystemExit(result.returncode)
