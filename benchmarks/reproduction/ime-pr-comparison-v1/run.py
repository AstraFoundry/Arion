"""Alternate exact native binaries; keep reference text in this controller only."""
import argparse, hashlib, json, os, pathlib, random, selectors, shutil, subprocess, time

BASE = pathlib.Path(__file__).resolve().parents[3]
def read(path): return json.loads(pathlib.Path(path).read_text())
def write(path,value): pathlib.Path(path).write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')
def digest(path): return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()
def command(*args):
 p=subprocess.run(args,text=True,capture_output=True)
 return {'status':p.returncode,'stdout':p.stdout,'stderr':p.stderr}
def receive(process):
 with selectors.DefaultSelector() as s:
  s.register(process.stdout,selectors.EVENT_READ)
  if not s.select(40): raise TimeoutError('Native output timed out')
 line=process.stdout.readline()
 if not line: raise RuntimeError('Native output ended before completion')
 value=json.loads(line)
 if value['status']=='failed':raise RuntimeError(value.get('failure'))
 return value

def load_variants(file=None):
 variants=read(file) if file else {'pr40':{'snapshot':'pr40','build':'build40'},'pr45':{'snapshot':'pr45','build':'build45'}}
 assert len(variants)==2 and all(isinstance(name,str) and name and set(value)=={'snapshot','build'} and all(isinstance(p,str) and p for p in value.values()) for name,value in variants.items()),'Two named snapshot/build variants are required'
 return variants

def variant_paths(work,variant,variants):
 config=variants[variant]
 return work/config['build']/'driver',work/config['snapshot']/'apps/desktop/resources/darwin/keyboard/data'

def observation(work,out,fixture,variant,round_id,page,pace,variants):
 name=f'{round_id}-{fixture["id"]}-{variant}-p{page}-pace{pace}'
 request=out/'requests'/f'{name}.json'; raw=out/'raw'/f'{name}.json'; profile=work/'profiles'/name
 request.parent.mkdir(exist_ok=True);raw.parent.mkdir(exist_ok=True);profile.parent.mkdir(exist_ok=True)
 write(request,{'input':fixture['input'],'warmups':['xuetang','heishan'],'pageSize':page,'paceMs':pace})
 binary,data=variant_paths(work,variant,variants)
 record={'name':name,'variant':variant,'round':round_id,'fixture':fixture['id'],'pageSize':page,'paceMs':pace,'request':str(request.relative_to(out)),'status':'failed','actions':[]}
 p=None;started=time.perf_counter()
 try:
  with (out/'raw'/f'{name}.stderr').open('w') as err:
   p=subprocess.Popen([str(binary),str(data),str(profile),str(request)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=err,text=True,bufsize=1)
   initial=receive(p);record['initial']=initial
   assert initial['status']=='ready'
   def send(op,**args):
    p.stdin.write(json.dumps({'op':op,**args})+'\n');p.stdin.flush()
    value=receive(p);record['actions'].append(value);return value
   state=initial['state'];segments=fixture.get('segments',[fixture['target']] if 'target' in fixture else [])
   record['searches']=[]
   for target in segments:
    search={'target':target,'pages':[],'found':False};record['searches'].append(search)
    for attempt in range((189+page-1)//page):
     search['pages'].append({'page':state['page'],'candidates':state['candidates'],'composing':state['composing']})
     if target in state['candidates']:
      index=state['candidates'].index(target);search['found']=True;search['rank']=state['page']*page+index+1
      selected=send('select',index=index);assert selected['handled'];state=selected['state'];break
     if not state['composing'] or state['lastPage']:break
     next_value=send('next');state=next_value['state']
     if not next_value['handled']:break
    if not search['found']:break
   if not segments and state['candidates']:
    state=send('select',index=0)['state']
   record['final']=state
   send('cancel')
   p.stdin.write('{"op":"quit"}\n');p.stdin.flush()
   terminal=receive(p);record['terminal']=terminal
   p.stdin.close();p.wait(timeout=10)
   assert terminal['status']=='completed' and p.returncode==0
   record['status']='completed';record['exitStatus']=p.returncode
 except Exception as e:
  record['error']=str(e)
  if p and p.poll() is None:p.kill();p.wait()
  record['exitStatus']=None if p is None else p.returncode
 finally:
  record['processWallMs']=(time.perf_counter()-started)*1000
  write(raw,record)
  if record['status']=='completed':shutil.rmtree(profile)
 return {'name':name,'path':str(raw.relative_to(out)),'sha256':digest(raw),'status':record['status'],'variant':variant,'round':round_id,'fixture':fixture['id'],'pageSize':page,'paceMs':pace}

def main():
 parser=argparse.ArgumentParser();parser.add_argument('work');parser.add_argument('out');parser.add_argument('--smoke',action='store_true');parser.add_argument('--variants');args=parser.parse_args()
 work=pathlib.Path(args.work).resolve();out=pathlib.Path(args.out).resolve();out.mkdir(parents=True,exist_ok=False)
 variants=load_variants(args.variants);names=list(variants)
 corpus_path=BASE/'benchmarks/corpora/ime-pr-comparison-v1/corpus.json';corpus=read(corpus_path)
 identities={v:read(work/variants[v]['build']/'identity.json') for v in names}
 assert set(read(work/'versions.json'))==set(names),'Version identities must match the selected variants'
 def verify():
  for v,identity in identities.items():
   assert digest(identity['binary'])==identity['sha256']
   for relative,entry in identity['files'].items():assert digest(pathlib.Path(identity['root'])/relative)==entry['sha256'],relative
 verify();write(out/'identities.json',identities)
 write(out/'metadata.json',{'versions':read(work/'versions.json'),'corpusSha256':digest(corpus_path),'driverSha256':digest(BASE/'benchmarks/reproduction/ime-pr-comparison-v1/driver.mm'),'runnerSha256':digest(__file__),'hardware':{k:command('sysctl','-n',k) for k in ['hw.model','machdep.cpu.brand_string','hw.ncpu','hw.memsize']},'os':command('sw_vers'),'clang':command('xcrun','clang','--version'),'power':command('pmset','-g','custom'),'battery':command('pmset','-g','batt'),'thermal':command('pmset','-g','therm'),'load':os.getloadavg(),'processes':command('ps','-axo','pid,pcpu,comm','-r'),'boundary':'Actual ArionTyping native key dispatch, candidate menu update and NSTextView marked/committed text; no system IMK events, visible window or GPU presentation. Pacing and controller/JSON overhead excluded from nativeStateMs. Warmup cancelled; fresh process/encrypted profile per observation.'})
 plan=[]
 for round_id in range(1,2 if args.smoke else 6):
  fixtures=list(corpus['cases']);random.Random(4300+round_id).shuffle(fixtures)
  if args.smoke:fixtures=[c for c in corpus['cases'] if c['input'] in ['hello','xin','Hello','nihaohello','hellonihao','github']]
  for i,fixture in enumerate(fixtures):
   for variant in (names if (i+round_id)%2 else list(reversed(names))):plan.append((round_id,fixture,variant,9,0))
 if not args.smoke:
  for i,fixture in enumerate(corpus['cases']):
   for v in (names if i%2 else list(reversed(names))):plan.append((6,fixture,v,5,0))
  for round_id in range(7,12):
   fixtures=[c for c in corpus['cases'] if c.get('paced')];random.Random(4300+round_id).shuffle(fixtures)
   for i,fixture in enumerate(fixtures):
    for v in (names if (i+round_id)%2 else list(reversed(names))):plan.append((round_id,fixture,v,9,80))
 write(out/'plan.json',[{'round':r,'fixture':f['id'],'variant':v,'pageSize':p,'paceMs':t} for r,f,v,p,t in plan])
 write(out/'status.json',{'status':'running','planned':len(plan)})
 index=[]
 for r,f,v,p,t in plan:
  item=observation(work,out,f,v,r,p,t,variants);index.append(item);write(out/'index.json',index)
  if item['status']!='completed':
   write(out/'status.json',{'status':'failed','observations':len(index),'item':item});raise RuntimeError('Native failure preserved: '+item['path'])
  if len(index)%50==0:print(f'Completed {len(index)}/{len(plan)} actual native observations',flush=True)
 verify();write(out/'status.json',{'status':'completed','observations':len(index),'finalLoad':os.getloadavg(),'thermal':command('pmset','-g','therm')});print('Completed',len(index),flush=True)
if __name__=='__main__':main()
