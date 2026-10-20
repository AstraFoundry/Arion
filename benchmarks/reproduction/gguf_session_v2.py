"""Explicit CPU/Metal session parameters for new revision-pinned diagnostics."""
import os,subprocess,time,urllib.error
import gguf_rewrite as engine
class Session:
 def __init__(self,out,name,port,*,gpu_layers,verbosity=5):
  self.port=port;self.process=None;self.log=(out/(name+'-server.log')).open('x')
  self.command=[str(engine.RUNTIME),'--model',str(engine.MODEL),'--host','127.0.0.1','--port',str(port),'--threads','4','--threads-batch','4','--n-gpu-layers',str(gpu_layers),'--ctx-size','4096','--parallel','1','--reasoning','off','--jinja','--offline','--no-warmup','-lv',str(verbosity)]
  env=dict(os.environ);env.pop('GGML_METAL_LOG_LEVEL',None);start=time.monotonic()
  try:
   self.process=subprocess.Popen(self.command,stdout=self.log,stderr=self.log,env=env)
   while True:
    if self.process.poll() is not None:raise RuntimeError('Native session exited during startup')
    try:engine.base.request(port,'/health',timeout=1);break
    except urllib.error.HTTPError as error:
     if error.code!=503:raise
    except urllib.error.URLError:pass
    if time.monotonic()-start>90:raise TimeoutError('Native session startup timeout')
    time.sleep(.1)
   self.startup_ms=(time.monotonic()-start)*1000;engine.write(out/(name+'-props.json'),engine.base.request(port,'/props'))
  except Exception:self.abort();raise
 def generate(self,request):
  start=time.monotonic();response=engine.base.request(self.port,'/v1/chat/completions',request,timeout=90);elapsed=(time.monotonic()-start)*1000
  rss=int(subprocess.check_output(['ps','-o','rss=','-p',str(self.process.pid)],text=True).strip())*1024
  return response,elapsed,rss
 def close(self):
  try:
   self.process.terminate()
   try:self.process.wait(timeout=15)
   except subprocess.TimeoutExpired:self.process.kill();self.process.wait()
  finally:self.log.close()
 def abort(self):
  if self.process and self.process.poll() is None:self.process.kill();self.process.wait()
  self.log.close()
