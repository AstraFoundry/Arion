"""Observe same-artifact Metal placement with verbose loading and no generation."""
import pathlib,sys,json,subprocess,time,os,urllib.error
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent));import gguf_rewrite as engine
ROOT=engine.ROOT
def main():
 original=ROOT/'.workspace/rewrite-metal-preflight-v1/run';out=ROOT/'.workspace/rewrite-metal-preflight-v1/load-observation';out.mkdir(parents=True,exist_ok=False);manifest=engine.read(original/'manifest.json');command=list(manifest['command']);command[command.index('--port')+1]='18800';command+=['-lv','5'];env=dict(os.environ);env.pop('GGML_METAL_LOG_LEVEL',None);process=None;state=dict(status='running',newGenerations=0,command=command,modelSha256=manifest['model']['sha256'],runtimeFiles=manifest['runtimeFiles'],scope='Separate load-only process with same artifact/provider/context/thread settings; more verbose logging, no generation. Not retrospective GPU telemetry of the original request.')
 engine.write(out/'manifest.json',state)
 try:
  with (out/'load.log').open('x') as log:
   process=subprocess.Popen(command,stdout=log,stderr=log,env=env);start=time.monotonic()
   while True:
    if process.poll() is not None:raise RuntimeError('Native verbose load exited')
    try:engine.base.request(18800,'/health',timeout=1);break
    except urllib.error.HTTPError as error:
     if error.code!=503:raise
    except urllib.error.URLError:pass
    if time.monotonic()-start>90:raise TimeoutError('Verbose Metal load timeout')
    time.sleep(.1)
   state.update(status='complete',startupMs=(time.monotonic()-start)*1000)
 finally:
  if process and process.poll() is None:process.terminate();process.wait(timeout=15)
  engine.write(out/'manifest.json',state)
if __name__=='__main__':main()
