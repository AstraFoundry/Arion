import argparse, hashlib, json, os, pathlib, platform, subprocess, tarfile, time

root = next(p for p in pathlib.Path(__file__).resolve().parents if (p/'models/catalog.json').exists())
parser = argparse.ArgumentParser()
parser.add_argument('--inputs', type=pathlib.Path, required=True)
parser.add_argument('--out', type=pathlib.Path, required=True)
args = parser.parse_args()
out = args.out.resolve()
out.mkdir(parents=True, exist_ok=False)
validator = root/'.workspace/rewrite-rejection-v1/validate'
model = pathlib.Path.home() / 'Library/Application Support/Arion/models/qwen3-1.7b-int4-cpu'
runtime = pathlib.Path('/Applications/Arion.app/Contents/Resources/llm')
def digest(path):
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        for block in iter(lambda: f.read(1024*1024), b''):
            h.update(block)
    return h.hexdigest()
def save(name, obj):
    (out/name).write_text(json.dumps(obj, ensure_ascii=False, indent=2)+'\n')
cases = json.loads(args.inputs.read_text())
save('inputs.json', cases)
catalog = json.loads((root/'models/catalog.json').read_text())
entry = next(x for x in catalog['rewriters'] if x['id'] == model.name)
assets = []
for name,pin in entry['files'].items():
    actual = digest(model/name)
    assert actual == pin['sha256'], name
    assets.append({'name': name, 'sha256': actual, 'source': pin['url']})
pins = json.loads((root/'scripts/llm-runtime.json').read_text())['platforms']['darwin-arm64']['archives']
for pin in pins:
    archive = root/'.workspace/genai'/pin['url'].rsplit('/',1)[1]
    assert digest(archive) == pin['sha256']
    with tarfile.open(archive) as tar:
        for member,dest in pin['members'].items():
            if not dest.endswith('.dylib'): continue
            original = tar.extractfile(member).read()
            # Packaged dylibs can differ after signing; preserve both identities.
            assets.append({'name': dest, 'archiveSha256': pin['sha256'], 'upstreamSha256': hashlib.sha256(original).hexdigest(), 'installedSha256': digest(runtime/dest)})
sources = ['apps/desktop/infrastructure/llm/engine/engine.go','apps/desktop/infrastructure/llm/prompt/prompt.go','apps/desktop/infrastructure/llm/prompt/live.json','apps/desktop/infrastructure/llm/prompt/context.json','packages/core/domain/rewrite.go','packages/core/domain/context.go']
save('manifest.json', {'status': 'running', 'kind': 'authored development diagnostic; not historical session replay', 'platform': platform.platform(), 'assets': assets, 'workerSha256': digest(runtime/'arion-llm'), 'inputsSha256': digest(out/'inputs.json'), 'sources': {s: digest(root/s) for s in sources}, 'gitHead': subprocess.check_output(['git','rev-parse','HEAD'],cwd=root,text=True).strip(), 'gitStatus': subprocess.check_output(['git','status','--short'],cwd=root,text=True), 'threads':4,'maxLength':4096,'maxNewTokens':384,'power': subprocess.check_output(['pmset','-g','batt'],text=True)})
env = dict(os.environ, ORT_LIB_PATH=str(runtime))
with open(out/'stderr.log','w') as stderr:
    p = subprocess.Popen([str(runtime/'arion-llm')], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=stderr, text=True, env=env)
    seq = 0
    def call(op, **kw):
        global seq
        seq += 1
        req = dict(version=2,id=seq,op=op,**kw)
        p.stdin.write(json.dumps(req,ensure_ascii=False)+'\n'); p.stdin.flush()
        line = p.stdout.readline()
        if not line: raise RuntimeError('worker exited')
        response = json.loads(line)
        assert response['id']==seq and response['ok'],response
        return response
    save('load.json', call('load', runtime=str(runtime), model=str(model), threads=4, maxLength=4096, maxNewTokens=384))
    with open(out/'results.jsonl','x') as results:
        for case in cases:
            start=time.monotonic()
            response=call('rewrite',session=case['id'],scene=case.get('scene','chat'),text=case['text'],context=case['context'])
            verdict=json.loads(subprocess.check_output([str(validator)],input=json.dumps({'operation':'validate','mode':'product','text':case['text'],'candidate':response.get('text',''),'context':case['context'],'truncated':response.get('truncated',False)},ensure_ascii=False),text=True))
            row=dict(case=case, response=response, validation=verdict, wallMs=(time.monotonic()-start)*1000)
            results.write(json.dumps(row,ensure_ascii=False)+'\n'); results.flush()
            print(case['id'],verdict['verdict'],repr(response.get('text','')),round(row['wallMs']),flush=True)
    call('close'); p.wait(timeout=10)
m=json.loads((out/'manifest.json').read_text()); m['status']='complete'; m['resultsSha256']=digest(out/'results.jsonl'); save('manifest.json',m)
