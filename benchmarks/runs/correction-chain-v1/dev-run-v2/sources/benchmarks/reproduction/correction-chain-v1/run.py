"""Run actual ASR -> correction -> native intent rewriting with isolated labels."""
import argparse
import hashlib
import json
import os
import pathlib
import random
import selectors
import subprocess
import sys
import time

ROOT=pathlib.Path(__file__).resolve().parents[3]
HERE=pathlib.Path(__file__).resolve().parent
PRIVATE=ROOT/'.workspace/correction-chain'
FIXTURES=ROOT/'benchmarks/corpora/correction-chain-v1'


def sha(path):
    with pathlib.Path(path).open('rb') as source:return hashlib.file_digest(source,'sha256').hexdigest()


def write_json(path,value):
    with pathlib.Path(path).open('x') as target:json.dump(value,target,ensure_ascii=False,indent=2);target.write('\n')


class Worker:
    def __init__(self,command,log):
        self.log=pathlib.Path(log).open('x')
        self.process=subprocess.Popen([str(x) for x in command],stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=self.log,text=True,bufsize=1)
        self.selector=selectors.DefaultSelector();self.selector.register(self.process.stdout,selectors.EVENT_READ)

    def call(self,request):
        start=time.perf_counter()
        self.process.stdin.write(json.dumps(request,ensure_ascii=False)+'\n');self.process.stdin.flush()
        if not self.selector.select(180):raise TimeoutError('Worker response deadline')
        line=self.process.stdout.readline()
        if not line:raise RuntimeError('Worker exited without a response')
        response=json.loads(line)
        if not response.get('ok'):raise RuntimeError(response)
        response['parentMs']=(time.perf_counter()-start)*1000
        return response

    def close(self):
        try:
            response=self.call(dict(op='close'))
            if not response.get('closed'):raise RuntimeError('Unconfirmed worker cleanup')
            self.process.stdin.close();self.process.wait(timeout=15)
            if self.process.returncode:raise RuntimeError('Worker nonzero exit')
        finally:
            if self.process.poll() is None:self.process.kill();self.process.wait()
            self.selector.close();self.log.close()


def model_files():
    driver=ROOT/'.workspace/benchmarks/driver.json'
    assets=json.loads(driver.read_text())['artifacts']
    def files(marker):
        chosen=[x for x in assets if marker in x['id']]
        paths={}
        records=[]
        for item in chosen:
            path=(driver.parent/item['path']).resolve()
            if sha(path)!=item['sha256']:raise ValueError('Existing acoustic model digest mismatch')
            role=next(role for role in ['encoder','decoder','joiner','tokens'] if role in path.name)
            paths[role]=str(path);records.append(dict(path=str(path),sha256=item['sha256']))
        if set(paths)!={'encoder','decoder','joiner','tokens'}:raise ValueError('Incomplete acoustic model')
        return paths,records
    online,first=files('x-asr-zh-en-160ms-streaming')
    offline,second=files('x-asr-zh-en-offline')
    return dict(op='load',online=online,offline=offline,artifacts=first+second)


def provenance(config,corpus):
    runtime=ROOT/'apps/desktop/resources/darwin-arm64/llm'
    model=ROOT/'.workspace/ablation-models/qwen3-1.7b'
    artifacts={}
    for directory in [runtime,model]:
        for path in directory.rglob('*'):
            if path.is_file():artifacts[str(path.relative_to(ROOT))]=sha(path)
    for name in ['arion-rewrite-bench','arion-llm']:
        path=PRIVATE/name;artifacts[str(path.relative_to(ROOT))]=sha(path)
    sources={str(p.relative_to(ROOT)):sha(p) for p in HERE.glob('*') if p.is_file()}
    for base in ['apps/desktop/infrastructure/llm','apps/desktop/cmd/arion-rewrite-bench','apps/desktop/cmd/arion-llm','packages/core/domain']:
        for path in (ROOT/base).rglob('*.go'):sources[str(path.relative_to(ROOT))]=sha(path)
    return dict(status='running',kind=config['kind'],config=config,inputsSha256=sha(FIXTURES/'inputs.json'),labelsSha256=sha(FIXTURES/'labels.json'),fixtureManifestSha256=sha(FIXTURES/'manifest.json'),caseIds=[s['id'] for s in corpus],
        artifactHashes=artifacts,sourceHashes=sources,gitRevision=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip(),dirty=subprocess.check_output(['git','status','--porcelain'],cwd=ROOT,text=True),
        hardware={k:subprocess.check_output(command,text=True).strip() for k,command in {'cpu':['sysctl','-n','machdep.cpu.brand_string'],'ramBytes':['sysctl','-n','hw.memsize'],'os':['sw_vers'],'power':['pmset','-g']}.items()})


def main():
    parser=argparse.ArgumentParser();parser.add_argument('--split',choices=['dev','test'],required=True);parser.add_argument('--out',required=True);args=parser.parse_args()
    out=pathlib.Path(args.out).resolve();out.mkdir(parents=True,exist_ok=False)
    config=json.loads((HERE/'config.json').read_text())
    manifest=json.loads((FIXTURES/'manifest.json').read_text())
    for name,digest in manifest['files'].items():
        if sha(FIXTURES/name)!=digest:raise ValueError('Frozen fixture digest mismatch')
    samples=[s for s in json.loads((FIXTURES/'inputs.json').read_text())['samples'] if s['split']==args.split]
    warmups=[s for s in json.loads((FIXTURES/'inputs.json').read_text())['samples'] if s['split']=='dev'][:2]
    random.Random(config['seed']).shuffle(samples)
    meta=provenance(config,samples);meta['split']=args.split
    def save():
        temporary=out/'run.tmp';temporary.write_text(json.dumps(meta,ensure_ascii=False,indent=2)+'\n');temporary.replace(out/'run.json')
    save()
    active=None
    try:
        acoustic_request=model_files();meta['acousticArtifacts']=acoustic_request;save()
        active=Worker([ROOT/'.workspace/phonetic-venv/bin/python',HERE/'acoustic.py'],out/'acoustic.stderr')
        meta['acousticLoad']=active.call(acoustic_request)
        acoustic={}
        with (out/'acoustic.jsonl').open('x') as rows,(out/'warmups.jsonl').open('x') as warms:
            for sample in warmups:
                audio={**sample['audio'],'path':str(FIXTURES/sample['audio']['path'])}
                result=active.call(dict(op='infer',audio=audio));result.update(caseId=sample['id'],audioSha256=sample['audio']['sha256'])
                warms.write(json.dumps(result,ensure_ascii=False)+'\n');warms.flush()
            for sample in samples:
                if sample['id'] in acoustic:continue
                audio={**sample['audio'],'path':str(FIXTURES/sample['audio']['path'])}
                result=active.call(dict(op='infer',audio=audio));result.update(caseId=sample['id'],audioSha256=sample['audio']['sha256']);acoustic[sample['id']]=result
                rows.write(json.dumps(result,ensure_ascii=False)+'\n');rows.flush();print('asr',sample['id'],flush=True)
        active.close();active=None
        correction_models={name:str(PRIVATE/'models'/name) for name in ['bart4csc','flanec']}
        meta['correctionManifests']={name:json.loads((pathlib.Path(path)/'manifest.json').read_text()) for name,path in correction_models.items()};save()
        active=Worker([PRIVATE/'venv/bin/python',HERE/'correct.py'],out/'correct.stderr')
        meta['correctionLoad']=active.call(dict(op='load',models=correction_models))
        correction={}
        with (out/'correction.jsonl').open('x') as rows,(out/'correction-warmups.jsonl').open('x') as warms:
            for language,text in [('zh','今天先检查邮件。'),('en','Keep the original document.')]:
                result=active.call(dict(op='infer',model=config['correctionModels'][language],text=text))
                warms.write(json.dumps(dict(language=language,**result),ensure_ascii=False)+'\n');warms.flush()
            for sample in samples:
                text=acoustic[sample['id']]['streaming']['text'];model=config['correctionModels'][sample['language']]
                result=active.call(dict(op='infer',model=model,text=text));result.update(caseId=sample['id'],model=model,inputText=text)
                correction[sample['id']]=result;rows.write(json.dumps(result,ensure_ascii=False)+'\n');rows.flush();print('correct',sample['id'],flush=True)
        active.close();active=None
        rewrite_inputs=[]
        for sample in samples:
            case=sample['id']
            sources={'asr_llm':acoustic[case]['streaming']['text'],'offline_llm':acoustic[case]['offline']['text']}
            if not correction[case]['truncated']:sources['asr_correct_llm']=correction[case]['text']
            for arm,text in sources.items():
                rewrite_inputs.append(dict(id=case+'--'+arm,sourceText=text,scenario=sample['scene'],mode='clean',language=sample['language'],approvedMemory=[]))
        random.Random(config['seed']).shuffle(rewrite_inputs)
        rewrite_inputs=[dict(id='warmup-'+language,sourceText=text,scenario='chat',mode='clean',language=language,approvedMemory=[]) for language,text in [('zh','今天先检查邮件。'),('en','Keep the original document.')]]+rewrite_inputs
        write_json(out/'rewrite-inputs.json',dict(samples=rewrite_inputs))
        command=[str(PRIVATE/'arion-rewrite-bench'),'-corpus',str(out/'rewrite-inputs.json'),'-worker',str(PRIVATE/'arion-llm'),'-runtime',str(ROOT/'apps/desktop/resources/darwin-arm64/llm'),'-model',str(ROOT/'.workspace/ablation-models/qwen3-1.7b'),'-variant','correction-chain-v1','-live=false','-correct=false','-out',str(out/'rewrite.jsonl')]
        with (out/'rewrite.stderr').open('x') as log:subprocess.run(command,stderr=log,check=True,timeout=1800)
        meta['status']='completed'
        meta['outputHashes']={p.name:sha(p) for p in out.iterdir() if p.is_file() and p.name!='run.json'}
        save()
    except BaseException as error:
        meta['status']='failed';meta['failure']=repr(error);save()
        if active and active.process.poll() is None:active.process.kill();active.process.wait()
        raise


if __name__=='__main__':main()
