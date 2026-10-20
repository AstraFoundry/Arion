"""Review hybrid native probes separately from complete paired selector evidence."""
import argparse
import importlib.util
import json
import pathlib
import sys
import types

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
RUN = ROOT/'benchmarks/runs/rewrite-selector-hybrid-v1/run'
PREFLIGHT = RUN.parent/'preflight'


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


before = list(sys.path)
sys.path.insert(0, str(HERE.parent))
try:
    prior = module('hybrid_model_prior_report', HERE.parent/'rewrite-selector-model-cpu-v1/report.py')
    shared = module('hybrid_shared_model_evidence', HERE.parent/'gguf_model_screen_evidence.py')
finally:
    sys.path[:] = before
read, sha, rowhash, assessment = prior.read, prior.sha, prior.rowhash, prior.assessment
ARMS = ['baseline', 'balanced']
prior.HERE, prior.RUN = HERE, RUN
base_checked = prior.checked
policy = types.SimpleNamespace(read=read, sha=sha, rowhash=rowhash, root=ROOT, here=HERE, arms=ARMS,
                               pipeline=prior.pipeline, ownership=prior.ownership, assessment=assessment,
                               template_path=HERE/'sources/gguf-header-inspection.json', gpu_layers=99,
                               offload_line='offloaded 41/41 layers to GPU')


def validate(rows, manifest, review, outcomes, decision, directory=RUN):
    return shared.validate(rows, manifest, review, outcomes, decision, directory, policy=policy)


prior.validate = validate


def runtime_row(row, reference):
    return dict(id=row['id'], variant=('metal99-' if row['placement']==99 else 'cpu-layer0-')+row['job'],
                cohort='native-runtime-compatibility-probe', modelId='granite-4.0-h-1b', scene=reference['scene'],
                effectiveModelScene='spelling-selection-runtime-probe', templateVersion=reference['templateVersion']+'-runtime-probe',
                source=reference['source'], context=reference['context'], request=row['request'], response=row['response'],
                generationMs=row['generationMs'], residentRssBytes=row['residentRssBytes'], stage='runtime-probe',
                rawReference=dict(path='benchmarks/runs/rewrite-selector-hybrid-v1/preflight/results.jsonl', rowSha256=rowhash(row)),
                expectedDiagnosticChoice={'e0':'replace' if row['sourceReference']['id']=='history-0095' else 'keep'},
                actualNativeCommand=row['command'])


def preflight_checked(directory=PREFLIGHT):
    manifest = read(directory/'manifest.json')
    rows = list(map(json.loads, (directory/'results.jsonl').read_text().splitlines()))
    review = read(directory/'review.json')
    qualification = read(directory/'qualification.json')
    freeze, plan = read(HERE/'freeze.json'), read(HERE/'preflight-plan.json')
    expected = dict(status='complete', plannedAttempts=12, attempts=12, completedResponses=12, errors=0,
                    model=freeze['model'], runtimeFiles=freeze['runtimeFiles'], freezeSha256=sha(HERE/'freeze.json'),
                    newGoPreparations=0, newAsrGenerations=0, newOcrGenerations=0, formatterGenerations=0)
    if any(manifest[key] != value for key, value in expected.items()) or len(rows) != 12:
        raise ValueError('Incomplete actual hybrid preflight')
    if manifest['resultsSha256'] != sha(directory/'results.jsonl') or review['resultsSha256'] != sha(directory/'results.jsonl') or qualification['resultsSha256'] != sha(directory/'results.jsonl') or qualification['freezeSha256'] != sha(HERE/'freeze.json'):
        raise ValueError('Changed hybrid runtime bytes or decision')
    for path, digest in freeze['files'].items():
        if sha(ROOT/path) != digest:
            raise ValueError('Changed frozen hybrid source: '+path)
    refs = {(r['arm'], r['id']): r for r in map(json.loads, (ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
    jobs = {job['id']: job for job in plan['jobs']}
    labels = {item['id']: item for item in review['reviews']}
    workers = {item['placement']: item for item in manifest['workers']}
    if set(workers) != {99, 0} or len(manifest['workers']) != 2 or len(labels) != 12:
        raise ValueError('Missing actual placement or raw-output review')
    seen = set()
    for row in rows:
        identity = (row['placement'], row['job'])
        job, worker = jobs[row['job']], workers[row['placement']]
        ref = refs[(row['sourceReference']['arm'], row['sourceReference']['id'])]
        request = json.loads(json.dumps(ref['request']))
        request['cache_prompt'] = row['job'] in ['other-source', 'cache-on-return']
        if row['job'].startswith('free-'):
            request.pop('response_format')
        if identity in seen or row['request'] != job['request'] or row['request'] != request or row['sourceReference'] != job['sourceReference'] or row['sourceReference']['rowSha256'] != rowhash(ref) or row['command'] != worker['command']:
            raise ValueError('Changed actual runtime probe source/treatment')
        command = worker['command']
        for flag, value in [('--model', str(ROOT/freeze['modelPath'])), ('--n-gpu-layers', str(row['placement'])), ('--threads', '4'), ('--ctx-size', '4096'), ('--parallel', '1'), ('--reasoning', 'off')]:
            if command[command.index(flag)+1] != value:
                raise ValueError('Changed actual native hybrid settings')
        props = read(directory/(worker['name']+'-props.json'))
        if props['chat_template'] != read(policy.template_path)['metadata']['tokenizer.chat_template']:
            raise ValueError('Wrong actual hybrid chat template')
        if 'offloaded '+('41/41' if row['placement']==99 else '0/41')+' layers to GPU' not in (directory/(worker['name']+'-server.log')).read_text():
            raise ValueError('Missing actual hybrid placement')
        if row['httpError'] is not None or row['response']['choices'][0]['finish_reason'] != 'stop':
            raise ValueError('Unexpected unfinished runtime probe')
        raw = row['response']['choices'][0]['message']['content']
        choice = json.loads(raw)
        expected_choice = {'e0': 'replace' if row['sourceReference']['id']=='history-0095' else 'keep'}
        item = labels[row['id']]
        if set(choice) != {'e0'} or choice['e0'] not in ['keep', 'replace'] or item['rowSha256'] != rowhash(row) or item['protocolConforms'] is not True or item['probeConforms'] != (choice == expected_choice) or not item['note']:
            raise ValueError('Changed runtime/task review or invented protocol success')
        seen.add(identity)
    if seen != {(placement, job) for placement in [99, 0] for job in jobs}:
        raise ValueError('Missing unique runtime request')
    parity = {}
    for placement in [99, 0]:
        selected = [r['response']['choices'][0]['message']['content'] for r in rows if r['placement']==placement and r['job'] in ['cache-off-first','cache-on-return','cache-off-return']]
        parity[str(placement)] = len(set(selected)) == 1
    if qualification['cacheRawParity'] != parity or qualification['qualifiedForSelector'] != parity['99'] or qualification['productionQualified']:
        raise ValueError('Unsupported runtime/full-release qualification')
    view = directory.parent/'runtime-view'
    viewed = list(map(json.loads, (view/'results.jsonl').read_text().splitlines()))
    vm, vj = read(view/'manifest.json'), read(view/'review.json')
    expected_rows = [runtime_row(row, refs[(row['sourceReference']['arm'], row['sourceReference']['id'])]) for row in rows]
    if viewed != expected_rows or vm['status'] != 'complete' or vm['resultsSha256'] != sha(view/'results.jsonl') or vj['resultsSha256'] != sha(view/'results.jsonl') or vm['rawResultsSha256'] != sha(directory/'results.jsonl'):
        raise ValueError('Changed raw-to-runtime-view derivation')
    expected_reviews = [dict(id=row['id'], variant=row['variant'], rowSha256=rowhash(row), probeConforms=labels[row['id']]['probeConforms'], note=labels[row['id']]['note']) for row in viewed]
    if vj['reviews'] != expected_reviews:
        raise ValueError('Changed runtime-view task review')
    return rows, labels, qualification


def checked(directory=RUN):
    preflight_checked()
    return base_checked(directory)


prior.checked = checked

def collect():
    probes, probe_labels, qualification = preflight_checked()
    summary, rows, labels = prior.collect()
    summary['preflight'] = dict(attempts=12, completedResponses=12, protocolConforming=12,
                               taskConforming=sum(item['probeConforms'] for item in probe_labels.values()),
                               errors=0, qualification=qualification, finalBodies=0)
    return summary, rows, labels


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    summary, rows, labels = collect()
    args.out.mkdir(parents=True, exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
    lines = ['# Granite hybrid native and selector comparison', '', 'All actual outputs personally reviewed; runtime probes are distinct from selector choices and final-body quality.', '']
    for row in rows:
        lines += [f"## {row['arm']} / {row['id']} / {row['scene']}", '', '```json', json.dumps({key: row[key] for key in ['source', 'context', 'request', 'selection', 'editedSource', 'edits']}, ensure_ascii=False, indent=2), '```', '', '```text', row['response']['choices'][0]['message']['content'], '```', '', labels[(row['variant'], row['id'])]['note'], '']
    for row in preflight_checked()[0]:
        lines += [f"## Native probe / {row['id']}", '', '```json', json.dumps(row, ensure_ascii=False, indent=2), '```', '']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
