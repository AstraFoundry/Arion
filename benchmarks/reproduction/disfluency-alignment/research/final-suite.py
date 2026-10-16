import pathlib,subprocess,sys,json,os
root=pathlib.Path.cwd();r=root/'.workspace/research/doubao-alignment'
state=r/'final-state.json'
with state.open('x') as f:json.dump({'status':'running'},f)
def run(*args,**kwargs):subprocess.run(args,check=True,**kwargs)
try:
 run(sys.executable,str(r/'freeze-run.py'))
 for split in ['dev','alignment','regression']:
  run(sys.executable,str(r/'run-text.py'),split,'final-'+split,'final')
 # No test generation before the frozen candidate passes historical regressions.
 script='''import fs from "node:fs";import {scoreText} from "./tools/evaluation/live/score-text.mjs";
const base=".workspace/research/doubao-alignment/";
for(const [split,p] of [["dev",".workspace/research/multilingual-disfluency/corpus.json"],["alignment",".workspace/research/rewrite-alignment/corpus.json"],["regression",".workspace/live-heldout-corpus.json"]]){
 const c=JSON.parse(fs.readFileSync(p));const load=f=>fs.readFileSync(base+f,"utf8").trim().split("\\n").map(JSON.parse);
 const before=scoreText(c,load(split+"-baseline.jsonl")),after=scoreText(c,load("final-"+split+"-final.jsonl"));const a=new Map(after.perCase.map(x=>[x.id,x.pass]));
 const regressions=before.perCase.filter(x=>x.pass&&!a.get(x.id));if(regressions.length)throw Error(split+" regressions: "+JSON.stringify(regressions));
 console.log(split,before.deliveredPass,"->",after.deliveredPass,"regressions",regressions.length);
}'''
 run('node','--input-type=module','-e',script)
 (r/'hold-test').unlink()
 run(sys.executable,str(r/'run-text.py'),'heldout','final-heldout','baseline','final')
 run(sys.executable,str(r/'run-audio.py'))
 env={**os.environ,'ARION_LLM_RUNTIME':str(root/'apps/desktop/resources/darwin-arm64/llm'),'ARION_LLM_MODEL':str(root/'.workspace/ablation-models/qwen3-1.7b')}
 with (r/'native-parity.log').open('x') as log:
  run('go','test','./infrastructure/llm/engine','-run','TestCleanupReuseParity','-count=1','-v',cwd=root/'apps/desktop',env=env,stdout=log,stderr=log)
 state.write_text(json.dumps({'status':'complete'})+'\n')
 print('final suite complete',flush=True)
except BaseException as e:
 state.write_text(json.dumps({'status':'failed','error':str(e)})+'\n');raise
