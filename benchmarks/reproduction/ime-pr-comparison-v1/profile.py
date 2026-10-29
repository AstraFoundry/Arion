"""Unreported sampler controls; run only after all measured rounds finish."""
import argparse,json,pathlib,subprocess,time
from run import receive,write,load_variants,variant_paths
p=argparse.ArgumentParser();p.add_argument('work');p.add_argument('out');p.add_argument('--variants');a=p.parse_args()
variants=load_variants(a.variants)
work=pathlib.Path(a.work).resolve();out=pathlib.Path(a.out).resolve();out.mkdir(parents=True,exist_ok=False)
for variant in variants:
 for raw in ['computer','artifi']:
  name=variant+'-'+raw;request=out/(name+'-request.json');write(request,{'input':raw,'warmups':['xuetang','heishan'],'pageSize':9,'paceMs':0})
  binary,data=variant_paths(work,variant,variants)
  with (out/(name+'.stderr')).open('w') as err:
   proc=subprocess.Popen([str(binary),str(data),str(work/'profile-users'/name),str(request)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=err,text=True,bufsize=1)
   try:
    initial=receive(proc);write(out/(name+'-initial.json'),initial)
    proc.stdin.write('{"op":"profile","iterations":12000}\n');proc.stdin.flush();time.sleep(.2)
    sample=subprocess.run(['/usr/bin/sample',str(proc.pid),'2','-file',str(out/(name+'.sample.txt'))],text=True,capture_output=True)
    write(out/(name+'-sampler.json'),{'status':sample.returncode,'stdout':sample.stdout,'stderr':sample.stderr,'note':'Unreported workload includes cancel, native typing, driver snapshots and allocation; interpret native key-handler subtree separately from harness work.'})
    result=receive(proc);write(out/(name+'-profile.json'),result)
    proc.stdin.write('{"op":"quit"}\n');proc.stdin.flush();terminal=receive(proc);write(out/(name+'-terminal.json'),terminal);proc.stdin.close();proc.wait(timeout=10)
    assert proc.returncode==0 and terminal['status']=='completed'
   finally:
    if proc.poll() is None:proc.kill();proc.wait()
  print(name,'sampled',flush=True)
