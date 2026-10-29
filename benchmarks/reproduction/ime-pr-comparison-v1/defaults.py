"""Functional follow-up: choose the default candidate, without reference-guided selection."""
import argparse,json,pathlib,subprocess
from run import read,write,receive,load_variants,variant_paths
p=argparse.ArgumentParser();p.add_argument('work');p.add_argument('out');p.add_argument('--variants');p.add_argument('--strategy',choices=['default','folded'],default='default');a=p.parse_args()
variants=load_variants(a.variants)
work=pathlib.Path(a.work).resolve();out=pathlib.Path(a.out).resolve();out.mkdir(parents=True,exist_ok=False)
corpus=read(pathlib.Path(__file__).resolve().parents[3]/'benchmarks/corpora/ime-pr-comparison-v1/corpus.json');summary=[]
for fixture in [c for c in corpus['cases'] if c['group']=='mixed']:
 for variant in variants:
  name=fixture['id']+'-'+variant;request=out/(name+'-request.json');write(request,{'input':fixture['input'],'warmups':['xuetang','heishan'],'pageSize':9,'paceMs':0})
  binary,data=variant_paths(work,variant,variants)
  with (out/(name+'.stderr')).open('w') as err:
   proc=subprocess.Popen([str(binary),str(data),str(work/(a.strategy+'-users')/name),str(request)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=err,text=True,bufsize=1)
   result={'fixture':fixture['id'],'input':fixture['input'],'variant':variant,'expected':''.join(fixture['segments']),'actions':[]}
   try:
    initial=receive(proc);result['initial']=initial;state=initial['state']
    if a.strategy=='default':
     for _ in range(16):
      if not state['composing'] or not state['candidates']:break
      proc.stdin.write('{"op":"select","index":0}\n');proc.stdin.flush();step=receive(proc);result['actions'].append(step);state=step['state'];assert step['handled']
    else:
     for target in fixture['segments']:
      found=False
      for _ in range(21):
       index=next((i for i,c in enumerate(state['candidates']) if c.lower()==target.lower()),None)
       if index is not None:
        proc.stdin.write(json.dumps({'op':'select','index':index})+'\n');proc.stdin.flush();step=receive(proc);result['actions'].append(step);state=step['state'];assert step['handled'];found=True;break
       if not state['composing'] or state['lastPage']:break
       proc.stdin.write('{"op":"next"}\n');proc.stdin.flush();step=receive(proc);result['actions'].append(step);state=step['state']
       if not step['handled']:break
      if not found:break
    result['final']=state
    result['strategy']=a.strategy
    textMatches=state['text']==result['expected'] if a.strategy=='default' else state['text'].lower()==result['expected'].lower()
    result['success']=all(s['handled'] for s in initial['steps']) and textMatches and state['insertions']==1 and not state['composing'] and not state['marked']
    proc.stdin.write('{"op":"quit"}\n');proc.stdin.flush();result['terminal']=receive(proc);proc.stdin.close();proc.wait(timeout=10);assert proc.returncode==0
   finally:
    if proc.poll() is None:proc.kill();proc.wait()
  write(out/(name+'.json'),result);summary.append({k:result[k] for k in ['fixture','input','variant','expected','success','final']})
write(out/'summary.json',summary)
print('Default-choice follow-up complete:',len(summary),'native processes')
