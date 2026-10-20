"""Acquire one revision-pinned <=2B Q8 export; never load or execute the model."""
import pathlib,json,urllib.request,hashlib,importlib.util,time
ROOT=pathlib.Path(__file__).resolve().parents[3];HERE=pathlib.Path(__file__).resolve().parent;PRIVATE=ROOT/'.workspace/rewrite-2b-precision-v1'
REPO='unsloth/Qwen3.5-2B-GGUF';REVISION='f6d5376be1edb4d416d56da11e5397a961aca8ae';FILENAME='Qwen3.5-2B-Q8_0.gguf'
def write(path,data):path.write_text(json.dumps(data,ensure_ascii=False,indent=2)+'\n')
def sha(path):
 digest=hashlib.sha256()
 with path.open('rb') as stream:
  for block in iter(lambda:stream.read(4*1024*1024),b''):digest.update(block)
 return digest.hexdigest()
def fetch(url,path):
 with urllib.request.urlopen(url,timeout=60) as response:data=response.read()
 with path.open('xb') as stream:stream.write(data)
 return dict(url=url,path=str(path.relative_to(ROOT)),sha256=sha(path))
def main():
 manifest=PRIVATE/'acquisition.json'
 if manifest.exists():raise FileExistsError('Existing acquisition attempt; inspect its state instead of overwriting')
 state=dict(status='running',repo=REPO,revision=REVISION,filename=FILENAME,inferenceExecuted=False,startedUnix=time.time());write(manifest,state)
 try:
  metadata=PRIVATE/'export-api.json';source=fetch('https://huggingface.co/api/models/'+REPO+'/revision/'+REVISION+'?blobs=true',metadata);api=json.loads(metadata.read_text())
  if api['sha']!=REVISION or api['cardData']['license']!='apache-2.0':raise ValueError('Unexpected export source/license')
  entry=next(r for r in api['siblings'] if r['rfilename']==FILENAME);expected=entry['lfs']['sha256'];size=entry['lfs']['size'];url='https://huggingface.co/'+REPO+'/resolve/'+REVISION+'/'+FILENAME
  state.update(expectedSha256=expected,expectedBytes=size,sources=[source]);write(manifest,state)
  card=fetch('https://huggingface.co/'+REPO+'/resolve/'+REVISION+'/README.md',PRIVATE/'export-README.md');state['sources'].append(card);write(manifest,state)
  partial=PRIVATE/(FILENAME+'.partial');completed=PRIVATE/FILENAME;downloaded=0;digest=hashlib.sha256()
  if partial.exists() or completed.exists():raise FileExistsError('Existing weight bytes; no silent overwrite')
  with urllib.request.urlopen(url,timeout=60) as response,partial.open('xb') as stream:
   while True:
    block=response.read(4*1024*1024)
    if not block:break
    stream.write(block);digest.update(block);downloaded+=len(block)
  if downloaded!=size or digest.hexdigest()!=expected:raise ValueError('Weight download identity mismatch')
  partial.rename(completed)
  spec=importlib.util.spec_from_file_location('precision_gguf_reader',HERE.parent/'rewrite-small-models-v1/gguf_info.py');reader=importlib.util.module_from_spec(spec);spec.loader.exec_module(reader);info=reader.inspect(completed)
  baseline_path=HERE.parent/'rewrite-small-models-v1/2b-gguf-info.json';baseline=json.loads(baseline_path.read_text())
  if info['parameters']!=1881825088 or info['parameters']>2000000000:raise ValueError('Wrong/ineligible actual tensor count')
  layout=lambda d:[(r['name'],r['shape']) for r in d['tensors']]
  if layout(info)!=layout(baseline):raise ValueError('Q8/Q4 tensor layout differs')
  if info['metadata']['tokenizer.chat_template']!=baseline['metadata']['tokenizer.chat_template']:raise ValueError('Q8/Q4 native template differs')
  write(PRIVATE/'gguf-info.json',info)
  state.update(status='complete',completedUnix=time.time(),bytes=downloaded,sha256=expected,weightPath=str(completed.relative_to(ROOT)),parameters=info['parameters'],ggufInfoSha256=sha(PRIVATE/'gguf-info.json'),baselineInfoPath=str(baseline_path.relative_to(ROOT)),baselineInfoSha256=sha(baseline_path),tensorLayoutEqual=True,nativeChatTemplateEqual=True,inferenceExecuted=False,caveat='Same pinned export family/revision and tensor layout; no independent converter-build/base-weight numeric identity proof, no causal quantization/quality or minimum16GB/native performance acceptance. Acquisition network/disk activity overlaps the separately recorded CPU prompt study; do not attribute prompt timing causally.');write(manifest,state);print('Verified Q8 acquisition:',downloaded,'bytes;',info['parameters'],'tensor elements; no inference',flush=True)
 except Exception as error:
  state.update(status='failed',completedUnix=time.time(),errorType=type(error).__name__,error=str(error));write(manifest,state);raise
if __name__=='__main__':main()
