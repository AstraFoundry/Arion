"""Pin official MiniCPM5 candidate metadata without acquiring model weights."""
from pathlib import Path
import json,urllib.request,urllib.parse,hashlib,datetime
R=Path.cwd();D=R/'.workspace/rewrite-minicpm5-v1/provisioning';catalog=json.loads((R/'.workspace/notes/rewrite-additional-small-model-catalog-v2/openbmb-models.json').read_text());indexed={m['id']:m for m in catalog};receipts=[]

def fetch(url,path):
 request=urllib.request.Request(url,headers={'User-Agent':'Arion-benchmark-metadata/1.0'})
 with urllib.request.urlopen(request,timeout=40) as response:raw=response.read();status=response.status;headers=dict(response.headers)
 path.write_bytes(raw);receipts.append(dict(url=url,path=str(path.relative_to(D)),sha256=hashlib.sha256(raw).hexdigest(),bytes=len(raw),status=status,headers=headers,retrievedAt=datetime.datetime.now(datetime.timezone.utc).isoformat()));return raw

for size in ['1B','2B']:
 for suffix in ['', '-GGUF']:
  repo='openbmb/MiniCPM5-'+size+suffix;revision=indexed[repo]['sha'];prefix=repo.split('/')[-1]
  raw=fetch('https://huggingface.co/api/models/'+repo+'?blobs=true',D/(prefix+'-api.json'));meta=json.loads(raw)
  assert meta['sha']==revision
  files={f['rfilename']:f for f in meta['siblings']}
  for name in ['README.md','LICENSE','config.json','tokenizer_config.json','chat_template.jinja']:
   if name in files:fetch('https://huggingface.co/'+repo+'/resolve/'+revision+'/'+name,D/(prefix+'-'+name))
  print(json.dumps(dict(repo=repo,revision=revision,license=(meta.get('cardData') or {}).get('license'),safetensors=meta.get('safetensors'),gguf=meta.get('gguf'),weights=[dict(path=f['rfilename'],size=f.get('size'),lfs=f.get('lfs')) for f in meta['siblings'] if f['rfilename'].endswith('.gguf')]),ensure_ascii=False),flush=True)
(D/'metadata-receipts.json').write_text(json.dumps(dict(receipts=receipts,modelGenerations=0,weightsDownloaded=0),ensure_ascii=False,indent=2)+'\n')
