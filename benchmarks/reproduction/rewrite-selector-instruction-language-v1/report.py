"""Replay model-specific English-system choices against unchanged Chinese references."""
import argparse
import importlib.util
import json
import pathlib
import statistics
import math
import sys
import types

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
RUN = ROOT/'benchmarks/runs/rewrite-selector-instruction-language-v1'
MODEL_KEYS = ['qwen', 'hybrid']
ARMS = ['baseline', 'balanced']


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


before = list(sys.path)
sys.path.insert(0, str(HERE.parent))
try:
    prior = module('language_old_model_evidence', HERE.parent/'rewrite-selector-model-cpu-v1/report.py')
    shared = module('language_projected_evidence', HERE.parent/'gguf_language_screen_evidence.py')
    pipeline = module('language_owned_pipeline', HERE/'pipeline.py')
finally:
    sys.path[:] = before
read, sha, rowhash, assessment = prior.read, prior.sha, prior.rowhash, prior.assessment


def validate(rows, manifest, review, outcomes, decision, directory, model_key):
    frozen = read(HERE/'freeze.json')
    policy = types.SimpleNamespace(read=read, sha=sha, rowhash=rowhash, root=ROOT, here=HERE, arms=ARMS,
                                   pipeline=pipeline, ownership=prior.ownership, assessment=assessment,
                                   model_key=model_key, gpu_layers=99, offload_line=frozen['models'][model_key]['offloadLine'],
                                   template=read(HERE/'runtime-templates.json')[model_key]['template'])
    return shared.validate(rows, manifest, review, outcomes, decision, directory, policy=policy)


def checked(model_key, directory=None):
    if model_key not in MODEL_KEYS:
        raise ValueError('Unknown model-language partition')
    directory = RUN/model_key if directory is None else directory
    rows = list(map(json.loads, (directory/'results.jsonl').read_text().splitlines()))
    manifest, review, decision = [read(directory/name) for name in ['manifest.json', 'review.json', 'selection.json']]
    outcomes = list(map(json.loads, (directory/'case-outcomes.jsonl').read_text().splitlines()))
    if any(value != sha(directory/'results.jsonl') for value in [manifest['resultsSha256'], review['resultsSha256'], decision['resultsSha256']]) or manifest['outcomesSha256'] != sha(directory/'case-outcomes.jsonl') or decision['reviewSha256'] != sha(directory/'review.json'):
        raise ValueError('Changed completed language evidence')
    labels, targets = validate(rows, manifest, review, outcomes, decision, directory, model_key)
    templates = read(HERE/'runtime-templates.json')
    for item in templates.values():
        if sha(ROOT/item['source']['path']) != item['source']['sha256']:
            raise ValueError('Changed actual template reference')
    prompt = read(HERE/'prompts.json')
    for item in prompt['sourceSystems'].values():
        if sha(ROOT/item['path']) != item['sha256']:
            raise ValueError('Changed Chinese instruction source')
    if ''.join(item['chinese'] for item in prompt['clauseAlignment']) != read(ROOT/prompt['sourceSystems']['baseline']['path'])['selectionSystem'] or ' '.join(item['english'] for item in prompt['clauseAlignment']) != prompt['selectionSystem']:
        raise ValueError('Changed complete clause translation')
    reference_path = read(HERE/'freeze.json')['models'][model_key]['referencePath']
    references = {(r['arm'],r['id']):r for r in map(json.loads,(ROOT/reference_path).read_text().splitlines())}
    pairs = read(directory/'paired-comparison.json')['records']
    actual = {(r['arm'],r['id']):r for r in rows}
    seen = set()
    for pair in pairs:
        key = (pair['arm'],pair['id'])
        row, ref = actual[key], references[key]
        expected = dict(id=row['id'], arm=row['arm'], rowSha256=rowhash(row), reference=row['comparisonReference'],
                        sameRequestExceptSystemLanguage=True, samePrepared=True,
                        changedChoice=row['selection'] != ref['selection'], priorChoice=ref['selection'], newChoice=row['selection'],
                        priorConforms=assessment(ref,targets)['conforms'], newConforms=assessment(row,targets)['conforms'])
        if key in seen or pair != expected:
            raise ValueError('Changed within-model language pair')
        seen.add(key)
    if seen != set(actual) or len(pairs) != 202:
        raise ValueError('Incomplete within-model language comparison')
    return rows, labels, targets


def collect():
    groups = {}
    all_rows = []
    all_labels = {}
    pairs_summary = {}
    for model_key in MODEL_KEYS:
        rows, labels, targets = checked(model_key)
        all_rows.extend(dict(row, modelKey=model_key) for row in rows)
        all_labels.update({(model_key,*key):value for key,value in labels.items()})
        for arm in ARMS:
            subset = [r for r in rows if r['arm']==arm]
            scores = [assessment(r,targets) for r in subset]
            groups[model_key+'/'+arm] = dict(cases=len(subset), conforming=sum(s['conforms'] for s in scores),
                harmful=sum(s['harmful'] for s in scores), missed=sum(s['missed'] for s in scores),
                requiredRepairs=sum(s['required'] for s in scores), repaired=sum(s['repaired'] for s in scores),
                protocolFailures=sum(r['renderError'] is not None for r in subset),
                requestMedianMs=statistics.median(r['generationMs'] for r in subset),
                requestP95Ms=sorted(r['generationMs'] for r in subset)[math.ceil(.95*len(subset))-1])
        pairs = read(RUN/model_key/'paired-comparison.json')['records']
        pairs_summary[model_key] = dict(cases=202, changedChoices=sum(p['changedChoice'] for p in pairs),
            improvements=sum(not p['priorConforms'] and p['newConforms'] for p in pairs),
            regressions=sum(p['priorConforms'] and not p['newConforms'] for p in pairs))
    return dict(newGenerations=len(all_rows), replayedSourcePreparations=457, newGoPreparations=0,
                sourceTreatments=1828, skippedSourceTreatments=1424, finalBodies=0, groups=groups,
                pairedComparison=pairs_summary, selection={key:read(RUN/key/'selection.json') for key in MODEL_KEYS},
                limits=read(HERE/'protocol.json')['limits']), all_rows, all_labels


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    summary, rows, labels = collect()
    args.out.mkdir(parents=True, exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
    lines = ['# Trusted instruction-language comparison', '', 'Every actual choice/edited source personally reviewed. No new final-body or release evidence.', '']
    for row in rows:
        label = labels[(row['modelKey'], row['variant'], row['id'])]
        lines += [f"## {row['modelKey']} / {row['arm']} / {row['id']} / {row['scene']}", '', '```json', json.dumps({key:row[key] for key in ['source','context','request','selection','editedSource','edits']}, ensure_ascii=False, indent=2), '```', '', '```text', row['response']['choices'][0]['message']['content'], '```', '', label['note'], '']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
