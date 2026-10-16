import pathlib,subprocess,json,hashlib
root=pathlib.Path.cwd();r=root/'.workspace/research/doubao-alignment'
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
state=r/'production-audio-state.json'
with state.open('x') as f:json.dump({'status':'running'},f)
manifest={'removeFillers':True,'sources':{p:sha(root/p) for p in ['apps/desktop/cmd/arion-live-bench/main.go','scripts/live-bench-command.mjs','tools/evaluation/live/score-audio.mjs','tools/evaluation/live/report.mjs']},'binaries':{n:sha(r/n) for n in ['baseline-production-audio','product-production-audio','baseline0-worker','product-worker']},'baselineCorrectionSha256':sha(r/'baseline2-correction.go'),'baselineOverlaySha256':sha(r/'baseline-production-audio-overlay.json')}
with (r/'production-audio-manifest.json').open('x') as f:json.dump(manifest,f,indent=2)
try:
 for arm,driver,worker in [('baseline0','baseline-production-audio','baseline0-worker'),('product','product-production-audio','product-worker')]:
  out=r/f'audio-production-{arm}.jsonl'
  cmd=['caffeinate','-i',str(r/driver),'-manifest',str(r/'audio-fixtures/manifest.json'),'-corpus',str(r/'audio-corpus.json'),'-models',str(root/'.workspace/models/store'),'-bundle','xasr-160-xasr-final','-remove-fillers=true','-llm-worker',str(r/worker),'-llm-runtime',str(root/'apps/desktop/resources/darwin-arm64/llm'),'-llm-model',str(root/'.workspace/ablation-models/qwen3-1.7b'),'-variant',arm+'-production','-out',str(out)]
  print('running production audio',arm,flush=True)
  with (r/f'audio-production-{arm}.log').open('x') as log:subprocess.run(cmd,check=True,stdout=log,stderr=log)
  data=list(map(json.loads,out.read_text().splitlines()));assert len(data)==8 and all(row['removeFillers'] for row in data)
 state.write_text(json.dumps({'status':'complete'})+'\n');print('production audio complete',flush=True)
except BaseException as e:state.write_text(json.dumps({'status':'failed','error':str(e)})+'\n');raise
