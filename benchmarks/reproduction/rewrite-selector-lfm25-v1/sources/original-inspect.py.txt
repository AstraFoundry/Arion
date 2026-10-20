"""Capture pinned latest small-family source and license metadata without weights."""
from pathlib import Path
import json,urllib.request,hashlib,concurrent.futures
D=Path(__file__).resolve().parent;C=D.parent/'notes/rewrite-additional-small-model-catalog-v4';names=['LFM2.5-1.2B-Instruct','LFM2.5-1.2B-Instruct-GGUF'];items=[]
for name in names:
 x=json.loads((C/(name+'-api.json')).read_text());p=D/name;p.mkdir(exist_ok=False)
 for row in x['siblings']:
  n=row['rfilename']
  if n in ['README.md','LICENSE','LICENSE.txt','config.json','tokenizer_config.json','chat_template.jinja','generation_config.json']:items.append((name,x['sha'],n))
 if name.endswith('GGUF'):(D/'weight-candidates.json').write_text(json.dumps([r for r in x['siblings'] if r['rfilename'].endswith('.gguf')],indent=2)+'\n')
def get(item):
 name,rev,n=item;url=f'https://huggingface.co/LiquidAI/{name}/resolve/{rev}/{n}'
 with urllib.request.urlopen(url,timeout=60) as f:b=f.read();headers=dict(f.headers)
 (D/name/n).write_bytes(b);return dict(url=url,path=name+'/'+n,bytes=len(b),sha256=hashlib.sha256(b).hexdigest(),headers=headers)
with concurrent.futures.ThreadPoolExecutor(max_workers=6) as pool:rows=list(pool.map(get,items))
(D/'receipts.json').write_text(json.dumps(rows,indent=2)+'\n');print('Official pinned source files',len(rows),'zero weights/model generations')
