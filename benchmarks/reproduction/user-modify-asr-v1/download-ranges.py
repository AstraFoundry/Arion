"""Fetch independent byte ranges and verify final pinned weight digest."""
import concurrent.futures,hashlib,json,pathlib,time,urllib.request
root=pathlib.Path('.workspace/ablation-models/qwen3-asr-1.7b-mlx8');parts=root/'range-parts';parts.mkdir(exist_ok=False)
size=2463307541;step=8*1024*1024
url='https://huggingface.co/mlx-community/Qwen3-ASR-1.7B-8bit/resolve/a8379a2e2f9e313c9292cdf1af4055ab56d50d55/model.safetensors?download=true'
with urllib.request.urlopen(urllib.request.Request(url,method='HEAD'),timeout=30) as r:resolved=r.url
start=time.monotonic()
def fetch(i):
 lo=i*step;hi=min(size,lo+step)-1;p=parts/f'{i:04d}.part';n=0
 with urllib.request.urlopen(urllib.request.Request(resolved,headers={'Range':f'bytes={lo}-{hi}'}),timeout=60) as r:
  if r.status!=206 or r.headers.get('Content-Range')!=f'bytes {lo}-{hi}/{size}':raise RuntimeError('Range contract mismatch '+str(i))
  with p.open('xb') as f:
   while b:=r.read(256*1024):f.write(b);n+=len(b)
 if n!=hi-lo+1:raise RuntimeError('Short range '+str(i))
 return i,n
try:
 total=0
 with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
  for i,n in pool.map(fetch,range((size+step-1)//step)):
   total+=n;print(json.dumps({'chunk':i,'bytes':total,'elapsed':round(time.monotonic()-start,1)}),flush=True)
 h=hashlib.sha256();target=root/'model.safetensors'
 with target.open('xb') as out:
  for p in sorted(parts.glob('*.part')):
   with p.open('rb') as f:
    while b:=f.read(1024*1024):h.update(b);out.write(b)
 if h.hexdigest()!='bf304b009cc7eca79283056f787b44c952d24ac22cec787b39732bba3c23c13c':raise RuntimeError('Weight digest mismatch')
 print('Verified SHA256',h.hexdigest(),flush=True)
except BaseException as e:
 (parts/'failure.json').write_text(json.dumps({'status':'failed','type':type(e).__name__,'message':str(e).split('http')[0]})+'\n');raise
