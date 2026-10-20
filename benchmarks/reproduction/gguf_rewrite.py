"""Shared pinned local GGUF session operations for follow-up diagnostics."""
import importlib.util,os,pathlib,subprocess,time,urllib.error
ROOT=pathlib.Path(__file__).resolve().parents[2]
spec=importlib.util.spec_from_file_location('small_gguf_base',pathlib.Path(__file__).parent/'rewrite-small-models-v1/run.py')
base=importlib.util.module_from_spec(spec);spec.loader.exec_module(base)
read=base.read;write=base.write;sha=base.sha;messages=base.messages
MODEL=ROOT/'.workspace/rewrite-small-models-v1/Qwen3.5-0.8B-Q4_K_M.gguf'
RUNTIME=ROOT/'.workspace/rewrite-small-models-v1/runtime/llama-b11734/llama-server'
def verify(freeze):
 for path,digest in freeze['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen input: '+path)
 if sha(MODEL)!=freeze['model']['sha256'] or freeze['model']['parameters']>2000000000:raise ValueError('Changed or ineligible model')
 for path,digest in freeze['runtimeFiles'].items():
  if sha(RUNTIME.parent/path)!=digest:raise ValueError('Changed runtime: '+path)
class Session:
 def __init__(self,out,name,port):
  self.log=(out/(name+'-server.log')).open('x');self.port=port;self.process=None
  self.command=[str(RUNTIME),'--model',str(MODEL),'--host','127.0.0.1','--port',str(port),'--threads','4','--threads-batch','4','--n-gpu-layers','0','--ctx-size','4096','--parallel','1','--reasoning','off','--jinja','--offline','--no-warmup']
  start=time.monotonic()
  try:
   self.process=subprocess.Popen(self.command,stdout=self.log,stderr=self.log,env=dict(os.environ,GGML_METAL_LOG_LEVEL='0'))
   while True:
    if self.process.poll() is not None:raise RuntimeError('Native server exited during startup')
    try:base.request(port,'/health',timeout=1);break
    except urllib.error.HTTPError as error:
     if error.code!=503:raise
    except urllib.error.URLError:pass
    if time.monotonic()-start>90:raise TimeoutError('Native server startup timeout')
    time.sleep(.1)
   self.startup_ms=(time.monotonic()-start)*1000;write(out/(name+'-props.json'),base.request(port,'/props'))
  except Exception:
   self.abort();raise
 def generate(self,sample,template,cache):
  payload=dict(messages=messages(sample,template),temperature=0,seed=42,max_tokens=384,cache_prompt=cache,stream=False)
  start=time.monotonic();response=base.request(self.port,'/v1/chat/completions',payload,timeout=60);elapsed=(time.monotonic()-start)*1000
  rss=int(subprocess.check_output(['ps','-o','rss=','-p',str(self.process.pid)],text=True).strip())*1024
  return dict(request=payload,response=response,generationMs=elapsed,residentRssBytes=rss)
 def close(self):
  try:
   self.process.terminate();self.process.wait(timeout=15)
  finally:
   if self.process.poll() is None:self.process.kill();self.process.wait()
   self.log.close()
 def abort(self):
  if self.process and self.process.poll() is None:self.process.kill();self.process.wait()
  self.log.close()
