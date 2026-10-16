import subprocess,pathlib,sys,json,hashlib
root=pathlib.Path.cwd();r=root/'.workspace/research/doubao-alignment'
split=sys.argv[1]
if split=='heldout' and (r/'hold-test').exists():
 print('held-out execution held until regression review completes',flush=True);sys.exit(0)
corpora={'fresh':r/'fresh-test.json','dev':root/'.workspace/research/multilingual-disfluency/corpus.json','heldout':r/'heldout.json','regression':root/'.workspace/live-heldout-corpus.json','alignment':root/'.workspace/research/rewrite-alignment/corpus.json'}
prefix=sys.argv[2] if len(sys.argv)>2 else split
for arm in (sys.argv[3:] or ['baseline','candidate']):
 out=r/f'{prefix}-{arm}.jsonl'
 cmd=[str(r/f'{arm}-bench'),'-corpus',str(corpora[split]),'-worker',str(r/f'{arm}-worker'),'-runtime',str(root/'apps/desktop/resources/darwin-arm64/llm'),'-model',str(root/'.workspace/ablation-models/qwen3-1.7b'),'-variant',arm,'-correct','-out',str(out)]
 with (r/f'{prefix}-{arm}.log').open('x') as log:
  print('running',split,arm,flush=True);subprocess.run(cmd,check=True,stdout=log,stderr=log)
 print('completed',split,arm,hashlib.sha256(out.read_bytes()).hexdigest(),flush=True)
