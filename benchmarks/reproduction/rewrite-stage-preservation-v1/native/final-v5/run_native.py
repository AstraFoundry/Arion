"""Explicit phases for serial native checks; no inference occurs in build phase."""
import argparse,copy,hashlib,json,os,pathlib,subprocess,time
p=argparse.ArgumentParser();p.add_argument('phase',choices=['build','ocr-and-pack','native-three','cache-on','cache-off','regressions']);a=p.parse_args()
root=pathlib.Path.cwd();base=pathlib.Path(__file__).resolve().parent;sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
def run(command,env=None):
 print(' '.join(map(str,command)),flush=True);subprocess.run(list(map(str,command)),check=True,env=env)
def exclusive_json(path,value):
 with path.open('x') as file:json.dump(value,file,ensure_ascii=False,indent=2)
if a.phase=='build':
 snapshot=base/'source-snapshot';snapshot.mkdir();sources={};mappings=[]
 for directory in ['packages/core/application','packages/core/domain','apps/desktop/infrastructure/llm','apps/desktop/cmd/arion-llm','apps/desktop/infrastructure/modelruntime','apps/desktop/infrastructure/audio','apps/desktop/infrastructure/macos','models']:
  for file in (root/directory).rglob('*'):
   if not file.is_file() or file.suffix not in ['.go','.json','.md','.txt','.m','.h']:continue
   rel=file.relative_to(root);target=snapshot/rel
   if target.suffix=='.md':target=target.with_name(target.name+'.txt')
   target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(file.read_bytes());sources[str(rel)]=sha(file);mappings.append(dict(source=str(rel),snapshot=str(target.relative_to(base)),sha256=sha(target)))
 runtime=base/'runtime';runtime.mkdir()
 s=(root/'packages/core/application/live.go').read_text();start=s.index('func (c *Coordinator) completedRewrite(');end=s.index('\nfunc (c *Coordinator) newLive(',start)
 overlay=base/'cache-off/live-overlay.go';overlay.write_text(s[:start]+'func (c *Coordinator) completedRewrite(r *run, request ports.RewriteRequest) (ports.RewriteResult, bool) {\n return ports.RewriteResult{}, false\n}\n'+s[end:])
 exclusive_json(base/'cache-off/overlay.json',{'Replace':{str(root/'packages/core/application/live.go'):str(overlay)}})
 run(['go','build','-o',runtime/'arion-llm','./apps/desktop/cmd/arion-llm'],dict(os.environ,CGO_ENABLED='0'))
 run(['go','build','-o',runtime/'arion-ocr','./apps/desktop/cmd/arion-ocr'],dict(os.environ,CGO_ENABLED='1'))
 for variant in ['native-three','cache-on','cache-off','regressions']:
  args=['go','build']
  if variant=='cache-off':args+=['-overlay',base/'cache-off/overlay.json']
  args+=['-o',base/variant/'arion-live-bench',base/variant/'main.go'];run(args,dict(os.environ,CGO_ENABLED='1'))
 assert all(sha(root/name)==digest for name,digest in sources.items()),'Source changed during builds'
 exclusive_json(base/'build-provenance.json',dict(status='completed-build-no-inference',sources=sources,pathMappings=mappings,binaries={str(f.relative_to(base)):sha(f) for f in base.rglob('*') if f.is_file() and f.name in ['arion-live-bench','arion-llm','arion-ocr']},overlaySha256=sha(overlay)))
elif a.phase=='ocr-and-pack':
 folder=base/'regressions';run([base/'runtime/arion-ocr','-corpus',folder/'numeric-corpus.json','-out',folder/'numeric-ocr.jsonl'])
 ocr={r['caseId']:r['text'] for r in map(json.loads,(folder/'numeric-ocr.jsonl').read_text().splitlines()) if r['level']=='accurate' and not r.get('error')}
 c=json.loads((folder/'existing-corpus.json').read_text());numeric=json.loads((folder/'numeric-corpus.json').read_text())
 for row in numeric['samples']:row['screen']=ocr[row['id']]
 c['samples']+=numeric['samples'];exclusive_json(folder/'corpus.json',c)
 m=json.loads((folder/'existing-audio-manifest.json').read_text());newm=json.loads((folder/'numeric-audio/manifest.json').read_text())
 for row in newm['samples']:row['audio']['path']='../numeric-audio/'+row['audio']['path']
 m['samples']+=newm['samples'];m['textCorpusSha256']=sha(folder/'corpus.json');exclusive_json(folder/'audio/manifest.json',m)
else:
 variant=a.phase;folder=base/variant
 fixtures=base/'cache-fixtures' if variant.startswith('cache-') else folder
 command=['caffeinate','-i',folder/'arion-live-bench','-manifest',fixtures/'audio/manifest.json','-corpus',fixtures/'corpus.json','-models',root/'.workspace/models/store','-bundle','xasr-160-xasr-final','-llm-worker',base/'runtime/arion-llm','-llm-runtime','/Applications/Arion.app/Contents/Resources/llm','-llm-model','/Users/steven/Library/Application Support/Arion/models/qwen3.5-2b-q4-k-m-metal','-variant','final-v5-'+variant,'-out',folder/'asr.jsonl']
 if variant.startswith('cache-'):command+=['-segment-finalize']
 metadata=dict(status='started',command=list(map(str,command)),workerSha256=sha(base/'runtime/arion-llm'),driverSha256=sha(folder/'arion-live-bench'),corpusSha256=sha(fixtures/'corpus.json'),manifestSha256=sha(fixtures/'audio/manifest.json'),limitations='Exposed development controls; no independent quality/generalization or isolated comparative latency claim.')
 exclusive_json(folder/'invocation.json',metadata)
 start=time.monotonic()
 try:
  with (folder/'stdout.log').open('x') as out,(folder/'stderr.log').open('x') as err:subprocess.run(list(map(str,command)),stdout=out,stderr=err,check=True)
  metadata['status']='completed'
 except Exception as exc:metadata['status']='failed';metadata['error']=repr(exc);raise
 finally:
  metadata['wallSeconds']=time.monotonic()-start;exclusive_json(folder/'completion.json',metadata)
