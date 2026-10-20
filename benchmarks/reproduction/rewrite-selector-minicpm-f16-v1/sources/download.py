"""Provision the pinned eligible official F16 artifact with original-byte receipts."""
from pathlib import Path
import json,urllib.request,hashlib,time,importlib.util,datetime,shutil
R=Path.cwd();D=Path(__file__).resolve().parent;W=D.parent/'weights';W.mkdir(exist_ok=False);S=R/'benchmarks/reproduction/rewrite-selector-minicpm-v1/sources'
meta=json.loads((S/'MiniCPM5-1B-GGUF-api.json').read_text());base=json.loads((S/'MiniCPM5-1B-api.json').read_text());name='MiniCPM5-1B-F16.gguf';entry=next(x for x in meta['siblings'] if x['rfilename']==name);expected=entry['lfs'];assert base['safetensors']['total']==1080632832<=2000000000 and base['cardData']['license']=='apache-2.0'
url='https://huggingface.co/'+meta['id']+'/resolve/'+meta['sha']+'/'+name;partial=W/(name+'.part');target=W/name;status=dict(status='running',url=url,expectedSha256=expected['sha256'],expectedBytes=expected['size'],modelGenerations=0,modelParameters=base['safetensors']['total'])
def save(): (D/'status.json').write_text(json.dumps(status,ensure_ascii=False,indent=2)+'\n')
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
 spec=importlib.util.spec_from_file_location('f16_header',R/'benchmarks/reproduction/rewrite-selector-model-v1/sources/gguf_info.py');m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);info=m.inspect(partial)
 assert info['parameters']==base['safetensors']['total'] and info['tensorCount']==219
 template=(S/'MiniCPM5-1B-chat_template.jinja').read_text();assert info['metadata']['tokenizer.chat_template']==template
 (D/'gguf-info.json').write_text(json.dumps(info,ensure_ascii=False,indent=2)+'\n');partial.rename(target)
 for name in ['MiniCPM5-1B-GGUF-api.json','MiniCPM5-1B-api.json','MiniCPM5-1B-chat_template.jinja']:shutil.copyfile(S/name,D/name)
 status.update(status='complete',path=str(target.relative_to(R)),actualParameters=info['parameters'],tensorCount=info['tensorCount'],architecture=info['metadata']['general.architecture'],actualEmbeddedTemplateMatchesPinnedUpstream=True,nativeCompatibility='not-tested',quality='not-tested',productionActivated=False,retrievedAt=datetime.datetime.now(datetime.timezone.utc).isoformat());save();print('Official F16 weight bytes/hash/header verified; zero generations.',flush=True)
except Exception as e:
 status.update(status='failed',receivedBytes=total,partialSha256=h.hexdigest(),error=repr(e));save();raise
