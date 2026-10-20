"""Provision the pinned official eligible1B GGUF; never download the over-budget2B."""
from pathlib import Path
import json,urllib.request,hashlib,time,importlib.util,datetime
R=Path.cwd();D=R/'.workspace/rewrite-minicpm5-v1/provisioning';W=R/'.workspace/rewrite-minicpm5-v1/weights';W.mkdir(exist_ok=True)
meta=json.loads((D/'MiniCPM5-1B-GGUF-api.json').read_text());base=json.loads((D/'MiniCPM5-1B-api.json').read_text());large=json.loads((D/'MiniCPM5-2B-api.json').read_text())
assert meta['cardData']['license']==base['cardData']['license']=='apache-2.0' and base['safetensors']['total']<=2_000_000_000 and large['safetensors']['total']>2_000_000_000
name='MiniCPM5-1B-Q4_K_M.gguf';entry=next(s for s in meta['siblings'] if s['rfilename']==name);expected=entry['lfs'];url='https://huggingface.co/'+meta['id']+'/resolve/'+meta['sha']+'/'+name;path=W/name;partial=W/(name+'.part')
assert not path.exists() and not partial.exists()
status=dict(status='running',url=url,expectedSha256=expected['sha256'],expectedBytes=expected['size'],modelGenerations=0,excludedModel=dict(id=large['id'],declaredTotalParameters=large['safetensors']['total'],reason='Exceeds user2B limit; zero weight downloads or inference'))
(D/'download-status.json').write_text(json.dumps(status,ensure_ascii=False,indent=2)+'\n');digest=hashlib.sha256();total=0;start=time.monotonic()
try:
 request=urllib.request.Request(url,headers={'User-Agent':'Arion-benchmark-provisioning/1.0'})
 with urllib.request.urlopen(request,timeout=180) as response,partial.open('xb') as stream:
  status.update(httpStatus=response.status,headers=dict(response.headers),resolvedUrl=response.url)
  while True:
   chunk=response.read(8*1024*1024)
   if not chunk:break
   stream.write(chunk);digest.update(chunk);total+=len(chunk)
   if total%(64*1024*1024)==0:print('downloaded',total,'of',expected['size'],flush=True)
 status.update(receivedBytes=total,actualSha256=digest.hexdigest(),downloadElapsedSeconds=time.monotonic()-start)
 if total!=expected['size'] or digest.hexdigest()!=expected['sha256']:raise ValueError('Official GGUF bytes/hash mismatch')
 spec=importlib.util.spec_from_file_location('minicpm_header_inspector',R/'benchmarks/reproduction/rewrite-selector-model-v1/sources/gguf_info.py');inspector=importlib.util.module_from_spec(spec);spec.loader.exec_module(inspector);info=inspector.inspect(partial)
 (D/'gguf-info.json').write_text(json.dumps(info,ensure_ascii=False,indent=2)+'\n')
 assert info['parameters']==base['safetensors']['total']<=2_000_000_000
 tokenizer=json.loads((D/'MiniCPM5-1B-tokenizer_config.json').read_text());embedded=info['metadata']['tokenizer.chat_template'];assert embedded==tokenizer['chat_template']
 partial.rename(path)
 status.update(status='complete',path=str(path.relative_to(R)),actualParameters=info['parameters'],tensorCount=info['tensorCount'],architecture=info['metadata']['general.architecture'],actualTemplateMatchesPinnedUpstream=True,modelGenerations=0,retrievedAt=datetime.datetime.now(datetime.timezone.utc).isoformat())
 (D/'download-status.json').write_text(json.dumps(status,ensure_ascii=False,indent=2)+'\n');print(json.dumps({k:v for k,v in status.items() if k not in ['headers','resolvedUrl']},ensure_ascii=False),flush=True)
except Exception as error:
 status.update(status='failed',receivedBytes=total,partialSha256=digest.hexdigest(),error=repr(error),modelGenerations=0);(D/'download-status.json').write_text(json.dumps(status,ensure_ascii=False,indent=2)+'\n');raise
