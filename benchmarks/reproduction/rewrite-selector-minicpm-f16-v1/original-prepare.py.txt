"""Prefreeze the F16 export comparison without changing historical records."""
from pathlib import Path
import json,hashlib,shutil,subprocess,importlib.util
R=Path.cwd();H=R/'benchmarks/reproduction/rewrite-selector-minicpm-f16-v1';W=R/'.workspace/rewrite-selector-minicpm-f16-v1';P=R/'.workspace/rewrite-minicpm5-f16-v1/provisioning'
def read(p):return json.loads(p.read_text())
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def save(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
assert read(P/'status.json')['status']=='complete'
H.mkdir(exist_ok=False);(H/'sources').mkdir()
for p in P.iterdir():
 if p.is_file():shutil.copyfile(p,H/'sources'/p.name)
shutil.copyfile(P/'gguf-info.json',H/'sources/gguf-header-inspection.json')
shutil.copyfile(Path(__file__),H/'original-prepare.py.txt')
for name,target,call in [('preflight.py','rewrite-selector-hybrid-v1/preflight.py','main'),('run.py','rewrite-selector-model-v1/run.py','execute')]:
 (H/name).write_text('"""Execute immutable native comparison machinery with the owned F16 freeze."""\nimport importlib.util\nimport pathlib\nHERE=pathlib.Path(__file__).resolve().parent\nspec=importlib.util.spec_from_file_location("minicpm_f16_executor",HERE.parent/'+repr(target)+')\nexecutor=importlib.util.module_from_spec(spec)\nspec.loader.exec_module(executor)\nif __name__=="__main__":\n    executor.HERE=HERE\n    executor.'+call+'()\n')
plan=read(R/'benchmarks/reproduction/rewrite-selector-minicpm-v1/preflight-plan.json');plan['gate']='All six Metal requests have valid JSON/normal stop, pinned actual template after exactly one native-lexer final-LF removal, actual25/25GPU layers, explicit nonthinking prefix, and exact raw positive cache-off/on/off parity. CPU0 diagnostics retained separately. Passing permits202 unchanged requests even if semantic probes fail; semantic failures never become compatibility or release credit.';save(H/'preflight-plan.json',plan)
protocol=read(R/'benchmarks/reproduction/rewrite-selector-minicpm-v1/protocol.json');protocol['purpose']='Compare pinned official MiniCPM5-1B F16 export to prior actual Q4, holding complete messages/candidates/runtime/decoding fixed, with12 native probes before202 unchanged requests.';protocol['modelComparison']='Precision/export artifact comparison only: shared upstream and official revision, identical template/tokenizer expected, but no numerical conversion/quantizer lineage equivalence has been established. Do not attribute every change solely to quantization.';protocol['priorQ4Reference']='benchmarks/runs/rewrite-selector-minicpm-v1/comparison-run/results.jsonl';protocol['priorQ4ProbeReference']='benchmarks/runs/rewrite-selector-minicpm-v1/preflight/results.jsonl';save(H/'protocol.json',protocol)
f=read(R/'benchmarks/reproduction/rewrite-selector-minicpm-v1/freeze.json');f['files']=dict(read(R/'benchmarks/runs/rewrite-selector-minicpm-v1/archive.json')['files']);f['modelPath']='.workspace/rewrite-minicpm5-f16-v1/weights/MiniCPM5-1B-F16.gguf';f['model']=dict(f['model'],id='MiniCPM5-1B-F16',filename='MiniCPM5-1B-F16.gguf',quantization='F16',sha256='68c40b08b1242754a107b9510af89aa75b10c75843ca7844643c70956b7f1e3d',bytes=2166551936)
info=read(H/'sources/gguf-header-inspection.json');assert info['parameters']==f['model']['parameters']==1080632832 and info['tensorCount']==219
q4=read(R/'benchmarks/reproduction/rewrite-selector-minicpm-v1/sources/gguf-header-inspection.json');assert info['metadata']['tokenizer.chat_template']==q4['metadata']['tokenizer.chat_template'];assert {x['name']:x['shape'] for x in info['tensors']}=={x['name']:x['shape'] for x in q4['tensors']}
spec=importlib.util.spec_from_file_location('f16_metadata_identity',P/'metadata_identity.py');metadata=importlib.util.module_from_spec(spec);spec.loader.exec_module(metadata)
left=metadata.identities(R/'.workspace/rewrite-minicpm5-v1/weights/MiniCPM5-1B-Q4_K_M.gguf');right=metadata.identities(R/f['modelPath']);assert left==right;save(H/'sources/complete-metadata-identity.json',dict(identical=True,fields=left,modelGenerations=0,scope='Exact binary metadata payload identity, including all tokenizer arrays; no tensor-value equivalence claim.'))
embedded=info['metadata']['tokenizer.chat_template'];assert embedded.endswith('\n');save(H/'sources/native-template-policy.json',dict(metadata={'tokenizer.chat_template':embedded[:-1]},embeddedTemplateSha256=hashlib.sha256(embedded.encode()).hexdigest(),transformation='Exactly one final LF removed by pinned f3fb2fcdb lexer; derived expectation, not replacement template.'))
save(H/'sources/artifact-comparison.json',dict(identicalTensorNamesAndShapes=True,tensorCount=219,parameters=1080632832,identicalEmbeddedTemplate=True,q4TensorTypes=sorted({t['type'] for t in q4['tensors']}),f16TensorTypes=sorted({t['type'] for t in info['tensors']}),numericalWeightLineageEstablished=False,modelGenerations=0))
shutil.copyfile(W/'report-source.py',H/'report.py')
f['files']['benchmarks/reproduction/gguf_native_probe_evidence_v2.py']=sha(R/'benchmarks/reproduction/gguf_native_probe_evidence_v2.py')
for p in H.rglob('*'):
 if p.is_file() and '__pycache__' not in p.parts:f['files'][str(p.relative_to(R))]=sha(p)
save(H/'freeze.json',f)
host={}
for key,cmd in [('os',['sw_vers']),('hardware',['sysctl','hw.model','hw.memsize','hw.physicalcpu','machdep.cpu.brand_string']),('power',['pmset','-g','custom']),('thermal',['pmset','-g','therm']),('gitRevision',['git','rev-parse','HEAD']),('dirty',['git','status','--porcelain']),('runtime',['.workspace/rewrite-small-models-v1/runtime/llama-b11734/llama-server','--version'])]:
 p=subprocess.run(cmd,capture_output=True,text=True);host[key]=dict(command=cmd,exitCode=p.returncode,stdout=p.stdout,stderr=p.stderr)
host.update(timer='time.monotonic',freezeSha256=sha(H/'freeze.json'),scope='M4/32GB development evidence, not actual16GB/ASR contention/native target acceptance.')
save(W/'host-provenance.json',host);print('F16 prefrozen',len(f['files']),sha(H/'freeze.json'))
