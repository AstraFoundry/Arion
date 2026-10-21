import pathlib,time,json,subprocess,os
root=pathlib.Path('.workspace/qwen-asr-speed-v1')
while True:
 run=root/'native-paired/run.json'
 if run.exists():
  status=json.load(open(run))['status']
  if status=='failed':raise RuntimeError('Native run failed; inspect before GPU trials')
  if status=='complete':break
 time.sleep(5)
while not all((root/'models'/n/'vocab.json').exists() for n in ['Qwen3-ASR-1.7B-4bit','Qwen3-ASR-0.6B-4bit']):time.sleep(5)
python='.workspace/user-modify-asr-venv/bin/python'
for name,model,meta in [('mlx17-8bit','.workspace/ablation-models/qwen3-asr-1.7b-mlx8','Qwen3-ASR-1.7B-8bit-metadata.json'),('mlx17-4bit',str(root/'models/Qwen3-ASR-1.7B-4bit'),'Qwen3-ASR-1.7B-4bit-metadata.json'),('mlx06-4bit',str(root/'models/Qwen3-ASR-0.6B-4bit'),'Qwen3-ASR-0.6B-4bit-metadata.json')]:
 print('START',name,flush=True)
 with (root/(name+'.log')).open('x') as log:
  subprocess.run([python,str(root/'mlx-run.py'),'--model',model,'--metadata',str(root/meta),'--corpus','benchmarks/corpora/asr-v1/corpus.json','--out',str(root/name)],env={**os.environ,'HF_HUB_OFFLINE':'1'},stdout=log,stderr=subprocess.STDOUT,check=True)
 print('DONE',name,flush=True)
