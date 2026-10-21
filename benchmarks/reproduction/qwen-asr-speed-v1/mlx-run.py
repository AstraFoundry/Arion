"""Offline, reference-free ASR compute diagnostic on the frozen development corpus."""
import argparse,hashlib,json,pathlib,subprocess,time,sys,wave,resource,importlib.metadata,traceback

def sha(p):
 h=hashlib.sha256()
 with open(p,'rb') as f:
  while b:=f.read(8*1024*1024):h.update(b)
 return h.hexdigest()

def main():
 a=argparse.ArgumentParser();a.add_argument('--model',required=True);a.add_argument('--metadata',required=True);a.add_argument('--corpus',required=True);a.add_argument('--out',required=True);a=a.parse_args()
 out=pathlib.Path(a.out);out.mkdir(parents=True,exist_ok=False)
 modeldir=pathlib.Path(a.model).resolve();corpuspath=pathlib.Path(a.corpus).resolve();metadata=json.load(open(a.metadata));corpus=json.load(open(corpuspath))
 run={'status':'running','stage':'offline-engine-compute','modelPath':str(modeldir),'revision':metadata['sha'],'corpusSha256':sha(corpuspath),'runnerSha256':sha(__file__),'metadataSha256':sha(a.metadata),'modelHashes':{},'seed':11,'maxTokens':128,'temperature':0,'context':'empty','order':'corpus order, per-model blocks','limitations':['Warm engine compute excludes streaming, coordinator, capture, rendering and insertion.','Community conversion; source FP parity unverified.','Development audio reused; no release or independent semantic acceptance.'],'hardware':{'os':subprocess.check_output(['sw_vers'],text=True),'cpu':subprocess.check_output(['sysctl','-n','machdep.cpu.brand_string'],text=True).strip(),'ramBytes':int(subprocess.check_output(['sysctl','-n','hw.memsize'],text=True)),'powerMode':subprocess.check_output(['pmset','-g'],text=True)},'runtime':{d.metadata['Name']:d.version for d in importlib.metadata.distributions()},'source':{'git':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),'dirty':subprocess.check_output(['git','status','--porcelain'],text=True)}}
 def save():(out/'run.json').write_text(json.dumps(run,ensure_ascii=False,indent=2)+'\n')
 save()
 try:
  for item in metadata['siblings']:
   name=item['rfilename']
   if not name.endswith(('.json','.safetensors','.txt','.md')):continue
   p=modeldir/name;digest=sha(p);run['modelHashes'][name]=digest
   if item.get('lfs'):assert digest==item['lfs']['sha256'],name
   else:
    data=p.read_bytes();assert hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()==item['blobId'],name
  run['weightBytes']=(modeldir/'model.safetensors').stat().st_size
  # Construct inputs with audio identity only. Reference and language labels never enter inference.
  groups={split:[{'id':s['id'],'audio':s['audio']} for s in corpus['samples'] if s['split']==split] for split in ['train','dev']}
  assert len(groups['train'])>=2 and len(groups['dev'])==118
  for s in groups['train'][:2]+groups['dev']:assert sha(corpuspath.parent/s['audio']['path'])==s['audio']['sha256'],s['id']
  import mlx.core as mx
  import mlx_audio
  import numpy as np
  from mlx_audio.stt import load
  run['runtimeArtifactHashes']={str(p):sha(p) for base in [pathlib.Path(mx.__file__).parent,pathlib.Path(mlx_audio.__file__).parent] for p in base.rglob('*') if p.is_file() and p.suffix in {'.py','.so','.dylib','.metallib'}}
  qwen_source=pathlib.Path(mlx_audio.__file__).parent/'stt/models/qwen3_asr/qwen3_asr.py'
  assert sha(qwen_source)=='661e6e930842dcd0cd509a75964f4b0fcf4d935a52011062d2978f050bc6b38d'
  run['mlxAudioSourceRevision']='dad2087f790dfe96b07d5aac1405db4b578b411f'
  run['pythonSha256']=sha(sys.executable)
  run['packageSources']={d.metadata['Name']:d.read_text('direct_url.json') for d in importlib.metadata.distributions() if d.read_text('direct_url.json')}
  mx.random.seed(11);t=time.perf_counter();model=load(str(modeldir));mx.synchronize();run['loadMs']=(time.perf_counter()-t)*1000;save()
  for group,name in [(groups['train'][:2],'warmups.jsonl'),(groups['dev'],'results.jsonl')]:
   with (out/name).open('x') as f:
    for s in group:
     with wave.open(str(corpuspath.parent/s['audio']['path']),'rb') as w:
      assert (w.getframerate(),w.getnchannels(),w.getsampwidth())==(16000,1,2)
      audio=np.frombuffer(w.readframes(w.getnframes()),dtype='<i2').astype(np.float32)/32768
     mx.random.seed(11);mx.synchronize();t=time.perf_counter();r=model.generate(audio,system_prompt='',max_tokens=128,temperature=0,verbose=False);mx.synchronize();elapsed=(time.perf_counter()-t)*1000
     row={'caseId':s['id'],'audioSha256':s['audio']['sha256'],'text':r.text,'elapsedMs':elapsed,'generationTokens':r.generation_tokens,'promptTokens':r.prompt_tokens,'tokenLimitReached':r.generation_tokens>=128,'peakRssBytes':resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,'peakMlxBytes':mx.get_peak_memory()}
     f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush();print(name,s['id'],round(elapsed),flush=True)
  run.update(status='complete',resultsSha256=sha(out/'results.jsonl'),warmupsSha256=sha(out/'warmups.jsonl'));save()
 except BaseException as e:run.update(status='failed',error=repr(e));save();raise
if __name__=='__main__':main()
