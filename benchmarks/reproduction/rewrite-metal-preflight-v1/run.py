"""One identical-request native Metal capability probe, not a latency benchmark."""
import pathlib,sys,json,subprocess,time,os,urllib.error
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent));import gguf_rewrite as engine
ROOT=engine.ROOT
def main():
 freeze=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/freeze['modelPath'];engine.verify(freeze);out=ROOT/'.workspace/rewrite-metal-preflight-v1/run';out.mkdir(parents=True,exist_ok=False)
 ref=next(json.loads(s) for s in (ROOT/'benchmarks/runs/rewrite-formatter-v3/run/results.jsonl').read_text().splitlines() if json.loads(s)['id']=='owner-anchor');request=ref['request'];port=18799;command=[str(engine.RUNTIME),'--model',str(engine.MODEL),'--host','127.0.0.1','--port',str(port),'--threads','4','--threads-batch','4','--n-gpu-layers','99','--ctx-size','4096','--parallel','1','--reasoning','off','--jinja','--offline','--no-warmup'];manifest=dict(status='running',phase='capability-only',freezeSha256=engine.sha(HERE/'freeze.json'),model=freeze['model'],runtimeFiles=freeze['runtimeFiles'],command=command,sourceCases=1);engine.write(out/'manifest.json',manifest);process=None
 try:
  with (out/'metal-server.log').open('x') as log:
   started=time.monotonic();process=subprocess.Popen(command,stdout=log,stderr=log,env=dict(os.environ,GGML_METAL_LOG_LEVEL='0'))
   while True:
    if process.poll() is not None:raise RuntimeError('Metal native server exited')
    try:engine.base.request(port,'/health',timeout=1);break
    except urllib.error.HTTPError as error:
     if error.code!=503:raise
    except urllib.error.URLError:pass
    if time.monotonic()-started>90:raise TimeoutError('Metal health startup timeout')
    time.sleep(.1)
   manifest['startupMs']=(time.monotonic()-started)*1000;engine.write(out/'metal-props.json',engine.base.request(port,'/props'));start=time.monotonic();response=engine.base.request(port,'/v1/chat/completions',request,timeout=90);elapsed=(time.monotonic()-start)*1000
   row=dict(ref,variant='metal-capability',pipeline='metal-preflight-v1',templateVersion=ref['templateVersion'],request=request,response=response,renderedBody=response['choices'][0]['message']['content'],generationMs=elapsed,textStageMs=elapsed,preparationMs=0,renderMs=0,residentRssBytes=int(subprocess.check_output(['ps','-o','rss=','-p',str(process.pid)],text=True).strip())*1024,sourceControl=dict(run='rewrite-formatter-v3/run',variant='formatter-v3',id='owner-anchor',rowSha256=__import__('hashlib').sha256(json.dumps(ref,ensure_ascii=False,sort_keys=True).encode()).hexdigest()),scope='One already-exposed no-context identical-request capability probe on M4/32GB. Not full36 quality, warmed/cold comparison, cache acceptance, latency SLO, ASR concurrency or16GB measurement.')
   (out/'results.jsonl').write_text(json.dumps(row,ensure_ascii=False)+'\n');manifest.update(status='complete',generations=1,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',manifest);print('actual Metal body',repr(row['renderedBody']),'requestMs',elapsed,'timings',response.get('timings'),flush=True)
 except Exception as error:
  manifest.update(status='failed',errorType=type(error).__name__,error=str(error));engine.write(out/'manifest.json',manifest);raise
 finally:
  if process and process.poll() is None:
   process.terminate()
   try:process.wait(timeout=15)
   except subprocess.TimeoutExpired:process.kill();process.wait()
if __name__=='__main__':main()
