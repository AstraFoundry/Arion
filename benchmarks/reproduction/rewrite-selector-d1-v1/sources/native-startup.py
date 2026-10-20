"""Observe pinned text-decision model startup without warmup or inference requests."""
from pathlib import Path
import hashlib,json,socket,subprocess,time,urllib.request,urllib.error
ROOT=Path.cwd();W=Path(__file__).resolve().parent;F=json.loads((W/'startup-freeze.json').read_text());OUT=W/'native-startup'
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def save(p,x):p.write_text(json.dumps(x,ensure_ascii=False,indent=2)+'\n')
for name,expected in F['files'].items():
 if sha(ROOT/name)!=expected:raise ValueError('Changed preregistered startup source: '+name)
if sha(ROOT/F['modelPath'])!=F['modelSha256']:raise ValueError('Changed model')
OUT.mkdir(exist_ok=False);results=[]
for gpu in [99,0]:
 with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
 d=OUT/('metal' if gpu else 'cpu');d.mkdir()
 cmd=[str(ROOT/F['runtimePath']/'llama-server'),'-m',str(ROOT/F['modelPath']),'-ngl',str(gpu),'-t','4','-c','4096','-np','1','-b','4096','-ub','4096','--no-warmup','--host','127.0.0.1','--port',str(port)]
 save(d/'command.json',cmd);history=[];ready=False
 with (d/'server.log').open('wb') as log:
  process=subprocess.Popen(cmd,stdout=log,stderr=subprocess.STDOUT);save(d/'pid.json',dict(pid=process.pid));start=time.monotonic()
  try:
   while process.poll() is None and time.monotonic()-start<45:
    try:
     with urllib.request.urlopen(f'http://127.0.0.1:{port}/health',timeout=1) as response:
      raw=response.read();(d/'health.json').write_bytes(raw);history.append(dict(endpoint='/health',status=response.status));ready=True;break
    except (urllib.error.URLError,TimeoutError) as error:history.append(dict(endpoint='/health',error=str(error)))
    time.sleep(.2)
   if ready:
    try:
     with urllib.request.urlopen(f'http://127.0.0.1:{port}/props',timeout=5) as response:(d/'props.json').write_bytes(response.read());history.append(dict(endpoint='/props',status=response.status))
    except (urllib.error.URLError,TimeoutError) as error:history.append(dict(endpoint='/props',error=str(error)))
  finally:
   if process.poll() is None:
    process.terminate()
    try:process.wait(timeout=10)
    except subprocess.TimeoutExpired:process.kill();process.wait(timeout=5)
  result=dict(gpuLayersRequested=gpu,ready=ready,exitCode=process.returncode,ownedProcessTerminated=True,modelResponses=0,generatedTokens=0,warmupRequested=False,httpHistory=history)
  save(d/'status.json',result);results.append(result)
save(OUT/'summary.json',dict(status='complete',results=results,modelResponses=0,generatedTokens=0,scope='Only native startup and read-only health/props. No typed decision, chat completion, template serialization or quality test. Timing under concurrent evidence-check processes is not acceptance evidence.'))
print(json.dumps(dict(status='complete',ready=sum(x['ready'] for x in results),modelResponses=0,generatedTokens=0)))
