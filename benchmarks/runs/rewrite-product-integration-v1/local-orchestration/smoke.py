import subprocess,json,time,pathlib,os
root=pathlib.Path('/Users/steven/code/Arion');base=root/'.workspace/rewrite-product-integration';runtime=root/'apps/desktop/resources/darwin-arm64/llm'
model=root/'.workspace/rewrite-small-models-v1'
p=subprocess.Popen([str(runtime/'arion-llm')],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=(base/'worker-stderr.log').open('w'),text=True)
rows=[]
def call(obj):
 obj=dict(version=2,id=len(rows)+1,**obj);start=time.monotonic();p.stdin.write(json.dumps(obj,ensure_ascii=False)+'\n');p.stdin.flush();line=p.stdout.readline()
 if not line:raise RuntimeError('worker exited')
 response=json.loads(line);rows.append(dict(request=obj,response=response,elapsedMs=(time.monotonic()-start)*1000));(base/'smoke-results.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2));print(str(rows[-1])[:1200],flush=True);return response
try:
 r=call(dict(op='load',runtime=str(runtime),model=str(model),threads=4,maxLength=4096,maxNewTokens=256))
 if not r['ok']:raise RuntimeError(r)
 for scene,text,context in [
 ('work','小陈负责设计，小刘负责测试，设计稿周三交，测试周五完成。',None),
 ('email','周经理您好，我周四请假一天，交接给小赵了，麻烦批准，谢谢。',None),
 ('work','给 load cash 加日志，不记录用户内容，先别改缓存策略。',{'Screen':'编辑器：cache.go\nfunc loadCache(key string)\n议题：为 loadCache 添加诊断日志'}),
 ('work','给 read cash 加日志，先别提交。',{'Screen':'readCash readCache'}),
 ('work','请核对服务器的配植清单，暂时不要重启。',{'Screen':'苗圃培植清单：浇水和施肥。'}),
 ('prompt','给 fetchState 加校验，不改返回字段，也别动数据库。',None),
 ('chat','If it rains, I might join online. Do not confirm for me.',None),
 ]:
  q=dict(op='rewrite',session='integration-smoke',scene=scene,text=text)
  if context:q['context']=context
  r=call(q)
  if not r['ok']:raise RuntimeError(r)
 call(dict(op='end',session='integration-smoke'))
 call(dict(op='prefill',session='prefill-smoke',scene='work',text='小陈负责设计，周三交。'))
 call(dict(op='close'));p.wait(timeout=5);print('exit',p.returncode)
finally:
 if p.poll() is None:p.kill();p.wait()
