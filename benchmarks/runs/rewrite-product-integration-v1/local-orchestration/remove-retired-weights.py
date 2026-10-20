"""Remove only explicitly inventoried retired rewrite/correction weight files."""
import json,pathlib,hashlib
root=pathlib.Path('/Users/steven/code/Arion');out=root/'.workspace/rewrite-product-integration/removed-weights.json'
rows=json.loads((root/'.workspace/rewrite-product-integration/weight-inventory.json').read_text())
paths=[]
for row in rows:
 p=pathlib.Path(row['path'])
 if p.suffix=='.gguf' and p.name!='Qwen3.5-2B-Q4_K_M.gguf':paths.append(p)
 elif str(p).startswith('.workspace/ablation-models/qwen3-') and '-asr' not in str(p):paths.append(p)
 elif str(p).startswith('.workspace/correction-chain/models/'):paths.append(p)
paths.extend([pathlib.Path('.workspace/correction-chain-reassessment/models/fastcorrect/fastcorrect_pretrain.pt'),pathlib.Path('.workspace/ablation-models/qwen3-4b/model.onnx')])
records=[]
for p in paths:
 absolute=root/p
 assert absolute.is_file() and not absolute.is_symlink() and str(absolute).startswith(str(root/'.workspace')+'/')
 with absolute.open('rb') as f:digest=hashlib.file_digest(f,'sha256').hexdigest()
 records.append(dict(path=str(p),bytes=absolute.stat().st_size,sha256=digest,removed=False))
out.write_text(json.dumps(dict(scope='Retired rewrite/correction weights only; ASR, runtime environments, raw observations and provenance retained.',files=records),indent=2)+'\n')
for item in records:
 (root/item['path']).unlink();item['removed']=True
 out.write_text(json.dumps(dict(scope='Retired rewrite/correction weights only; ASR, runtime environments, raw observations and provenance retained.',files=records),indent=2)+'\n')
print(json.dumps(dict(files=len(records),logicalBytes=sum(x['bytes'] for x in records),gib=sum(x['bytes'] for x in records)/1024**3)))
