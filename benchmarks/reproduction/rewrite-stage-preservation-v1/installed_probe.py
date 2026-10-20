"""Probe the immutable installed rewrite worker through its public JSONL protocol.

No internal stage attribution is possible from this installed response schema.
Input checks are evaluator-only and never copied to worker requests.
"""
import argparse,hashlib,json,pathlib,select,subprocess,time
p=argparse.ArgumentParser();p.add_argument('--out',required=True);p.add_argument('--timeout',type=float,default=240);a=p.parse_args()
base=pathlib.Path(__file__).resolve().parent; prefix=pathlib.Path(a.out)
worker=pathlib.Path('/Applications/Arion.app/Contents/Resources/llm/arion-llm');runtime=worker.parent
model=pathlib.Path('/Users/steven/Library/Application Support/Arion/models/qwen3.5-2b-q4-k-m-metal')
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
paths={k:pathlib.Path(str(prefix)+v) for k,v in {'raw':'.jsonl','stderr':'.stderr.log','summary':'.summary.json','metadata':'.metadata.json'}.items()}
for path in paths.values():
 if path.exists():raise FileExistsError(path)
raw=paths['raw'].open('x');err=paths['stderr'].open('x');counter=0
metadata={'status':'running','worker':str(worker),'workerSha256':sha(worker),'runtime':str(runtime),'model':str(model),'modelSha256':{x.name:sha(x) for x in model.iterdir() if x.is_file()},'concurrencyLimitations':'Root simultaneously probes source worker. Native inference timings are observations, not comparable isolated performance evidence.','protocolVersion':2,'stageAttribution':'Unavailable: installed worker returns delivered text/template/timings without separate spelling and formatter observations.','inputs':{str(x):sha(x) for x in [base/'user-case.json',base/'native/chatterfly/pairs.json']},'sourceSha256':sha(pathlib.Path(__file__))}
paths['metadata'].write_text(json.dumps(metadata,ensure_ascii=False,indent=2))
proc=subprocess.Popen([str(worker)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=err,text=True,bufsize=1)
def call(payload,**labels):
 global counter
 counter+=1;req=dict(version=2,id=counter,**payload);start=time.monotonic()
 raw.write(json.dumps(dict(kind='request',request=req,**labels),ensure_ascii=False)+'\n');raw.flush()
 proc.stdin.write(json.dumps(req,ensure_ascii=False)+'\n');proc.stdin.flush()
 ready,_,_=select.select([proc.stdout],[],[],a.timeout)
 if not ready:raise TimeoutError(f'Worker request {counter} exceeded {a.timeout} seconds')
 line=proc.stdout.readline();elapsed=(time.monotonic()-start)*1000
 if not line:raise RuntimeError(f'Worker stdout closed; exit={proc.poll()}')
 response=json.loads(line)
 raw.write(json.dumps(dict(kind='response',requestId=counter,response=response,rawLine=line,wallMs=elapsed,**labels),ensure_ascii=False)+'\n');raw.flush()
 if response.get('id')!=counter:raise ValueError(f'Response ID mismatch: {response}')
 return response
rows=[]
try:
 load=call(dict(op='load',runtime=str(runtime),model=str(model),threads=4,maxLength=4096,maxNewTokens=256),phase='load')
 if not load.get('ok'):raise RuntimeError(f'Load failed: {load}')
 cases=json.loads((base/'user-case.json').read_text())+json.loads((base/'native/chatterfly/pairs.json').read_text())
 for suffix,terms in [('none',[]),('mapping',['chad fly → Chatterfly']),('name-only',['Chatterfly'])]:
  cases.append(dict(id='explicit-approved-term-'+suffix,scene='prompt',preview='检查 chad fly。',final='检查 chad fly。',screen='软件名称：Chatterfly',terms=terms,pairKind='constructed-approved-mapping-control',checks=['Chatterfly']))
 for case in cases:
  responses={}
  for stage,key,live in [('preview','preview',True),('final','final',False)]:
   payload=dict(op='rewrite',session=case['id'],scene=case['scene'],text=case[key],context={'Screen':case['screen']},live=live)
   if case.get('terms'):payload['terms']=case['terms']
   responses[stage]=call(payload,caseId=case['id'],phase=stage)
  row=dict(case=case,preview=responses['preview'],final=responses['final'])
  rows.append(row)
  print(case['id'],repr(responses['preview'].get('text')), '=>',repr(responses['final'].get('text')),flush=True)
  call(dict(op='end',session=case['id']),caseId=case['id'],phase='end')
 call(dict(op='close'),phase='close');proc.wait(timeout=20)
 metadata['status']='completed';metadata['exitCode']=proc.returncode
except Exception as exc:
 metadata['status']='failed';metadata['error']=repr(exc)
 raw.write(json.dumps(dict(kind='failure',error=repr(exc)))+'\n');raw.flush()
 proc.terminate()
 try:proc.wait(timeout=15)
 except subprocess.TimeoutExpired:proc.kill();proc.wait()
 raise
finally:
 raw.close();err.close();paths['summary'].write_text(json.dumps(rows,ensure_ascii=False,indent=2))
 metadata['outputs']={str(x):sha(x) for k,x in paths.items() if k!='metadata' and x.exists()}
 paths['metadata'].write_text(json.dumps(metadata,ensure_ascii=False,indent=2))
