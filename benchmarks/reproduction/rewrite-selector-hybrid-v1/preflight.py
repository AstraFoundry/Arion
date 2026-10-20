"""Measure native hybrid-model protocol, unrestricted output and cache parity."""
import argparse
import hashlib
import json
import pathlib
import sys
import urllib.error

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))
import gguf_rewrite as engine
from gguf_session_v2 import Session
ROOT = engine.ROOT


def rowhash(value):
    return hashlib.sha256(json.dumps(value, ensure_ascii=False, sort_keys=True).encode()).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    freeze = engine.read(HERE/'freeze.json')
    engine.MODEL = ROOT/freeze['modelPath']
    engine.verify(freeze)
    plan = engine.read(HERE/'preflight-plan.json')
    out = args.out.resolve()
    out.mkdir(parents=True, exist_ok=False)
    manifest = dict(status='running', plannedAttempts=12, completedResponses=0, attempts=0, workers=[],
                    model=freeze['model'], runtimeFiles=freeze['runtimeFiles'], freezeSha256=engine.sha(HERE/'freeze.json'),
                    newGoPreparations=0, newAsrGenerations=0, newOcrGenerations=0, formatterGenerations=0, scope=plan['limits'])
    engine.write(out/'manifest.json', manifest)
    records = []
    with (out/'results.jsonl').open('x') as stream:
        for placement in plan['placements']:
            name = 'hybrid-preflight-'+str(placement)
            worker = None
            try:
                worker = Session(out, name, 18825, gpu_layers=placement, verbosity=5)
                manifest['workers'].append(dict(name=name, placement=placement, command=worker.command, startupMs=worker.startup_ms, startupError=None))
                for job in plan['jobs']:
                    row = dict(id=str(placement)+'-'+job['id'], placement=placement, job=job['id'],
                               sourceReference=job['sourceReference'], request=job['request'], command=worker.command,
                               response=None, httpError=None, generationMs=None, residentRssBytes=None)
                    try:
                        row['response'], row['generationMs'], row['residentRssBytes'] = worker.generate(job['request'])
                    except urllib.error.HTTPError as error:
                        row['httpError'] = dict(code=error.code, body=error.read().decode())
                    records.append(row)
                    stream.write(json.dumps(row, ensure_ascii=False)+'\n')
                    stream.flush()
                    manifest.update(attempts=len(records), completedResponses=sum(r['response'] is not None for r in records))
                    engine.write(out/'manifest.json', manifest)
                    print(row['id'], row['httpError'] or repr(row['response']['choices'][0]['message']['content']), flush=True)
            except Exception as error:
                manifest['workers'].append(dict(name=name, placement=placement, startupError=str(error), activeRequests=sum(r['placement']==placement for r in records)))
                engine.write(out/'manifest.json', manifest)
                raise
            finally:
                if worker:
                    worker.close()
    manifest.update(status='complete', errors=sum(r['httpError'] is not None for r in records), resultsSha256=engine.sha(out/'results.jsonl'))
    engine.write(out/'manifest.json', manifest)


if __name__ == '__main__':
    main()
