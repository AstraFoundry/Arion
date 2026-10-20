"""Capture pinned official small-model sources before native qualification."""
from pathlib import Path
import json,urllib.request,hashlib,concurrent.futures
D=Path(__file__).resolve().parent
items=[('tencent/Youtu-LLM-2B','8b0e73594661a945a7f5aafc737b63e3d8ad3f75',n) for n in ['config.json','tokenizer_config.json','README.md','LICENSE.txt']]+[('tencent/Youtu-LLM-2B-GGUF','698ad804316c9b815f9881a709d583c6214e0927',n) for n in ['README.md','LICENSE.txt']]
def get(item):
 repo,rev,name=item;url=f'https://huggingface.co/{repo}/resolve/{rev}/{name}'
 with urllib.request.urlopen(url,timeout=60) as f:b=f.read();headers=dict(f.headers)
 p=D/repo.split('/')[-1]/name;p.parent.mkdir(exist_ok=True);p.write_bytes(b)
 return dict(url=url,path=str(p.relative_to(D)),bytes=len(b),sha256=hashlib.sha256(b).hexdigest(),headers=headers)
with concurrent.futures.ThreadPoolExecutor(max_workers=6) as pool:rows=list(pool.map(get,items))
(D/'receipts.json').write_text(json.dumps(rows,indent=2)+'\n')
a=json.loads((D.parent/'notes/rewrite-additional-small-model-catalog-v3/details/Youtu-LLM-2B-GGUF-api.json').read_text())
weights=[x for x in a['siblings'] if x['rfilename'].endswith('.gguf')]
(D/'weight-candidates.json').write_text(json.dumps(weights,indent=2)+'\n')
print(json.dumps(weights,indent=2))
