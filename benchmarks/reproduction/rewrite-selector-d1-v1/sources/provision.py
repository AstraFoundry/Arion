"""Provision a pinned research-only official text decision-model Q8 component; retain full receipts."""
from pathlib import Path
import json,urllib.request,hashlib,time,importlib.util
R=Path.cwd();D=Path(__file__).resolve().parent;W=D/'weights';W.mkdir(exist_ok=False)
meta=json.loads((D/'gguf-api.json').read_text());base=json.loads((R/'.workspace/notes/rewrite-additional-small-model-catalog-v4/d1-omni-600M-api.json').read_text());name='d1-omni-600M-Q8_0.gguf';entry=next(x for x in meta['siblings'] if x['rfilename']==name);expected=entry['lfs'];assert base['safetensors']['total']==587161089<=2000000000
url='https://huggingface.co/'+meta['id']+'/resolve/'+meta['sha']+'/'+name;partial=W/(name+'.part');target=W/name
status=dict(status='running',url=url,expectedSha256=expected['sha256'],expectedBytes=expected['size'],modelGenerations=0,productionActivated=False,license='custom-lfm-open-license-v1.0',authorization='Research authorized by user; production license review required.')
def save():(D/'provision-status.json').write_text(json.dumps(status,ensure_ascii=False,indent=2)+'\n')
save();h=hashlib.sha256();total=0;start=time.monotonic()
try:
 with urllib.request.urlopen(urllib.request.Request(url,headers={'User-Agent':'Arion-benchmark-provisioning/1.0'}),timeout=180) as response,partial.open('xb') as stream:
  status.update(httpStatus=response.status,headers=dict(response.headers),resolvedUrl=response.url)
  while True:
   block=response.read(8*1024*1024)
   if not block:break
   stream.write(block);h.update(block);total+=len(block)
   if total%(128*1024*1024)==0:print('downloaded',total,'of',expected['size'],flush=True)
 status.update(receivedBytes=total,actualSha256=h.hexdigest(),elapsedSeconds=time.monotonic()-start)
 if total!=expected['size'] or h.hexdigest()!=expected['sha256']:raise ValueError('Pinned official weight bytes/hash mismatch')
 spec=importlib.util.spec_from_file_location('d1_header',R/'benchmarks/reproduction/rewrite-selector-model-v1/sources/gguf_info.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);info=m.inspect(partial)
 assert info['parameters']<=base['safetensors']['total']<=2000000000
 (D/'gguf-info.json').write_text(json.dumps(info,ensure_ascii=False,indent=2)+'\n');partial.rename(target)
 status.update(status='complete',path=str(target.relative_to(R)),actualParameters=info['parameters'],tensorCount=info['tensorCount'],architecture=info['metadata']['general.architecture'],nativeCompatibility='not-tested',quality='not-tested');save();print('Official Q8 hash/header verified; zero model generations.',flush=True)
except Exception as e:
 status.update(status='failed',receivedBytes=total,partialSha256=h.hexdigest(),error=repr(e));save();raise
