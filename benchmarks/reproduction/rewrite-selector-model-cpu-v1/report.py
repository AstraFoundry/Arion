"""Replay complete model-choice evidence without attributing final-body quality."""
import argparse
import hashlib
import importlib.util
import json
import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
RUN = ROOT/'benchmarks/runs/rewrite-selector-model-cpu-v1/run'


def read(path):
    return json.loads(path.read_text())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rowhash(value):
    return hashlib.sha256(json.dumps(value, ensure_ascii=False, sort_keys=True).encode()).hexdigest()


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


before = list(sys.path)
sys.path.insert(0, str(HERE.parent))
try:
    ownership = module('model_choice_ownership', HERE.parent/'gguf_template_identity.py')
finally:
    sys.path[:] = before
pipeline = module('model_choice_pipeline', HERE.parent/'rewrite-selector-screen-v1/pipeline.py')
ARMS = ['baseline', 'balanced']


def assessment(row, targets):
    target = targets[row['id']]
    required = target['requiredRepairs']
    if row['renderError'] is not None:
        return dict(conforms=False, harmful=False, missed=False, required=len(required), repaired=0)
    wrong = {k: v for k, v in row['selection'].items() if v not in target['allowedChoices'][k]}
    return dict(conforms=not wrong, harmful=any(v == 'replace' for v in wrong.values()),
                missed=any(v == 'keep' for v in wrong.values()), required=len(required),
                repaired=sum(row['selection'][k] == 'replace' for k in required))


def validate(rows, manifest, review, outcomes, decision, directory=RUN):
    freeze = read(HERE/'freeze.json')
    for path, digest in freeze['files'].items():
        if sha(ROOT/path) != digest:
            raise ValueError('Changed frozen model-comparison source: '+path)
    preps_path = ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/preparations.jsonl'
    preps = {p['sample']['id']: p for p in map(json.loads, preps_path.read_text().splitlines())}
    refs_path = ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl'
    refs = {(r['arm'], r['id']): r for r in map(json.loads, refs_path.read_text().splitlines())}
    targets = {t['id']: t for t in read(ROOT/'benchmarks/corpora/rewrite-selector-screen-v1/evaluator.json')}
    active = {i for i, p in preps.items() if p['prepared']['hypotheses']}
    if active != set(targets) or len(preps) != 457:
        raise ValueError('Changed frozen source scope')
    expected = dict(status='complete', arms=ARMS, model=freeze['model'], runtimeFiles=freeze['runtimeFiles'],
                    freezeSha256=sha(HERE/'freeze.json'), replayedSourcePreparations=457,
                    newGoPreparations=0, newAsrGenerations=0, newOcrGenerations=0, formatterGenerations=0,
                    preparationReference=str(preps_path.relative_to(ROOT)), preparationsSha256=sha(preps_path),
                    sourceTreatmentCount=914, expectedGenerations=202, generations=202,
                    selectorGenerations=202, skippedSourceTreatments=712)
    if any(manifest[k] != v for k, v in expected.items()) or len(rows) != 202 or len(outcomes) != 914:
        raise ValueError('Incomplete or invented model-comparison execution')
    workers = manifest['workers']
    if len(workers) != 8 or {(w['arm'], w['scene']) for w in workers} != {(a, s) for a in ARMS for s in ['work', 'email', 'prompt', 'chat']}:
        raise ValueError('Missing scene/model worker')
    template = read(HERE.parent/'rewrite-selector-model-v1/sources/gguf-header-inspection.json')['metadata']['tokenizer.chat_template']
    for worker in workers:
        for flag, value in [('--model', str(ROOT/freeze['modelPath'])), ('--n-gpu-layers', '0'), ('--ctx-size', '4096'), ('--threads', '4'), ('--threads-batch', '4'), ('--parallel', '1'), ('--reasoning', 'off')]:
            if worker['command'][worker['command'].index(flag)+1] != value:
                raise ValueError('Changed actual native worker treatment')
        props = read(directory/(worker['name']+'-props.json'))
        if props['chat_template'] != template:
            raise ValueError('Unexpected actual model template')
        if 'offloaded 0/41 layers to GPU' not in (directory/(worker['name']+'-server.log')).read_text():
            raise ValueError('Unexpected actual Granite layer placement')
    labels = {(j['variant'], j['id']): j for j in review['reviews']}
    if len(labels) != len(rows):
        raise ValueError('Missing unique personal reviews')
    actual = {}
    for row in rows:
        key = (row['arm'], row['id'])
        prep = preps[row['id']]
        ref = refs[key]
        label = labels[(row['variant'], row['id'])]
        if key in actual or key[0] not in ARMS or key[1] not in active:
            raise ValueError('Unexpected or duplicate selector observation')
        if any(row[k] != v for k, v in prep['sample'].items()) or row['sourceControl'] != prep['sourceControl']:
            raise ValueError('Changed source identity')
        expected_reference = dict(path=str(refs_path.relative_to(ROOT)), arm=row['arm'], rowSha256=rowhash(ref), sameRequest=True)
        if row['comparisonReference'] != expected_reference or row['request'] != ref['request'] or row['prepared'] != prep['prepared']:
            raise ValueError('Changed comparison request or candidate coverage')
        if row['cleanupObservation'] != prep['cleanupObservation'] or row['cleanup'] != prep['cleanupObservation']['cleanup'] or row['preparationRowSha256'] != rowhash(prep):
            raise ValueError('Changed actual preparation lineage')
        if row['sourcePreparation'] != dict(kind='replayed-frozen-actual-go-preparation', modelSelectorCalled=True, formatterExecuted=False) or row['actualContextUsed'] != row['context']:
            raise ValueError('Changed execution/context claim')
        if row['stage'] != 'selection' or row['renderedBody'] is not None or 'pipelineMs' in row or 'cleanupIpcMs' in row:
            raise ValueError('Invented body or new-cleanup/end-to-end timing')
        prompt = ownership.identity(ROOT, freeze, pipeline, row['arm'], 'selection', row['scene'], row['request'])
        if any(row[k] != v for k, v in prompt.items()) or row['modelSha256'] != freeze['model']['sha256']:
            raise ValueError('Changed actual prompt/model identity')
        raw = row['response']['choices'][0]['message']['content']
        if row['response']['choices'][0]['finish_reason'] != 'stop':
            raise ValueError('Incomplete native selector completion')
        if row['renderError'] is None:
            edited, edits = pipeline.candidates_for(row['arm']).apply(raw, row['prepared'])
            if row['selection'] != json.loads(raw) or row['editedSource'] != edited or row['edits'] != edits:
                raise ValueError('Changed raw selector/edited-source evidence')
        elif row['selection'] is not None or row['editedSource'] is not None:
            raise ValueError('Invalid selection substituted source')
        score = assessment(row, targets)
        if label['rowSha256'] != rowhash(row) or label['choiceConforms'] != score['conforms'] or label['harmfulChoice'] != score['harmful'] or label['missedRepair'] != score['missed'] or label['sceneUseful'] is not None or label['faithful'] != (None if row['renderError'] else not score['harmful']):
            raise ValueError('Changed review or invented final style')
        actual[key] = row
    if set(actual) != {(a, i) for a in ARMS for i in active} or manifest['stageProtocolFailures'] != sum(r['renderError'] is not None for r in rows):
        raise ValueError('Missing candidate request or protocol failure')
    seen = set()
    for outcome in outcomes:
        key = (outcome['arm'], outcome['id'])
        called = outcome['id'] in active
        row = actual.get(key)
        status = 'selection-invalid' if called and row['renderError'] else 'selected' if called else 'no-hypotheses'
        if outcome != dict(id=outcome['id'], arm=outcome['arm'], preparationRowSha256=rowhash(preps[outcome['id']]), formatterExecuted=False,
                           status=status, modelSelectorCalled=called, resultRowSha256=rowhash(row) if called else None):
            raise ValueError('Changed complete source-treatment outcome')
        seen.add(key)
    if seen != {(a, i) for a in ARMS for i in preps} or len(seen) != len(outcomes):
        raise ValueError('Missing or duplicate source-level skip')
    passed = [a for a in ARMS if all(labels[(r['variant'], r['id'])]['choiceConforms'] for r in rows if r['arm'] == a)]
    if decision['screenPassedArms'] != passed or decision['qualified'] or decision['productionActivated'] or decision['freshExecuted'] or decision['formatterExecuted']:
        raise ValueError('Unsupported model adoption or full-chain claim')
    return labels, targets


def checked(directory=RUN):
    rows = list(map(json.loads, (directory/'results.jsonl').read_text().splitlines()))
    manifest, review, decision = [read(directory/name) for name in ['manifest.json', 'review.json', 'selection.json']]
    outcomes = list(map(json.loads, (directory/'case-outcomes.jsonl').read_text().splitlines()))
    if any(v != sha(directory/'results.jsonl') for v in [manifest['resultsSha256'], review['resultsSha256'], decision['resultsSha256']]) or manifest['outcomesSha256'] != sha(directory/'case-outcomes.jsonl') or decision['reviewSha256'] != sha(directory/'review.json'):
        raise ValueError('Changed completed model-comparison bytes')
    labels, targets = validate(rows, manifest, review, outcomes, decision, directory)
    pairs = read(directory/'paired-comparison.json')['records']
    references = {(r['arm'], r['id']): r for r in map(json.loads, (ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
    current = {(r['arm'], r['id']): r for r in rows}
    seen = set()
    for pair in pairs:
        key = (pair['arm'], pair['id'])
        row, prior = current[key], references[key]
        if key in seen or pair != dict(id=row['id'], arm=row['arm'], rowSha256=rowhash(row), reference=row['comparisonReference'],
                                     sameRequest=True, samePrepared=True, changedChoice=row['selection'] != prior['selection'],
                                     priorChoice=prior['selection'], newChoice=row['selection'],
                                     priorConforms=assessment(prior, targets)['conforms'], newConforms=assessment(row, targets)['conforms']):
            raise ValueError('Changed paired model-choice comparison')
        seen.add(key)
    if set(current) != seen or len(pairs) != 202:
        raise ValueError('Incomplete paired comparison')
    return rows, labels, targets


def collect():
    rows, labels, targets = checked()
    groups = {}
    for arm in ARMS:
        subset = [r for r in rows if r['arm'] == arm]
        scores = [assessment(r, targets) for r in subset]
        groups[arm] = dict(cases=len(subset), conforming=sum(s['conforms'] for s in scores), harmful=sum(s['harmful'] for s in scores),
                           missed=sum(s['missed'] for s in scores), requiredRepairs=sum(s['required'] for s in scores),
                           repaired=sum(s['repaired'] for s in scores), protocolFailures=sum(r['renderError'] is not None for r in subset))
    pairs = read(RUN/'paired-comparison.json')['records']
    comparison = dict(sameRequests=202, changedChoices=sum(p['changedChoice'] for p in pairs),
                      improvements=sum(not p['priorConforms'] and p['newConforms'] for p in pairs),
                      regressions=sum(p['priorConforms'] and not p['newConforms'] for p in pairs),
                      limits=read(RUN/'paired-comparison.json')['limits'])
    return dict(newGenerations=len(rows), replayedSourcePreparations=457, newGoPreparations=0, sourceTreatments=914,
                skippedSourceTreatments=712, finalBodies=0, groups=groups, selection=read(RUN/'selection.json'),
                pairedComparison=comparison, limits=read(HERE/'protocol.json')['limits']), rows, labels


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    summary, rows, labels = collect()
    args.out.mkdir(parents=True, exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
    lines = ['# Granite selector development comparison', '', 'All actual model choices personally reviewed. Source preparation is explicitly replayed; no new final bodies, native insertion or minimum-device acceptance.', '']
    for row in rows:
        lines += [f"## {row['arm']} / {row['id']} / {row['scene']}", '', '```json', json.dumps({k: row[k] for k in ['source', 'context', 'editedSource', 'edits', 'request']}, ensure_ascii=False, indent=2), '```', '', '```text', row['response']['choices'][0]['message']['content'], '```', '', labels[(row['variant'], row['id'])]['note'], '']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
