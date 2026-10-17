"""Persistent, local-only correction worker; never receives intent/reference labels."""
import hashlib
import json
import pathlib
import resource
import sys
import time
import torch
from transformers import AutoModelForSeq2SeqLM, AutoTokenizer, BertTokenizerFast

torch.set_num_threads(4)
torch.set_num_interop_threads(1)


def detokenize(text, model):
    # Match textgen BartSeq2SeqModel.predict(split_on_space=False).
    return ''.join(text.split(' ')) if model == 'bart4csc' else text


def main():
    models={}
    for line in sys.stdin:
        request=json.loads(line)
        try:
            if request['op']=='load':
                start=time.perf_counter()
                for name, directory in request['models'].items():
                    path=pathlib.Path(directory)
                    manifest=json.loads((path/'manifest.json').read_text())
                    for artifact in manifest['files']:
                        with (path/artifact['file']).open('rb') as source:
                            if hashlib.file_digest(source,'sha256').hexdigest()!=artifact['sha256']:raise ValueError('Correction artifact digest mismatch')
                    tokenizer=(BertTokenizerFast if name=='bart4csc' else AutoTokenizer).from_pretrained(path,local_files_only=True)
                    model=AutoModelForSeq2SeqLM.from_pretrained(path,local_files_only=True,use_safetensors=True).eval()
                    models[name]=(model,tokenizer)
                result=dict(loadMs=(time.perf_counter()-start)*1000,torch=torch.__version__)
            elif request['op']=='infer':
                model,tokenizer=models[request['model']]
                text=request['text']
                prompt=text if request['model']=='bart4csc' else 'Generate the correct transcription for the following n-best list of ASR hypotheses: \n\n1. '+text
                encoded=tokenizer(prompt,return_tensors='pt',truncation=False)
                if encoded['input_ids'].shape[1]>512:raise ValueError('Correction input exceeds frozen budget')
                encoded.pop('token_type_ids',None)
                start=time.perf_counter()
                with torch.inference_mode():
                    ids=model.generate(**encoded,do_sample=False,num_beams=1,max_new_tokens=192)
                elapsed=(time.perf_counter()-start)*1000
                generated=ids[0,1:].tolist()
                eos=model.generation_config.eos_token_id
                eos_set=set(eos if isinstance(eos,list) else [eos])
                raw_decoded=tokenizer.decode(ids[0],skip_special_tokens=True,clean_up_tokenization_spaces=True)
                result=dict(text=detokenize(raw_decoded,request['model']),rawDecoded=raw_decoded,rawTokenIds=ids[0].tolist(),inputTokens=int(encoded['input_ids'].shape[1]),outputTokens=len(generated),truncated=len(generated)>=192 or not any(t in eos_set for t in generated),generationMs=elapsed,peakRssBytes=resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,prompt=prompt)
            elif request['op']=='close':
                print(json.dumps(dict(ok=True,closed=True)),flush=True);break
            else:raise ValueError('Unknown operation')
            print(json.dumps(dict(ok=True,**result),ensure_ascii=False),flush=True)
        except Exception as error:
            print(json.dumps(dict(ok=False,error=str(error))),flush=True)


if __name__=='__main__':main()
