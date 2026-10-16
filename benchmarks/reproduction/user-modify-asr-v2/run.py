"""Audio-only memory interventions with explicit first/second-pass provenance."""
import argparse,importlib.metadata,json,pathlib,random,resource,subprocess,sys,time,wave
import policy


def write_json(path,value):path.write_text(json.dumps(value,ensure_ascii=False,indent=2)+'\n')


def verify_model(directory,fixtures):
    metadata=json.loads((fixtures/'model-metadata.json').read_text());hashes={}
    all_meta=json.loads((fixtures/'all-model-files.json').read_text())
    import hashlib
    for item in metadata['assets']+[next(x for x in all_meta['siblings'] if x['rfilename']=='merges.txt')]:
        path=directory/item['rfilename'];h=policy.sha(path);hashes[item['rfilename']]=h
        if item.get('lfs'):
            if h!=item['lfs']['sha256']:raise ValueError('Model LFS digest mismatch')
        else:
            b=path.read_bytes();blob=hashlib.sha1(b'blob '+str(len(b)).encode()+b'\0'+b).hexdigest()
            if blob!=item['blobId']:raise ValueError('Model Git-blob digest mismatch')
    return hashes,metadata['revision']


def main():
    p=argparse.ArgumentParser();p.add_argument('--fixtures',required=True);p.add_argument('--model',required=True);p.add_argument('--out',required=True);p.add_argument('--selection');p.add_argument('--limit',type=int);a=p.parse_args()
    fixtures=pathlib.Path(a.fixtures).resolve();out=pathlib.Path(a.out);out.mkdir(parents=True,exist_ok=False)
    config=json.loads((fixtures/'config.json').read_text());corpus=json.loads((fixtures/'corpus.json').read_text());entries=json.loads((fixtures/'memory.json').read_text())['entries']
    selection=json.loads(pathlib.Path(a.selection).read_text()) if a.selection else None
    arms=['none','legacy_pairs','global_destinations',selection['selectedArm']] if selection else config['arms']
    if selection and selection['memorySha256']!=policy.sha(fixtures/'memory.json'):raise ValueError('Selection memory mismatch')
    inputs=[dict(id=s['id'],audio=s['audio']) for s in corpus['samples']];del corpus
    if a.limit:inputs=inputs[:a.limit]
    for s in inputs:
        if policy.sha(fixtures/s['audio']['path'])!=s['audio']['sha256']:raise ValueError('Audio digest mismatch')
    model_hashes,revision=verify_model(pathlib.Path(a.model).resolve(),fixtures)
    if revision!=config['revision']:raise ValueError('Model revision mismatch')
    run=dict(status='running',kind='recognition-memory-v2',config=config,arms=arms,caseIds=[s['id'] for s in inputs],modelHashes=model_hashes,
             inputHashes={n:policy.sha(fixtures/n) for n in ['corpus.json','config.json','memory.json','model-metadata.json','all-model-files.json','warmup.json']},
             selectionSha256=policy.sha(a.selection) if a.selection else None,
             sourceHashes={str(p):policy.sha(p) for p in [pathlib.Path(__file__),pathlib.Path(policy.__file__),policy.V1_PATH]},
             revision=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),dirty=subprocess.check_output(['git','status','--porcelain'],text=True),
             hardware=dict(os=subprocess.check_output(['sw_vers'],text=True),cpu=subprocess.check_output(['sysctl','-n','machdep.cpu.brand_string'],text=True).strip(),ramBytes=int(subprocess.check_output(['sysctl','-n','hw.memsize'],text=True)),powerMode=subprocess.check_output(['pmset','-g'],text=True)),
             runtime={d.metadata['Name']:d.version for d in importlib.metadata.distributions()},
             limitations=['Full-utterance offline audio, no microphone streaming or native insertion.','Community MLX 8-bit source parity unmeasured.','Single-author synthetic recurrence; natural reads are historical general-speech controls.','Phonetics derive from first-pass text, not independent acoustic evidence.','Baseline runs first; remaining arms shuffled. Logical two-pass latency sums measured first pass, retrieval and re-decode calls.'])
    def save():write_json(out/'run.json',run)
    save()
    try:
        import mlx.core as mx,mlx_audio,numpy as np,pypinyin,editdistance
        from mlx_audio.stt import load
        bases=[pathlib.Path(mx.__file__).parent,pathlib.Path(mlx_audio.__file__).parent,pathlib.Path(pypinyin.__file__).parent,pathlib.Path(editdistance.__file__).parent]
        run['runtimeArtifactHashes']={str(f):policy.sha(f) for base in bases for f in base.rglob('*') if f.is_file() and f.suffix in ['.py','.so','.dylib','.metallib']};run['pythonSha256']=policy.sha(sys.executable)
        mx.random.seed(config['seed']);t=time.perf_counter();model=load(str(pathlib.Path(a.model).resolve()));run['loadMs']=(time.perf_counter()-t)*1000
        def audio_for(path):
            with wave.open(str(path),'rb') as f:
                if f.getframerate()!=16000 or f.getnchannels()!=1 or f.getsampwidth()!=2:raise ValueError('Audio format mismatch')
                return np.frombuffer(f.readframes(f.getnframes()),dtype='<i2').astype(np.float32)/32768
        def infer(audio,context):
            mx.random.seed(config['seed']);mx.synchronize();t=time.perf_counter()
            r=model.generate(audio,system_prompt=context,max_tokens=config['maxTokens'],temperature=0,verbose=False);mx.synchronize()
            return dict(text=r.text,elapsedMs=(time.perf_counter()-t)*1000,promptTokens=r.prompt_tokens,generationTokens=r.generation_tokens,tokenLimitReached=r.generation_tokens>=config['maxTokens'],peakRssBytes=resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,peakMlxBytes=mx.get_peak_memory())
        warm=fixtures/'warmup.wav';warm_meta=json.loads((fixtures/'warmup.json').read_text())
        if policy.sha(warm)!=warm_meta['sha256']:raise ValueError('Warmup digest mismatch')
        run['warmupAudioSha256']=policy.sha(warm);warm_audio=audio_for(warm)
        with (out/'warmups.jsonl').open('x') as w:
            for arm in arms:
                context='' if arm=='none' else (policy.context_for(arm,'麦狗',entries,config)[0])
                r=infer(warm_audio,context);r.update(arm=arm,context=context);w.write(json.dumps(r,ensure_ascii=False)+'\n');w.flush()
        with (out/'calls.jsonl').open('x') as calls,(out/'results.jsonl').open('x') as rows:
            next_call=0;rng=random.Random(config['seed'])
            def call(case,stage,audio,context):
                nonlocal next_call
                next_call+=1;r=infer(audio,context);r.update(callId=next_call,caseId=case,stage=stage,context=context);calls.write(json.dumps(r,ensure_ascii=False)+'\n');calls.flush();return r
            for s in inputs:
                audio=audio_for(fixtures/s['audio']['path']);base=call(s['id'],'first_pass',audio,'')
                baseline=dict(caseId=s['id'],arm='none',text=base['text'],firstPassText=base['text'],selected=[],context='',route='first_pass',callIds=[base['callId']],retrievalMs=0,secondPassMs=0,pipelineMs=base['elapsedMs'],tokenLimitReached=base['tokenLimitReached'])
                rows.write(json.dumps(baseline,ensure_ascii=False)+'\n');rows.flush()
                remaining=[x for x in arms if x!='none'];rng.shuffle(remaining)
                for arm in remaining:
                    t=time.perf_counter();context,selected=policy.context_for(arm,base['text'],entries,config);retrieval_ms=(time.perf_counter()-t)*1000
                    adaptive=arm.startswith('retrieved_')
                    if adaptive and not context:
                        result=base;route='no_retrieval';ids=[base['callId']];second_ms=0
                    else:
                        result=call(s['id'],arm,audio,context);route='audio_redecode' if adaptive else 'single_pass_context';ids=[base['callId'],result['callId']] if adaptive else [result['callId']];second_ms=result['elapsedMs']
                    row=dict(caseId=s['id'],arm=arm,text=result['text'],firstPassText=base['text'],selected=selected,context=context,route=route,callIds=ids,retrievalMs=retrieval_ms,secondPassMs=second_ms,pipelineMs=(base['elapsedMs']+retrieval_ms+second_ms) if adaptive else second_ms,tokenLimitReached=result['tokenLimitReached'])
                    rows.write(json.dumps(row,ensure_ascii=False)+'\n');rows.flush();print(s['id'],arm,route,round(row['pipelineMs']),flush=True)
        run.update(status='complete',modelCalls=next_call,observationCount=len(inputs)*len(arms),resultsSha256=policy.sha(out/'results.jsonl'),callsSha256=policy.sha(out/'calls.jsonl'),warmupsSha256=policy.sha(out/'warmups.jsonl'));save()
    except BaseException as e:
        run.update(status='failed',error=type(e).__name__+': '+str(e));save();raise

if __name__=='__main__':main()
