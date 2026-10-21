import pathlib,json,hashlib
root=pathlib.Path.cwd();work=root/'.workspace/native-asr-migration';worker=root/'apps/desktop/resources/darwin-arm64/asr/arion-asr'
paths=[work/'arion-eval-driver',root/'models/catalog.json']+[p for p in worker.parent.rglob('*') if p.is_file()]+[p for p in (work/'model-store/qwen3-asr-1.7b-mlx-8bit').iterdir() if p.is_file()]
artifacts=[]
for i,p in enumerate(paths):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for b in iter(lambda:f.read(8*1024*1024),b''):h.update(b)
 artifacts.append({'id':str(i)+'-'+p.name,'path':str(p),'sha256':h.hexdigest(),'license':'Arion AGPL-3.0-only; MLX and MLX Audio MIT; Qwen model Apache-2.0; see pinned dependency notices'})
j={'executable':str(work/'arion-eval-driver'),'args':['-config',str(work/'native-driver.json'),'-models',str(work/'model-store')],'timeoutMs':120000,'artifacts':artifacts};(work/'native-driver.json').write_text(json.dumps(j,indent=2)+'\n')
