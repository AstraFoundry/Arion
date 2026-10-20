"""Reproduce fixed native piece budgets with actual tokenizer calls, never inference."""
import argparse,hashlib,json,pathlib,re,socket,subprocess,time,urllib.request,urllib.error
import jinja2
import sys
sys.path.insert(0,str(pathlib.Path(__file__).resolve().parent.parent/'rewrite-selector-d1-v1'))
from request_contract import rowhash
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parents[2]
SEP='<<d1omni:sep>>';STATE='<<d1omni:state>>';QUESTION='<<d1omni:question>>';OPTION='<<d1omni:option>>'
def sha(path):return hashlib.sha256(path.read_bytes()).hexdigest()
def save(path,value):path.write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
def escape(value):
 if isinstance(value,str):return re.sub(r'<\|([A-Za-z0-9_]+)\|>',lambda m:'<¦'+m[1]+'¦>',value).replace('<<d1omni:','<<d1omni ')
 if isinstance(value,list):return [escape(v) for v in value]
 if isinstance(value,dict):return {escape(k):escape(v) for k,v in value.items()}
 return value

def render(request):
 template=json.loads((HERE.parent/'rewrite-selector-d1-v1/sources/gguf-info.json').read_text())['metadata']['tokenizer.chat_template.systemone']
 identity,question=next(iter(request['questions'].items()))
 inp=escape(dict(id=identity,type=question['type'],instructions=question['instructions'],state=request['state'],options=[dict(key=k,description=v) for k,v in question['criteria'].items()]))
 inp.update(audio=False,sep=SEP,mark_state=STATE,mark_question=QUESTION,mark_option=OPTION,images=[])
 environment=jinja2.Environment();environment.filters['tojson']=lambda value:json.dumps(value,ensure_ascii=False)
 return environment.from_string(template).render(**inp)

def execute():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);args=parser.parse_args();out=args.out;f=json.loads((HERE/'freeze.json').read_text())
 for path,expected in f['files'].items():
  if sha(ROOT/path)!=expected:raise ValueError('Changed budget source: '+path)
 runtime=ROOT/f['runtimePath'];model=ROOT/f['modelPath']
 for name,expected in f['runtimeFiles'].items():
  if sha(runtime/name)!=expected:raise ValueError('Changed actual tokenizer runtime')
 if sha(model)!=f['model']['sha256']:raise ValueError('Changed actual tokenizer weights')
 out.mkdir(parents=True,exist_ok=False);groups=[]
 for study,kind in [('rewrite-selector-d1-v1',''),('rewrite-selector-d1-layout-v1','structured'),('rewrite-selector-d1-layout-v1','sentence'),('rewrite-selector-d1-compact-v1','structured'),('rewrite-selector-d1-compact-v1','sentence')]:
  root=ROOT/'.workspace'/study/'preflight';directory=root/kind if kind else root
  rows=list(map(json.loads,(directory/'results.jsonl').read_text().splitlines()));assert len(rows)==12
  logged=[]
  for backend in ['metal','cpu']:
   prompts=[]
   for line in (directory/backend/'server.log').read_bytes().splitlines():
    if b'launching slot : ' in line:
     slot=json.loads(line.split(b'launching slot : ',1)[1]);value=slot.get('prompt')
     if value:prompts.append(value)
   if len(prompts)!=5:raise ValueError('Changed observed native prompt scope')
   logged.append(prompts)
  groups.append(dict(name=study+('/'+kind if kind else ''),records=[dict(id=r['id'],request=r['request'],actualRows=[rows[i],rows[i+6]],nativeLoggedPrompts=[v[i] for v in logged] if i<5 else []) for i,r in enumerate(rows[:6])]))
 groups.append(dict(name='policy',records=json.loads((HERE/'probe-inputs.json').read_text())['records']))
 with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
 command=[str(runtime/'llama-server'),'-m',str(model),'-ngl','99','-t','4','-c','4096','-np','1','-b','4096','-ub','4096','--no-warmup','-lv','5','--host','127.0.0.1','--port',str(port)];save(out/'command.json',command)
 def call(endpoint,payload):
  request=urllib.request.Request(f'http://127.0.0.1:{port}'+endpoint,data=json.dumps(payload,ensure_ascii=False).encode(),headers={'Content-Type':'application/json'},method='POST')
  with urllib.request.urlopen(request,timeout=20) as response:raw=response.read();value=json.loads(raw)
  with (out/'native-calls.jsonl').open('a') as trace:trace.write(json.dumps(dict(endpoint=endpoint,request=payload,response=value,rawResponse=raw.decode()),ensure_ascii=False)+'\n')
  return value
 records=[]
 with (out/'server.log').open('wb') as log:
  process=subprocess.Popen(command,stdout=log,stderr=subprocess.STDOUT);start=time.monotonic()
  try:
   ready=False
   while process.poll() is None and time.monotonic()-start<45:
    try:
     with urllib.request.urlopen(f'http://127.0.0.1:{port}/health',timeout=1) as response:ready=response.status==200
     if ready:break
    except (urllib.error.URLError,TimeoutError):pass
    time.sleep(.2)
   if not ready:raise RuntimeError('Native tokenization startup failed')
   for group in groups:
    for item in group['records']:
     request=item['request'];raw=render(request);parts=raw.split(SEP);pieces=[];tokens=[]
     if parts[0]!='':raise ValueError('Unexpected native media prefix')
     for text in parts[1:]:
      kind='other';limit=None
      for marker,label,budget in [(STATE,'state',None),(QUESTION,'question',96),(OPTION,'option',45)]:
       if text.startswith(marker):text=text[len(marker):];kind=label;limit=budget;break
      ids=call('/tokenize',dict(content=text,add_special=False,parse_special=True,with_pieces=False))['tokens'];used=ids if limit is None else ids[:limit];tokens+=used
      pieces.append(dict(kind=kind,text=text,tokens=ids,usedTokens=used,limit=limit,truncated=len(used)!=len(ids)))
     detokenized=call('/detokenize',dict(tokens=tokens))['content']
     if len(tokens)>4096:raise ValueError('Full input exceeds declared context/batch')
     references=item.get('actualRows',[])
     if references and not all(r['response']['usage']['input_tokens']==len(tokens) for r in references):raise ValueError('Piece tokens differ from actual native inference usage')
     logged=item.get('nativeLoggedPrompts',[])
     if logged and not all(p==detokenized for p in logged):raise ValueError('Derived embedded template differs from observed native prompt')
     record=dict(group=group['name'],id=item['id'],request=request,requestSha256=rowhash(request),pieces=pieces,effectiveTokens=tokens,effectivePrompt=detokenized,inputTokens=len(tokens),truncatedPieces=sum(p['truncated'] for p in pieces),actualInferenceReferences=[dict(rowSha256=rowhash(r),inputTokens=r['response']['usage']['input_tokens']) for r in references],nativeLoggedPromptMatches=len(logged),modelGenerations=0)
     if group['name']=='policy' and (record['truncatedPieces'] or request['state']['policy'] not in detokenized):raise ValueError('Policy request is clipped or loses complete trusted rules')
     records.append(record)
     with (out/'requests.jsonl').open('a') as trace:trace.write(json.dumps(record,ensure_ascii=False)+'\n')
  finally:
   if process.poll() is None:
    process.terminate()
    try:process.wait(timeout=10)
    except subprocess.TimeoutExpired:process.kill();process.wait(timeout=5)
 summary=dict(status='complete',modelGenerations=0,records=len(records),priorInferenceTokenMatches=sum(len(r['actualInferenceReferences']) for r in records),priorNativeLoggedPromptMatches=sum(r['nativeLoggedPromptMatches'] for r in records),priorTruncatedRequests=sum(r['truncatedPieces']>0 for r in records if r['group']!='policy'),policyRequestsWithoutTruncation=sum(r['truncatedPieces']==0 for r in records if r['group']=='policy'),maximumPolicyInputTokens=max(r['inputTokens'] for r in records if r['group']=='policy'),requestsSha256=sha(out/'requests.jsonl'),renderer='LocalJinja3.1.2 plus exact pinned native96/45 per-piece policy and actual native tokenization/detokenization. Not official Transformers/reference-logit parity.',logScope='Slot launch dumps the preceding completed prompt; five of six prompts are observable per worker. Missing last prompts are not credited as observed.',workerExitCode=process.returncode)
 save(out/'summary.json',summary);print(json.dumps(summary))
if __name__=='__main__':execute()
