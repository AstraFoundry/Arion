"""Reference-free native worker diagnostic; completed results retain all replies."""
import argparse,base64,hashlib,json,pathlib,subprocess,time,wave,queue,threading,os

def sha(path):
 h=hashlib.sha256()
 with open(path,'rb') as f:
  for b in iter(lambda:f.read(8*1024*1024),b''):h.update(b)
 return h.hexdigest()

def main():
 p=argparse.ArgumentParser();p.add_argument('--worker',required=True);p.add_argument('--model',required=True);p.add_argument('--corpus',required=True);p.add_argument('--catalog',required=True);p.add_argument('--out',required=True);a=p.parse_args()
 out=pathlib.Path(a.out).resolve();out.mkdir(parents=True,exist_ok=False)
 worker=pathlib.Path(a.worker).resolve();model=pathlib.Path(a.model).resolve();corpuspath=pathlib.Path(a.corpus).resolve();corpus=json.loads(corpuspath.read_text());catalog=json.load(open(a.catalog));artifact=next(x for x in catalog['artifacts'] if x['kind']=='qwen3-asr-mlx')
 run={'status':'running','stage':'native-worker-compute','split':'dev','corpusSha256':sha(corpuspath),'runnerSha256':sha(__file__),'catalogSha256':sha(a.catalog),'runtimeFiles':{},'modelFiles':{},'limits':{'maxTokens':128,'seed':11,'temperature':0},'limitations':['Engine compute only; excludes capture, preview scheduling, rewrite and insertion.','Development corpus; held-out test remains unused.','No Windows or Intel inference acceptance.']}
 def save(): (out/'run.json').write_text(json.dumps(run,indent=2)+'\n')
 save()
 child=None
 try:
  for name,f in artifact['files'].items():
   file=model/f['member'];assert file.stat().st_size==f['sizeBytes'];assert sha(file)==f['sha256'];run['modelFiles'][f['member']]=f['sha256']
  for file in worker.parent.rglob('*'):
   if file.is_file():run['runtimeFiles'][str(file.relative_to(worker.parent))]=sha(file)
  samples=[{'id':s['id'],'audio':s['audio']} for s in corpus['samples'] if s['split']=='dev'];warmups=[{'id':s['id'],'audio':s['audio']} for s in corpus['samples'] if s['split']=='train'][:2]
  assert len(samples)==118 and len(warmups)==2
  for s in warmups+samples:assert sha(corpuspath.parent/s['audio']['path'])==s['audio']['sha256']
  replies=queue.Queue()
  stderr=(out/'stderr.log').open('x')
  # Running from an unrelated working directory proves runtime relocation.
  start=time.monotonic();child=subprocess.Popen([str(worker),'worker',str(model)],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=stderr,cwd='/tmp',env={'PATH':'/usr/bin:/bin','HOME':os.environ['HOME'],'SPEECH_NO_UPDATE_CHECK':'1'},text=True)
  def read():
   with (out/'stdout.jsonl').open('x') as raw:
    for line in child.stdout:
     raw.write(line);raw.flush()
     try: replies.put(json.loads(line))
     except ValueError: replies.put({'type':'invalid-protocol'});return
   replies.put({'type':'eof'})
  threading.Thread(target=read,daemon=True).start()
  ready=replies.get(timeout=90);assert ready['type']=='ready' and ready['protocol']==3,ready;run['loadMs']=(time.monotonic()-start)*1000;save()
  def send(j):child.stdin.write(json.dumps(j)+'\n');child.stdin.flush()
  index=0
  for group,name in [(warmups,'warmups.jsonl'),(samples,'results.jsonl')]:
   with (out/name).open('x') as f:
    for s in group:
     index+=1;id=str(index)
     with wave.open(str(corpuspath.parent/s['audio']['path']),'rb') as w:
      assert (w.getframerate(),w.getnchannels(),w.getsampwidth())==(16000,1,2)
      pcm=w.readframes(w.getnframes())
     start=time.monotonic();send({'type':'chunk','id':id,'seq':0,'pcm':base64.b64encode(pcm).decode()});send({'type':'transcribe','id':id,'sample_rate':16000})
     reply=replies.get(timeout=60);elapsed=(time.monotonic()-start)*1000
     row={'caseId':s['id'],'audioSha256':s['audio']['sha256'],'elapsedMs':elapsed,'response':reply,'text':reply.get('text','')}
     f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush()
     assert reply['id']==id and reply['type']=='end' and reply['stop']=='complete',reply
     print(name,s['id'],round(elapsed),flush=True)
  # Cancel in-flight work, retain its terminal reply, then reuse the model.
  send({'type':'chunk','id':'cancel','seq':0,'pcm':base64.b64encode(pcm).decode()});send({'type':'transcribe','id':'cancel','sample_rate':16000});send({'type':'cancel','id':'cancel'})
  cancellation=replies.get(timeout=10);assert cancellation['type']=='cancelled',cancellation
  send({'type':'chunk','id':'reuse','seq':0,'pcm':base64.b64encode(pcm).decode()});send({'type':'transcribe','id':'reuse','sample_rate':16000});reuse=replies.get(timeout=60);assert reuse['type']=='end',reuse
  (out/'lifecycle.json').write_text(json.dumps({'cancel':cancellation,'reuse':reuse},ensure_ascii=False,indent=2)+'\n')
  child.stdin.close();child.wait(timeout=15);assert child.returncode==0
  assert replies.get(timeout=5)["type"]=="eof"
  run.update(status='complete',resultsSha256=sha(out/'results.jsonl'),warmupsSha256=sha(out/'warmups.jsonl'));save()
 except BaseException as error:
  run.update(status='failed',error=repr(error));save();raise
 finally:
  if child and child.poll() is None:child.kill();child.wait()
if __name__=='__main__':main()
