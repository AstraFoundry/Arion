"""Run frozen recognition-stage memory arms; gold labels never reach the model."""
import argparse
import hashlib
import importlib.metadata
import json
import os
import pathlib
import random
import resource
import subprocess
import sys
import time


def sha(path):
    h=hashlib.sha256()
    with pathlib.Path(path).open('rb') as f:
        for b in iter(lambda:f.read(1024*1024),b''):h.update(b)
    return h.hexdigest()


def select(entries,c):
    pairs={(e['source'],e['target']) for e in entries}
    eligible=[e for e in entries if e['approved'] and c['logicalDay']-e['updatedDay']<c['expireDays']
              and (e['target'],e['source']) not in pairs]
    today=sorted((e for e in eligible if e['updatedDay']==c['logicalDay']),key=lambda e:(-e['updatedDay'],e['id']))[:c['todayQuota']]
    used={e['id'] for e in today}
    recalled=sorted((e for e in eligible if e['id'] not in used),key=lambda e:(-e['recallCount'],-e['updatedDay'],e['id']))[:c['recallQuota']]
    chosen=today+recalled
    used={e['id'] for e in chosen}
    rest=sorted((e for e in eligible if e['id'] not in used),key=lambda e:(-e['useCount'],-e['updatedDay'],e['id']))
    chosen=(chosen+rest)[:c['totalLimit']]
    result=[];chars=0;seen=set()
    for e in chosen:
        pair=(e['source'],e['target']);size=len(e['source']+' → '+e['target'])+1
        if pair in seen:continue
        if chars+size>c['maxChars']:break
        result.append(e);seen.add(pair);chars+=size
    return result


def contexts(entries,c):
    selected=select(entries,c)
    pairs='\n'.join(e['source']+' → '+e['target'] for e in selected)
    return selected,{'none':'','destinations':' '.join(e['target'] for e in selected),
                     'pairs':pairs,'tagged_pairs':'<USER_MODIFY>\n'+pairs+'\n</USER_MODIFY>',
                     'all_pairs':'\n'.join(e['source']+' → '+e['target'] for e in entries)}


def main():
    p=argparse.ArgumentParser();p.add_argument('--fixtures',required=True);p.add_argument('--model',required=True);p.add_argument('--out',required=True);p.add_argument('--preflight',action='store_true');a=p.parse_args()
    fixtures=pathlib.Path(a.fixtures).resolve();modeldir=pathlib.Path(a.model).resolve();out=pathlib.Path(a.out);out.mkdir(parents=True,exist_ok=False)
    c=json.loads((fixtures/'config.json').read_text());memory=json.loads((fixtures/'memory.json').read_text())['entries'];corpus=json.loads((fixtures/'corpus.json').read_text())
    selected,ctx=contexts(memory,c)
    if [e['id'] for e in selected]!=['arion','mygo','redis','kubernetes','feishu','safari','openai','sql']:raise ValueError('Selection policy contract failed')
    if select([{'id':'r1','source':'a','target':'b','updatedDay':100,'useCount':1,'recallCount':1,'approved':True},{'id':'r2','source':'b','target':'a','updatedDay':100,'useCount':1,'recallCount':1,'approved':True}],c):raise ValueError('Reverse suppression failed')
    # Separate evaluator annotations from the object retained in the inference loop.
    inputs=[{'id':s['id'],'audio':s['audio']} for s in corpus['samples']]
    del corpus
    for s in inputs:
        if sha(fixtures/s['audio']['path'])!=s['audio']['sha256']:raise ValueError('Audio digest mismatch')
    metadata=json.loads((fixtures/'model-metadata.json').read_text())
    if metadata['revision']!=c['revision']:raise ValueError('Model revision mismatch')
    model_hashes={}
    for item in metadata['assets']:
        f=modeldir/item['rfilename']
        digest=sha(f);model_hashes[f.name]=digest
        if item.get('lfs'):
            if digest!=item['lfs']['sha256']:raise ValueError('Weight digest mismatch')
        else:
            data=f.read_bytes();blob=hashlib.sha1(b'blob '+str(len(data)).encode()+b'\0'+data).hexdigest()
            if blob!=item['blobId']:raise ValueError('Model configuration digest mismatch')
    # Tokenizer merges are also used; compare the pinned upstream blob recorded by provisioning.
    merges=modeldir/'merges.txt';model_hashes['merges.txt']=sha(merges)
    prov=json.loads((fixtures/'all-model-files.json').read_text())
    original=next(x for x in prov['siblings'] if x['rfilename']=='merges.txt')
    b=merges.read_bytes()
    if hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()!=original['blobId']:raise ValueError('Merges digest mismatch')
    run={'status':'running','experiment':c,'stage':'offline_audio_recognition','modelHashes':model_hashes,
         'inputHashes':{n:sha(fixtures/n) for n in ['corpus.json','memory.json','config.json','model-metadata.json','all-model-files.json','warmup.json']},
         'selectedIds':[e['id'] for e in selected],'contexts':ctx,
         'hardware':{'os':subprocess.check_output(['sw_vers'],text=True),'cpu':subprocess.check_output(['sysctl','-n','machdep.cpu.brand_string'],text=True).strip(),'ramBytes':int(subprocess.check_output(['sysctl','-n','hw.memsize'],text=True)),'powerMode':subprocess.check_output(['pmset','-g'],text=True)},
         'runtime':{d.metadata['Name']:d.version for d in importlib.metadata.distributions()},
         'source':{'git':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),'dirty':subprocess.check_output(['git','status','--porcelain'],text=True),'runnerSha256':sha(__file__)},
         'limitations':['Single-author TTS diagnostic, not independent natural speech or microphone streaming.','Community 8-bit model; official BF16 conversion parity unmeasured.','No text rewriter, vocabulary replacement or postprocessing in primary arms.','No independent ratings or release acceptance.','Model context is supported; USER_MODIFY semantics are experimental.']}
    runpath=out/'run.json'
    def save():runpath.write_text(json.dumps(run,ensure_ascii=False,indent=2)+'\n')
    save()
    try:
        import mlx.core as mx
        import mlx_audio
        from mlx_audio.stt import load
        import numpy as np
        import wave
        run['runtimeArtifactHashes']={str(f):sha(f) for base in [pathlib.Path(mx.__file__).parent,pathlib.Path(mlx_audio.__file__).parent] for f in base.rglob('*') if f.is_file() and f.suffix in {'.so','.dylib','.metallib','.py'}}
        run['pythonSha256']=sha(sys.executable)
        mx.random.seed(c['seed']);t=time.perf_counter();model=load(str(modeldir));run['loadMs']=(time.perf_counter()-t)*1000
        # Every arm warms on the same separate historical development recording.
        warm=fixtures/'warmup.wav'
        warm_meta=json.loads((fixtures/'warmup.json').read_text())
        if sha(warm)!=warm_meta['sha256']:raise ValueError('Warmup digest mismatch')
        with wave.open(str(warm),'rb') as w:warm_audio=np.frombuffer(w.readframes(w.getnframes()),dtype='<i2').astype(np.float32)/32768
        run['warmupAudioSha256']=sha(warm)
        with (out/'warmups.jsonl').open('x') as f:
            for arm in c['arms']:
                mx.synchronize();t=time.perf_counter();r=model.generate(warm_audio,system_prompt=ctx[arm],max_tokens=c['maxTokens'],temperature=0,verbose=False);mx.synchronize()
                f.write(json.dumps({'arm':arm,'text':r.text,'elapsedMs':(time.perf_counter()-t)*1000,'promptTokens':r.prompt_tokens,'generationTokens':r.generation_tokens},ensure_ascii=False)+'\n');f.flush()
        if a.preflight:
            run['status']='preflight_complete';save();return
        with (out/'results.jsonl').open('x') as f:
            rng=random.Random(c['seed'])
            for s in inputs:
                with wave.open(str(fixtures/s['audio']['path']),'rb') as w:
                    if w.getframerate()!=16000 or w.getnchannels()!=1 or w.getsampwidth()!=2:raise ValueError('Unsupported fixture format')
                    audio=np.frombuffer(w.readframes(w.getnframes()),dtype='<i2').astype(np.float32)/32768
                arms=list(c['arms']);rng.shuffle(arms)
                for arm in arms:
                    mx.random.seed(c['seed']);mx.synchronize();t=time.perf_counter()
                    r=model.generate(audio,system_prompt=ctx[arm],max_tokens=c['maxTokens'],temperature=0,verbose=False);mx.synchronize()
                    row={'caseId':s['id'],'arm':arm,'text':r.text,'elapsedMs':(time.perf_counter()-t)*1000,'promptTokens':r.prompt_tokens,'generationTokens':r.generation_tokens,'tokenLimitReached':r.generation_tokens>=c['maxTokens'],'peakRssBytes':resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,'peakMlxBytes':mx.get_peak_memory()}
                    f.write(json.dumps(row,ensure_ascii=False)+'\n');f.flush();print(s['id'],arm,round(row['elapsedMs']),flush=True)
        run.update(status='complete',resultsSha256=sha(out/'results.jsonl'),warmupsSha256=sha(out/'warmups.jsonl'));save()
    except BaseException as e:
        run.update(status='failed',error=type(e).__name__+': '+str(e));save();raise

if __name__=='__main__':main()
