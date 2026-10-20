"""Diagnostic inference on all120 pinned natural ASR sources without hidden references."""
import pathlib,sys,json,argparse,importlib.util
HERE=pathlib.Path(__file__).resolve().parent;sys.path.insert(0,str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
ROOT=engine.ROOT
spec=importlib.util.spec_from_file_location('natural_formatter_v3',HERE.parent/'rewrite-formatter-v3/pipeline.py');pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--out',type=pathlib.Path,required=True);a=parser.parse_args();f=engine.read(HERE/'freeze.json');engine.MODEL=ROOT/f['modelPath'];engine.verify(f);p=engine.read(HERE/'protocol.json')
 inputs=engine.read(ROOT/p['inputs']);cleanup={r['id']:r for r in engine.read(ROOT/p['cleanup'])};out=a.out.resolve();out.mkdir(parents=True,exist_ok=False);m=dict(status='running',phase='diagnostic',freezeSha256=engine.sha(HERE/'freeze.json'),model=f['model'],runtimeFiles=f['runtimeFiles'],workers=[],sourceCases=120,newAsrGenerations=0,newSelectorGenerations=0,pipeline='natural-format-v1');engine.write(out/'manifest.json',m);worker=None;count=0
 try:
  worker=Session(out,'metal-chat',18811,gpu_layers=99,verbosity=5);m['workers'].append(dict(scene='chat',command=worker.command,startupMs=worker.startup_ms))
  with (out/'results.jsonl').open('x') as stream:
   for s in inputs:
    prior=cleanup[s['id']]
    if prior['source']!=s['source'] or s['context']!={} or s['scene']!='chat':raise ValueError('Changed natural source or undeclared context/scene')
    cleaned=prior['cleanup'];prepared=pipeline.candidates.prepare(cleaned['text'],s['context'],s['scene'])
    if prepared['hypotheses']:raise ValueError('Unexpected no-context spelling candidates; no simulated selector response')
    request,encoding=pipeline.format_payload(s,cleaned['text'],prepared,{},[]);response,elapsed,rss=worker.generate(request);body=response['choices'][0]['message']['content']
    row=dict(**s,variant='natural-metal-v3',stage='format',pipeline='natural-format-v1',effectiveModelScene='chat',templateVersion='formatter-v3-chat',templateSha256=engine.sha(HERE.parent/'rewrite-formatter-v3/prompts.json'),modelSha256=f['model']['sha256'],request=request,actualContextUsed={},cleanup=cleaned,editedSource=cleaned['text'],edits=[],sourcePreparation=dict(kind='deterministic-no-context',hypotheses=prepared['hypotheses'],modelSelectorCalled=False,cleanupSource=p['cleanup']),encoding=encoding,response=response,renderedBody=body,renderError=None,preparationMs=None,generationMs=elapsed,renderMs=0,textStageMs=elapsed,residentRssBytes=rss,scope='Historical natural read-speech frozen ASR; no new ASR/audio/OCR inference. Source-only meaning/style diagnostic, not intended-speech reference recovery or independent/held-out/scene-general/native/min16GB acceptance.')
    stream.write(json.dumps(row,ensure_ascii=False)+'\n');stream.flush();count+=1;print(count,s['id'],round(elapsed),repr(body),flush=True)
  worker.close();worker=None;m.update(status='complete',generations=count,resultsSha256=engine.sha(out/'results.jsonl'));engine.write(out/'manifest.json',m)
 except Exception as error:
  if worker:worker.abort()
  m.update(status='failed',completedGenerations=count,error=str(error));engine.write(out/'manifest.json',m);raise
if __name__=='__main__':main()
