"""Freeze the eligible MiniCPM native probes and conditional unchanged comparison."""
from pathlib import Path
import hashlib,json,shutil,subprocess
R=Path.cwd(); H=R/'benchmarks/reproduction/rewrite-selector-minicpm-v1'; W=R/'.workspace/rewrite-selector-minicpm-v1'
def read(p): return json.loads(p.read_text())
def sha(p): return hashlib.sha256(p.read_bytes()).hexdigest()
def save(p,x): p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
H.mkdir(exist_ok=False); (H/'sources').mkdir()
P=R/'.workspace/rewrite-minicpm5-v1/provisioning'
for p in P.iterdir():
 if p.is_file(): shutil.copyfile(p,H/'sources'/p.name)
shutil.copyfile(P/'gguf-info.json',H/'sources/gguf-header-inspection.json')
shutil.copyfile(Path(__file__),H/'original-prepare.py.txt')
for name,target,call in [('preflight.py','rewrite-selector-hybrid-v1/preflight.py','main'),('run.py','rewrite-selector-model-v1/run.py','execute')]:
 (H/name).write_text('"""Run unchanged comparison machinery with owned MiniCPM evidence."""\nimport importlib.util\nimport pathlib\nHERE=pathlib.Path(__file__).resolve().parent\nspec=importlib.util.spec_from_file_location("minicpm_owned_executor", HERE.parent/'+repr(target)+')\nexecutor=importlib.util.module_from_spec(spec)\nspec.loader.exec_module(executor)\nif __name__=="__main__":\n    executor.HERE=HERE\n    executor.'+call+'()\n')
plan=read(R/'benchmarks/reproduction/rewrite-selector-hybrid-v1/preflight-plan.json')
plan['gate']='All six Metal requests complete with valid JSON keep/replace and normal stop, pinned actual embedded template, exact positive cache-off/on/off raw parity. Twelve attempts across Metal99 and CPU0; task correctness is separately reviewed and not required for runtime qualification. CPU is diagnostic, never automatic fallback. Passing only permits the unchanged 202-choice comparison.'
save(H/'preflight-plan.json',plan)
protocol=read(R/'benchmarks/reproduction/rewrite-selector-hybrid-v1/protocol.json')
protocol['purpose']='Evaluate eligible actual1.08B MiniCPM5-1B Q4_K_M with pinned native template: 12 native protocol/cache probes, then conditional 202 unchanged baseline/balanced requests.'
protocol['nativeProbeCount']=12
protocol['modelComparison']='Model, architecture, tokenizer and template change together; original complete request messages and decoding settings stay unchanged. No checkpoint-only causal attribution.'
protocol['userCashPolicy']='read cash may normalize to readCash, but cannot change without evidence to readCache; unchanged frozen evaluator remains authoritative.'
save(H/'protocol.json',protocol)
f=read(R/'benchmarks/reproduction/rewrite-selector-hybrid-v1/freeze.json')
f['modelPath']='.workspace/rewrite-minicpm5-v1/weights/MiniCPM5-1B-Q4_K_M.gguf'
f['model']=dict(id='MiniCPM5-1B',repo='openbmb/MiniCPM5-1B-GGUF',revision='3d55fac80935ae6456986ad2384b5cbcc4d6c948',filename='MiniCPM5-1B-Q4_K_M.gguf',sha256='81b64d05a23b17b34c475f42b3e72fbde62d4b92cc34541f7a8031d0752deafa',bytes=688065920,parameters=1080632832,license='apache-2.0',textOnly=True,quantization='Q4_K_M',upstreamRepo='openbmb/MiniCPM5-1B',upstreamRevision='87179e5c1f455ef22e6223592d2d61351b525bfc')
for p in [R/'benchmarks/reproduction/rewrite-selector-hybrid-v1/preflight.py',R/'benchmarks/reproduction/rewrite-selector-model-v1/run.py']:
 f['files'][str(p.relative_to(R))]=sha(p)
for p in H.rglob('*'):
 if p.is_file() and '__pycache__' not in p.parts: f['files'][str(p.relative_to(R))]=sha(p)
save(H/'freeze.json',f)
host={}
for key,cmd in [('os',['sw_vers']),('hardware',['sysctl','hw.model','hw.memsize','hw.physicalcpu','machdep.cpu.brand_string']),('power',['pmset','-g','custom']),('thermal',['pmset','-g','therm']),('gitRevision',['git','rev-parse','HEAD']),('dirty',['git','status','--porcelain']),('runtime',['.workspace/rewrite-small-models-v1/runtime/llama-b11734/llama-server','--version'])]:
 p=subprocess.run(cmd,capture_output=True,text=True); host[key]=dict(command=cmd,exitCode=p.returncode,stdout=p.stdout,stderr=p.stderr)
host.update(timer='time.monotonic',freezeSha256=sha(H/'freeze.json'),scope='M4/32GB native diagnostic, not Apple Silicon16GB acceptance. No new ASR/OCR/final body or production change.')
save(W/'host-provenance.json',host)
print('Frozen',len(f['files']),'files',sha(H/'freeze.json'))
