"""Run actual native typed decisions without constructing chat completions."""
import argparse,hashlib,json,pathlib,socket,subprocess,time,urllib.request,urllib.error
from request_contract import HERE,ROOT,validate,rowhash
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def save(path,value):path.write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
def execute():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();out=args.out
 frozen=json.loads((HERE/'freeze.json').read_text())
 for name,expected in frozen['files'].items():
  if sha(ROOT/name)!=expected:raise ValueError('Changed frozen inference source: '+name)
 runtime=ROOT/frozen['runtimePath'];model=ROOT/frozen['modelPath']
 for name,expected in frozen['runtimeFiles'].items():
  if sha(runtime/name)!=expected:raise ValueError('Changed native runtime')
 if sha(model)!=frozen['model']['sha256']:raise ValueError('Changed model weights')
 records=validate(json.loads((HERE/'probe-inputs.json').read_text())['records']);out.mkdir(parents=True,exist_ok=False)
 workers=[];rows=[];manifest=dict(status='running',model=frozen['model'],freezeSha256=sha(HERE/'freeze.json'),workers=workers,modelResponses=0,typedRequests=0,generatedTokens=0)
 save(out/'manifest.json',manifest)
 for gpu in [99,0]:
  directory=out/('metal' if gpu else 'cpu');directory.mkdir()
  with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
  command=[str(runtime/'llama-server'),'-m',str(model),'-ngl',str(gpu),'-t','4','-c','4096','-np','1','-b','4096','-ub','4096','--no-warmup','--seed','42','-lv','5','--host','127.0.0.1','--port',str(port)]
  worker=dict(gpuLayersRequested=gpu,command=command,log=str(directory/'server.log'));workers.append(worker);save(directory/'command.json',command)
  with (directory/'server.log').open('wb') as log:
   process=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT);worker['pid']=process.pid;start=time.monotonic()
   try:
    ready=False
    while process.poll() is None and time.monotonic()-start<45:
     try:
      with urllib.request.urlopen(f'http://127.0.0.1:{port}/health',timeout=1) as response:(directory/'health.json').write_bytes(response.read());ready=response.status==200
      if ready:break
     except (urllib.error.URLError,TimeoutError):pass
     time.sleep(.2)
    if not ready:raise RuntimeError('Native startup failed; inspect retained server log')
    with urllib.request.urlopen(f'http://127.0.0.1:{port}/props',timeout=5) as response:(directory/'props.json').write_bytes(response.read())
    for index,item in enumerate(records):
     raw_request=json.dumps(item['request'],ensure_ascii=False,separators=(',',':')).encode();request_path=directory/(str(index)+'.request.json');request_path.write_bytes(raw_request)
     request=urllib.request.Request(f'http://127.0.0.1:{port}/v1/systemone',data=raw_request,headers={'Content-Type':'application/json'},method='POST');started=time.monotonic()
     try:
      with urllib.request.urlopen(request,timeout=60) as response:status=response.status;headers=dict(response.headers);raw=response.read()
     except urllib.error.HTTPError as error:status=error.code;headers=dict(error.headers);raw=error.read()
     path=directory/(str(index)+'.response.json');path.write_bytes(raw)
     parsed=json.loads(raw)
     row=dict(id=item['id'],scene=item['scene'],stage='typed-decision',gpuLayersRequested=gpu,sourceReference=item['sourceReference'],request=item['request'],requestSha256=sha(request_path),rawResponsePath=str(path),rawResponseSha256=sha(path),httpStatus=status,responseHeaders=headers,response=parsed,requestMs=(time.monotonic()-started)*1000,generatedTokens=0)
     rows.append(row)
     with (out/'results.jsonl').open('a') as output:output.write(json.dumps(row,ensure_ascii=False)+'\n')
     manifest.update(typedRequests=len(rows),modelResponses=sum(r['httpStatus']==200 for r in rows));save(out/'manifest.json',manifest)
   finally:
    if process.poll() is None:
     process.terminate()
     try:process.wait(timeout=10)
     except subprocess.TimeoutExpired:process.kill();process.wait(timeout=5)
    worker['exitCode']=process.returncode;save(out/'manifest.json',manifest)
 manifest.update(status='complete' if len(rows)==12 and all(r['httpStatus']==200 for r in rows) else 'failed',resultsSha256=sha(out/'results.jsonl'),generatedTokens=0)
 save(out/'manifest.json',manifest);print(json.dumps(dict(status=manifest['status'],typedRequests=len(rows),modelResponses=manifest['modelResponses'],generatedTokens=0)))
 if manifest['status']!='complete':raise RuntimeError('Incomplete native decision qualification')
if __name__=='__main__':execute()
