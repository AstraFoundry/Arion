import pathlib,subprocess,sys,json,hashlib,time,os
root=pathlib.Path.cwd();r=root/'.workspace/research/doubao-alignment'
state=r/'product-state.json'
with state.open('x') as f:json.dump({'status':'running'},f)
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
try:
 while json.loads((r/'matched-state.json').read_text())['status']=='running':time.sleep(2)
 assert json.loads((r/'matched-state.json').read_text())['status']=='complete'
 manifest=json.loads((r/'final-manifest.json').read_text())
 assert all(sha(root/p)==digest for p,digest in manifest['artifacts'].items())
 for p in manifest['sources']:manifest['sources'][p]=sha(root/p)
 for p in manifest['scorers']:manifest['scorers'][p]=sha(root/p)
 manifest['binaries']={n:sha(r/n) for n in ['baseline0-worker','baseline0-bench','baseline0-audio','product-worker','product-bench','product-audio']}
 manifest['settings']['rules']='arion-rules-v3'
 manifest['settings']['baselineRules']='arion-rules-v2'
 manifest['settings']['baselineWorkerCGO']=0
 manifest['settings']['productWorkerCGO']=0
 manifest['settings']['schedule']='sequential matched baseline and product, no simultaneous model arms'
 manifest['lineage']={'initialManifestSha256':sha(r/'final-manifest.json'),'matchedManifestSha256':sha(r/'matched-manifest.json')}
 assert sha(r/'fresh-test.json')==json.loads((r/'fresh-freeze.json').read_text())['corpusSha256']
 manifest['corpora'][str((r/'fresh-test.json').relative_to(root))]=sha(r/'fresh-test.json')
 manifest['testExposure']={'original28':'regression after audio-stage review exposed first-person acronym corruption','fresh16':'frozen before model outputs; no prompt tuning after this freeze'}
 manifest['buildInfo']={n:subprocess.check_output(['go','version','-m',str(r/n)],text=True).replace(str(root),'$REPOSITORY') for n in manifest['binaries']}
 with (r/'product-manifest.json').open('x') as f:json.dump(manifest,f,indent=2)
 for split in ['dev','alignment','regression','heldout']:
  subprocess.run([sys.executable,str(r/'run-text.py'),split,'product-'+split,'product'],check=True)
 subprocess.run([sys.executable,str(r/'run-text.py'),'fresh','fresh','baseline0','product'],check=True)
 subprocess.run([sys.executable,str(r/'run-audio.py'),'product'],check=True)
 env={**os.environ,'ARION_LLM_RUNTIME':str(root/'apps/desktop/resources/darwin-arm64/llm'),'ARION_LLM_MODEL':str(root/'.workspace/ablation-models/qwen3-1.7b')}
 with (r/'product-native-parity.log').open('x') as log:subprocess.run(['go','test','./infrastructure/llm/engine','-run','TestCleanupReuseParity','-count=1','-v'],check=True,cwd=root/'apps/desktop',env=env,stdout=log,stderr=log)
 state.write_text(json.dumps({'status':'complete'})+'\n');print('product suite complete',flush=True)
except BaseException as e:
 state.write_text(json.dumps({'status':'failed','error':str(e)})+'\n');raise
