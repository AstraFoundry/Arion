"""Complete model-screen checks with exact native-log bytes and explicit placement/template policy.

Derived from the immutable historical CPU comparison verifier. Historical source
remains frozen for archive replay; new model experiments supply actual policies.
"""
import json


def validate(rows, manifest, review, outcomes, decision, directory, *, policy):
    read, sha, rowhash = policy.read, policy.sha, policy.rowhash
    ROOT, HERE, ARMS = policy.root, policy.here, policy.arms
    pipeline, ownership, assessment = policy.pipeline, policy.ownership, policy.assessment
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
    template = read(policy.template_path)['metadata']['tokenizer.chat_template']
    for worker in workers:
        for flag, value in [('--model', str(ROOT/freeze['modelPath'])), ('--n-gpu-layers', str(policy.gpu_layers)), ('--ctx-size', '4096'), ('--threads', '4'), ('--threads-batch', '4'), ('--parallel', '1'), ('--reasoning', 'off')]:
            if worker['command'][worker['command'].index(flag)+1] != value:
                raise ValueError('Changed actual native worker treatment')
        props = read(directory/(worker['name']+'-props.json'))
        if props['chat_template'] != template:
            raise ValueError('Unexpected actual model template')
        if policy.offload_line.encode() not in (directory/(worker['name']+'-server.log')).read_bytes():
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

