import json,pathlib,subprocess,time,re,sys
root=pathlib.Path.cwd(); r=root/'.workspace/research/doubao-alignment'
model=sys.argv[1]; out=r/('tag-dev-'+model+'.jsonl')
p=subprocess.Popen([str(r/'tag-worker')],stdin=subprocess.PIPE,stdout=subprocess.PIPE,text=True)
n=0
def call(op,**data):
 global n
 n+=1;p.stdin.write(json.dumps(dict(version=2,id=n,op=op,**data),ensure_ascii=False)+'\n');p.stdin.flush()
 result=json.loads(p.stdout.readline()); assert result['ok'],result; return result
try:
 print(call('load',runtime=str(root/'apps/desktop/resources/darwin-arm64/llm'),model=str(root/'.workspace/ablation-models'/model),threads=4,maxLength=4096,maxNewTokens=384),flush=True)
 with out.open('x') as f:
  for s in json.loads((root/'.workspace/research/multilingual-disfluency/corpus.json').read_text())['samples']:
   start=time.monotonic();res=call('rewrite',session=s['id'],scene='chat',text=s['sourceText']);ms=(time.monotonic()-start)*1000
   annotated=res.get('text','');plain=annotated.replace('<remove>','').replace('</remove>','');valid=plain==s['sourceText'] and annotated.count('<remove>')==annotated.count('</remove>')
   cleaned=re.sub(r'<remove>.*?</remove>','',annotated) if valid else s['sourceText']
   passed=all(t in cleaned for t in s['requiredStrings']+s['protected']) and all(t not in cleaned for t in s['forbiddenStrings'])
   row=dict(id=s['id'],source=s['sourceText'],annotation=annotated,cleaned=cleaned,valid=valid,passed=passed,ms=ms,result=res)
   f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush(); print(s['id'],valid,passed,annotated,flush=True)
   call('end',session=s['id'])
finally:
 call('close');p.stdin.close();p.wait(timeout=10)
