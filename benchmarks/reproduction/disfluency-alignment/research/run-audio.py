import pathlib,subprocess,json,hashlib,sys
root=pathlib.Path.cwd();r=root/'.workspace/research/doubao-alignment'
for arm in (sys.argv[1:] or ['baseline','final']):
 out=r/f'audio-{arm}.jsonl'
 cmd=['caffeinate','-i',str(r/f'{arm}-audio'),'-manifest',str(r/'audio-fixtures/manifest.json'),'-corpus',str(r/'audio-corpus.json'),'-models',str(root/'.workspace/models/store'),'-bundle','xasr-160-xasr-final','-llm-worker',str(r/f'{arm}-worker'),'-llm-runtime',str(root/'apps/desktop/resources/darwin-arm64/llm'),'-llm-model',str(root/'.workspace/ablation-models/qwen3-1.7b'),'-variant',arm,'-out',str(out)]
 with (r/f'audio-{arm}.log').open('x') as log:
  print('running audio',arm,flush=True);subprocess.run(cmd,check=True,stdout=log,stderr=log)
 print('completed audio',arm,hashlib.sha256(out.read_bytes()).hexdigest(),flush=True)
