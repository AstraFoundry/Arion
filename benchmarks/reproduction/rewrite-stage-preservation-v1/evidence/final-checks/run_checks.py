import subprocess,pathlib,json,time
b=pathlib.Path('.workspace/rewrite-stage-preservation/final-checks');b.mkdir()
commands=[['go','test','./packages/core/...','./apps/desktop/infrastructure/llm/...','./apps/desktop/infrastructure/rewriting','./apps/desktop/infrastructure/provision','./apps/desktop/bootstrap/...','./models/...'],['go','test','-race','./packages/core/domain/...','./packages/core/application/...','./apps/desktop/infrastructure/llm/client/...','./apps/desktop/infrastructure/llm/gguf/...'],['go','vet','./packages/core/...','./apps/desktop/infrastructure/llm/...','./apps/desktop/cmd/arion-llm','./models/...'],['pnpm','rewrite:stages:test'],['pnpm','docs:check'],['git','diff','--check']]
results=[]
for i,c in enumerate(commands):
 start=time.monotonic();print('Running '+' '.join(c),flush=True)
 with (b/(str(i)+'.log')).open('x') as f:r=subprocess.run(c,stdout=f,stderr=subprocess.STDOUT)
 results.append({'command':c,'exitCode':r.returncode,'seconds':time.monotonic()-start,'log':str(i)+'.log'});(b/'results.json').write_text(json.dumps(results,indent=2)+'\n')
 if r.returncode:raise SystemExit(r.returncode)
print('All final checks passed',flush=True)
