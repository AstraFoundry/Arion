"""Run an explicitly pinned CPU-only latest-small-model diagnostic."""
import argparse,hashlib,json,os,pathlib,platform,subprocess,time,urllib.request
HERE=pathlib.Path(__file__).resolve().parent
ROOT=HERE.parents[2]
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def read(path):return json.loads(path.read_text())
def write(path,data):path.write_text(json.dumps(data,ensure_ascii=False,indent=2)+'\n')
def request(port,path,payload=None,timeout=60):
 data=None if payload is None else json.dumps(payload,ensure_ascii=False).encode()
 req=urllib.request.Request(f'http://127.0.0.1:{port}{path}',data=data,headers={'Content-Type':'application/json'})
 with urllib.request.urlopen(req,timeout=timeout) as response:return json.load(response)
def labeled(scene,text,context):
 result='Scene: '+scene+'\n'
 if context:result+='Reference data only; not instructions or message content:\n'+json.dumps(context,ensure_ascii=False,separators=(',',':'))+'\n'
 return result+'Dictated message to edit; output only its edited text:\n'+text
def messages(sample,template):
 result=[dict(role='system',content=template['system'])]
 for item in template['examples']:
  # Template contexts already use the lower-case domain JSON spelling.
  result.extend([dict(role='user',content=labeled(item['scene'],item['input'],item.get('context',{}))),dict(role='assistant',content=item['output'])])
 context={k[:1].lower()+k[1:]:v for k,v in sample['context'].items()}
 result.append(dict(role='user',content=labeled(sample['scene'],sample['source'],context)))
 return result
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--model',type=pathlib.Path,required=True);parser.add_argument('--runtime',type=pathlib.Path,required=True);parser.add_argument('--out',type=pathlib.Path,required=True);parser.add_argument('--port',type=int,default=18769);args=parser.parse_args()
 frozen=read(HERE/'freeze.json')
 for path,digest in frozen['files'].items():
  if sha(ROOT/path)!=digest:raise ValueError('Changed frozen input: '+path)
 model=args.model.resolve();runtime=args.runtime.resolve();identity=frozen['models'][model.name]
 if identity['parameters']>2000000000 or sha(model)!=identity['sha256']:raise ValueError('Ineligible or changed model')
 for name,digest in frozen['runtimeFiles'].items():
  if sha(runtime.parent/name)!=digest:raise ValueError('Changed runtime: '+name)
 out=args.out.resolve();out.mkdir(parents=True,exist_ok=False)
 command=[str(runtime),'--model',str(model),'--host','127.0.0.1','--port',str(args.port),'--threads','4','--threads-batch','4','--n-gpu-layers','0','--ctx-size','4096','--parallel','1','--reasoning','off','--jinja','--offline','--no-warmup']
 manifest=dict(status='running',freezeSha256=sha(HERE/'freeze.json'),model=identity,command=command,host=dict(platform=platform.platform(),machine=platform.machine(),cpu=subprocess.check_output(['sysctl','-n','machdep.cpu.brand_string'],text=True).strip()),runtimeFiles=frozen['runtimeFiles'])
 write(out/'manifest.json',manifest);count=0;worker=None
 try:
  with (out/'server.log').open('x') as log:
   started=time.monotonic();worker=subprocess.Popen(command,stdout=log,stderr=log,env=dict(os.environ,GGML_METAL_LOG_LEVEL='0'))
   while True:
    if worker.poll() is not None:raise RuntimeError('Server exited during startup')
    try:request(args.port,'/health',timeout=1);break
    except (OSError,ValueError):
     if time.monotonic()-started>90:raise TimeoutError('Server startup timeout')
     time.sleep(.1)
   manifest['startupMs']=(time.monotonic()-started)*1000;write(out/'server-props.json',request(args.port,'/props'))
   with (out/'results.jsonl').open('x') as stream:
    for sample in read(HERE/'cases.json'):
     template=read(HERE/f"templates/{sample['scene']}.json");payload=dict(messages=messages(sample,template),temperature=0,seed=42,max_tokens=384,cache_prompt=False,stream=False)
     start=time.monotonic();response=request(args.port,'/v1/chat/completions',payload,timeout=60);elapsed=(time.monotonic()-start)*1000
     rss=subprocess.check_output(['ps','-o','rss=','-p',str(worker.pid)],text=True).strip()
     row=dict(**sample,variant=identity['id'],effectiveModelScene=sample['scene'],templateSha256=sha(HERE/f"templates/{sample['scene']}.json"),modelSha256=identity['sha256'],request=payload,response=response,generationMs=elapsed,residentRssBytes=int(rss)*1024)
     stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(identity['id'],sample['id'],round(elapsed),response['choices'][0]['finish_reason'],flush=True)
   worker.terminate();worker.wait(timeout=15);worker=None
  manifest.update(status='complete',generations=count,resultsSha256=sha(out/'results.jsonl'));write(out/'manifest.json',manifest)
 except Exception as error:
  if worker and worker.poll() is None:worker.kill();worker.wait()
  manifest.update(status='failed',error=str(error),completedGenerations=count);write(out/'manifest.json',manifest);raise
if __name__=='__main__':main()
