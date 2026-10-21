import urllib.request,json,pathlib,hashlib,time
base=pathlib.Path('.workspace/qwen-asr-speed-v1')
for name in ['Qwen3-ASR-1.7B-4bit','Qwen3-ASR-0.6B-4bit']:
 repo='mlx-community/'+name
 meta=json.load(urllib.request.urlopen('https://huggingface.co/api/models/'+repo+'?blobs=true',timeout=60))
 (base/(name+'-metadata.json')).write_text(json.dumps(meta,indent=2)+'\n')
 target=base/'models'/name;target.mkdir(parents=True,exist_ok=True)
 for f in meta['siblings']:
  n=f['rfilename']
  if not n.endswith(('.json','.safetensors','.txt','.md')):continue
  out=target/n;out.parent.mkdir(parents=True,exist_ok=True)
  url='https://huggingface.co/'+repo+'/resolve/'+meta['sha']+'/'+n
  print(name,n,flush=True)
  if not out.exists():
   with urllib.request.urlopen(url,timeout=120) as r,open(str(out)+'.part','wb') as w:
    while b:=r.read(4*1024*1024):w.write(b)
   pathlib.Path(str(out)+'.part').rename(out)
  data=out.read_bytes()
  if f.get('lfs'):assert hashlib.sha256(data).hexdigest()==f['lfs']['sha256'],n
  else:assert hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()==f['blobId'],n
 print('VERIFIED',name,meta['sha'],flush=True)
