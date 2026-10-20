"""Replay complete counterbalanced sentence choices without final-body credit."""
import argparse
import importlib.util
import json
import math
import pathlib
import statistics
import sys

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
RUN = ROOT/'benchmarks/runs/rewrite-selector-sentence-choice-v1/run'


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


before = list(sys.path)
sys.path.insert(0, str(HERE.parent))
try:
    pipeline = module('sentence_choice_report_pipeline', HERE/'pipeline.py')
    core = module('sentence_choice_core_evidence', HERE.parent/'rewrite-selector-screen-v1/report.py')
finally:
    sys.path[:] = before
read, sha, rowhash = core.read, core.sha, core.rowhash
core.HERE, core.RUN, core.pipeline = HERE, RUN, pipeline


def assessment(row):
    target = core.targets()[row['id']]
    required = target['requiredRepairs']
    if row['renderError'] is not None:
        return dict(conforms=False, harmful=False, missed=False, required=len(required), repaired=0)
    choices = pipeline.decision_choices(row['arm'], row['selection'], row['prepared'])
    wrong = {key: value for key, value in choices.items() if value not in target['allowedChoices'][key]}
    return dict(conforms=not wrong, harmful=any(value == 'replace' for value in wrong.values()),
                missed=any(value == 'keep' for value in wrong.values()), required=len(required),
                repaired=sum(choices[key] == 'replace' for key in required))


core.assessment = assessment


def validate(rows, manifest, review, outcomes, preps, decision, directory=RUN):
    labels = core.validate(rows, manifest, review, outcomes, preps, decision, directory)
    for row in rows:
        normalized = None if row['renderError'] else pipeline.decision_choices(row['arm'], row['selection'], row['prepared'])
        if labels[(row['variant'], row['id'])]['normalizedChoices'] != normalized:
            raise ValueError('Changed normalized sentence decision')
    return labels


def swap_labels(request):
    result = json.loads(json.dumps(request, ensure_ascii=False))
    swap = {'A': 'B', 'B': 'A'}
    for message in result['messages']:
        if message['role'] == 'user':
            data = json.loads(message['content'])
            for hypothesis in data['hypotheses']:
                old = hypothesis['sentenceChoices']
                hypothesis['sentenceChoices'] = {'A': old['B'], 'B': old['A']}
            message['content'] = json.dumps(data, ensure_ascii=False, separators=(',', ':'))
        elif message['role'] == 'assistant':
            message['content'] = json.dumps({key: swap[value] for key, value in json.loads(message['content']).items()}, separators=(',', ':'))
    for field in result['response_format']['schema']['properties'].values():
        field['enum'] = sorted(swap[value] for value in field['enum'])
    return result


def checked(directory=RUN):
    rows = list(map(json.loads, (directory/'results.jsonl').read_text().splitlines()))
    manifest, review, decision = [read(directory/name) for name in ['manifest.json', 'review.json', 'selection.json']]
    outcomes = list(map(json.loads, (directory/'case-outcomes.jsonl').read_text().splitlines()))
    preps = list(map(json.loads, (directory/'preparations.jsonl').read_text().splitlines()))
    if any(value != sha(directory/'results.jsonl') for value in [manifest['resultsSha256'], review['resultsSha256'], decision['resultsSha256']]) or manifest['outcomesSha256'] != sha(directory/'case-outcomes.jsonl') or manifest['preparationsSha256'] != sha(directory/'preparations.jsonl') or decision['reviewSha256'] != sha(directory/'review.json'):
        raise ValueError('Changed completed sentence-choice bytes')
    labels = validate(rows, manifest, review, outcomes, preps, decision, directory)
    source = read(HERE/'prompts.json')['baseExamples']
    if sha(ROOT/source['path']) != source['sha256']:
        raise ValueError('Changed inherited example source')
    refs = {(r['arm'], r['id']): r for r in map(json.loads, (ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
    current = {(r['arm'], r['id']): r for r in rows}
    pairs = read(directory/'control-parity.json')['records']
    seen = set()
    for pair in pairs:
        key = (pair['arm'], pair['id'])
        row, prior = current[key], refs[key]
        if key in seen or key[0] not in ['baseline', 'balanced'] or row['request'] != prior['request'] or row['prepared'] != prior['prepared']:
            raise ValueError('Changed plain control request or candidates')
        expected = dict(arm=row['arm'], id=row['id'], rowSha256=rowhash(row),
                        referencePath='benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl', referenceRowSha256=rowhash(prior),
                        sameRequest=True, samePrepared=True,
                        sameRawChoice=row['response']['choices'][0]['message']['content'] == prior['response']['choices'][0]['message']['content'],
                        sameEditedSource=row['editedSource'] == prior['editedSource'])
        if pair != expected:
            raise ValueError('Changed actual control parity')
        seen.add(key)
    if len(pairs) != 202 or seen != {(a, r['id']) for a in ['baseline', 'balanced'] for r in rows if r['arm'] == a}:
        raise ValueError('Incomplete plain control coverage')
    comparisons = read(directory/'counterbalance-comparison.json')['records']
    seen = set()
    for pair in comparisons:
        identity = pair['id']
        a, b = current[('sentence-ab', identity)], current[('sentence-ba', identity)]
        if identity in seen or swap_labels(a['request']) != b['request']:
            raise ValueError('Counterbalance changes more than option labels/order')
        expected = dict(id=identity, abRowSha256=rowhash(a), baRowSha256=rowhash(b),
                        sameRequestAfterLabelSwap=True, samePrepared=a['prepared'] == b['prepared'],
                        normalizedChoiceAgrees=labels[(a['variant'], identity)]['normalizedChoices'] == labels[(b['variant'], identity)]['normalizedChoices'],
                        sameEditedSource=a['editedSource'] == b['editedSource'])
        if pair != expected or not pair['samePrepared']:
            raise ValueError('Changed counterbalanced choice comparison')
        seen.add(identity)
    if len(comparisons) != 101 or seen != {r['id'] for r in rows if r['arm'] == 'sentence-ab'}:
        raise ValueError('Incomplete A/B counterbalance coverage')
    return rows, labels


def collect():
    rows, labels = checked()
    groups = {}
    for arm in pipeline.ARMS:
        subset = [r for r in rows if r['arm'] == arm]
        results = [assessment(r) for r in subset]
        groups[arm] = dict(cases=len(subset), conforming=sum(x['conforms'] for x in results),
                           harmful=sum(x['harmful'] for x in results), missed=sum(x['missed'] for x in results),
                           requiredRepairs=sum(x['required'] for x in results), repaired=sum(x['repaired'] for x in results),
                           protocolFailures=sum(r['renderError'] is not None for r in subset),
                           forcedOriginalCases=sum(all(len(field['enum']) == 1 for field in r['request']['response_format']['schema']['properties'].values()) for r in subset),
                           requestMedianMs=statistics.median(r['generationMs'] for r in subset),
                           requestP95Ms=sorted(r['generationMs'] for r in subset)[math.ceil(.95*len(subset))-1],
                           outputTokensMedian=statistics.median(r['response']['usage']['completion_tokens'] for r in subset))
    for arm, group in groups.items():
        free = [row for row in rows if row['arm'] == arm and any(len(field['enum']) > 1 for field in row['request']['response_format']['schema']['properties'].values())]
        group.update(nonForcedCases=len(free), nonForcedConforming=sum(assessment(row)['conforms'] for row in free))
    pairs = read(RUN/'control-parity.json')['records']
    comparisons = read(RUN/'counterbalance-comparison.json')['records']
    current = {(row['arm'], row['id']): row for row in rows}
    quality = dict(conformanceDisagreements=0, harmfulDisagreements=0, bothHarmfulCases=0, bothConformingCases=0, bothConformingDifferentOperations=0)
    for pair in comparisons:
        a, b = current[('sentence-ab', pair['id'])], current[('sentence-ba', pair['id'])]
        sa, sb = assessment(a), assessment(b)
        quality['conformanceDisagreements'] += sa['conforms'] != sb['conforms']
        quality['harmfulDisagreements'] += sa['harmful'] != sb['harmful']
        quality['bothHarmfulCases'] += sa['harmful'] and sb['harmful']
        quality['bothConformingCases'] += sa['conforms'] and sb['conforms']
        quality['bothConformingDifferentOperations'] += sa['conforms'] and sb['conforms'] and not pair['normalizedChoiceAgrees']
    return dict(newGenerations=len(rows), sourcePreparations=457, sourceTreatments=1828, skippedSourceTreatments=1424,
                finalBodies=0, groups=groups, selection=read(RUN/'selection.json'), controlPairs=len(pairs),
                controlRawParity=sum(p['sameRawChoice'] for p in pairs), controlEditedParity=sum(p['sameEditedSource'] for p in pairs),
                counterbalanceQuality=quality, counterbalancePairs=len(comparisons), counterbalanceChoiceAgreement=sum(p['normalizedChoiceAgrees'] for p in comparisons),
                counterbalanceEditedAgreement=sum(p['sameEditedSource'] for p in comparisons),
                limits=read(HERE/'protocol.json')['limits']), rows, labels


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    summary, rows, labels = collect()
    args.out.mkdir(parents=True, exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
    lines = ['# Counterbalanced whole-sentence selection', '', 'Every actual raw output and edited source personally reviewed. Raw labels and normalized operations retained separately. No new final bodies or release acceptance.', '']
    for row in rows:
        label = labels[(row['variant'], row['id'])]
        lines += [f"## {row['arm']} / {row['id']} / {row['scene']}", '', '```json', json.dumps({key: row[key] for key in ['source', 'context', 'request', 'selection', 'editedSource', 'edits']}, ensure_ascii=False, indent=2), '```', '', '```text', row['response']['choices'][0]['message']['content'], '```', '', json.dumps(label['normalizedChoices']), '', label['note'], '']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
