import base64,json,os,pathlib,queue,subprocess,threading,wave
root=pathlib.Path('/Users/steven/code/Arion')
binary=root/'.workspace/native-asr-migration/build/Build/Products/Release/arion-asr'
model=root/'.workspace/native-asr-migration/model-store/qwen3-asr-1.7b-mlx-8bit'
out=root/'.workspace/native-asr-hotwords-smoke'
log=(out/'stderr.txt').open('w')
p=subprocess.Popen([str(binary),'worker',str(model)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=log,text=True,env={**os.environ,'HF_HUB_OFFLINE':'1'})
q=queue.Queue()
def read():
 for line in p.stdout:q.put(json.loads(line))
 q.put(None)
threading.Thread(target=read,daemon=True).start()
rows=[]
def recv():
 r=q.get(timeout=180)
 assert r is not None,('worker exited',p.poll())
 rows.append(r)
 return r
def send(m):p.stdin.write(json.dumps(m,ensure_ascii=False)+'\n');p.stdin.flush()
try:
 ready=recv()
 assert ready['type']=='ready' and ready['protocol']==3 and 'prompt' in ready['capabilities'],ready
 with wave.open(str(root/'benchmarks/corpora/asr-window-v1/ocr-work-term.wav'),'rb') as w:
  assert (w.getnchannels(),w.getsampwidth(),w.getframerate())==(1,2,16000)
  pcm=base64.b64encode(w.readframes(w.getnframes())).decode()
 for identity,prompt in [('contextual','MLX Swift'),('empty',''),('overflow','x '*70000)]:
  send({'type':'chunk','id':identity,'seq':0,'pcm':pcm})
  send({'type':'transcribe','id':identity,'sample_rate':16000,'prompt':prompt})
  result=recv()
  assert result.get('id')==identity,result
  assert result['type']==('error' if identity=='overflow' else 'end'),result
  if identity!='overflow':assert result.get('stop')=='complete',result
 print(json.dumps({'checks':['native-ready-prompt-capability','contextual-complete','subsequent-empty-complete','actual-context-overflow-rejected'],'responses':rows},ensure_ascii=False))
finally:
 p.stdin.close()
 try:p.wait(timeout=10)
 except subprocess.TimeoutExpired:p.kill();p.wait()
 log.close()
 (out/'responses.json').write_text(json.dumps(rows,ensure_ascii=False,indent=2)+'\n')
