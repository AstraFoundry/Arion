"""Qualify existing verified bytes using the pinned separate upstream template file."""
from pathlib import Path
import json,hashlib,shutil,importlib.util,datetime
R=Path.cwd();D=R/'.workspace/rewrite-minicpm5-v1/provisioning';W=R/'.workspace/rewrite-minicpm5-v1/weights';initial=json.loads((D/'download-status.json').read_text())
assert initial['status']=='failed' and initial['error']=="KeyError('chat_template')" and initial['receivedBytes']==initial['expectedBytes'] and initial['actualSha256']==initial['expectedSha256']
shutil.copyfile(D/'download.py',D/'download-first-attempt.py.txt');shutil.copyfile(D/'download-status.json',D/'download-first-attempt-status.json')
source=W/'MiniCPM5-1B-Q4_K_M.gguf.part';target=W/'MiniCPM5-1B-Q4_K_M.gguf';assert source.exists() and not target.exists()
sha=hashlib.sha256()
with source.open('rb') as stream:
 for chunk in iter(lambda:stream.read(8*1024*1024),b''):sha.update(chunk)
assert sha.hexdigest()==initial['expectedSha256'] and source.stat().st_size==initial['expectedBytes']
spec=importlib.util.spec_from_file_location('qualify_minicpm_header',R/'benchmarks/reproduction/rewrite-selector-model-v1/sources/gguf_info.py');inspector=importlib.util.module_from_spec(spec);spec.loader.exec_module(inspector);info=inspector.inspect(source)
base=json.loads((D/'MiniCPM5-1B-api.json').read_text());template_path=D/'MiniCPM5-1B-chat_template.jinja';template=template_path.read_text()
assert info['parameters']==base['safetensors']['total']==1_080_632_832<=2_000_000_000 and info['metadata']['general.architecture']=='llama'
assert info['metadata']['tokenizer.chat_template']==template
source.rename(target)
result=dict(status='complete',path=str(target.relative_to(R)),bytes=target.stat().st_size,sha256=sha.hexdigest(),actualParameters=info['parameters'],tensorCount=info['tensorCount'],architecture=info['metadata']['general.architecture'],upstreamRevision=base['sha'],upstreamTemplatePath=template_path.name,upstreamTemplateSha256=hashlib.sha256(template_path.read_bytes()).hexdigest(),actualEmbeddedTemplateMatchesPinnedUpstream=True,correctedInspectionAssumption='Pinned upstream stores its chat template in chat_template.jinja, not in tokenizer_config.json. Original metadata request receipts, failed script/status/log and verified download bytes are retained; no redownload or template substitution.',modelGenerations=0,nativeRuntimeCompatibility='not-tested',modelQuality='not-tested',productionActivated=False,retrievedAt=datetime.datetime.now(datetime.timezone.utc).isoformat())
(D/'qualification-status.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))
