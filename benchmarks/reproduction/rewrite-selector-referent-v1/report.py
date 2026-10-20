"""Validate bounded choices and separately reviewed public referent descriptions."""
import argparse
import hashlib
import importlib.util
import json
import pathlib
import sys
import math
import statistics

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
RUN = ROOT/'benchmarks/runs/rewrite-selector-referent-v1/run'


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


before = list(sys.path)
sys.path.insert(0, str(HERE.parent))
try:
    pipeline = module('referent_report_pipeline', HERE/'pipeline.py')
    core = module('referent_screen_evidence', HERE.parent/'rewrite-selector-screen-v1/report.py')
finally:
    sys.path[:] = before
read, sha, rowhash = core.read, core.sha, core.rowhash
core.HERE = HERE
core.pipeline = pipeline


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
        label = labels[(row['variant'], row['id'])]
        explained = row['arm'] in pipeline.EXPLANATORY
        normalized = None if row['renderError'] else pipeline.decision_choices(row['arm'], row['selection'], row['prepared'])
        if label['normalizedChoices'] != normalized:
            raise ValueError('Changed normalized choice projection')
        if not explained or row['renderError']:
            if label['descriptionReview'] is not None or label['descriptionConforms'] is not None or label['descriptionOrderConforms'] is not None:
                raise ValueError('Invented public description review')
            continue
        descriptions = label['descriptionReview']
        if set(descriptions) != set(row['selection']):
            raise ValueError('Missing public description review')
        for key, value in row['selection'].items():
            item = descriptions[key]
            if item['text'] != value['referent'] or not (type(item['conforms']) is bool or item['conforms'] is None) or not isinstance(item['note'], str) or not item['note']:
                raise ValueError('Changed or unreviewed public description')
        flags = [item['conforms'] for item in descriptions.values()]
        aggregate = False if any(value is False for value in flags) else None if any(value is None for value in flags) else True
        ordered = all(list(value) == ['referent', 'choice'] for value in row['selection'].values())
        if label['descriptionConforms'] is not aggregate or label['descriptionOrderConforms'] != ordered:
            raise ValueError('Changed public description aggregation/order')
    eligible = [arm for arm in pipeline.ARMS if all(label['choiceConforms'] and (arm not in pipeline.EXPLANATORY or label['descriptionConforms'] is True)
                for row in rows if row['arm'] == arm for label in [labels[(row['variant'], row['id'])]])]
    if decision['eligibleArms'] != eligible:
        raise ValueError('Joint eligibility omits description failures')
    return labels


def checked(directory=RUN):
    rows = list(map(json.loads, (directory/'results.jsonl').read_text().splitlines()))
    manifest, review, decision = [read(directory/name) for name in ['manifest.json', 'review.json', 'selection.json']]
    outcomes = list(map(json.loads, (directory/'case-outcomes.jsonl').read_text().splitlines()))
    preps = list(map(json.loads, (directory/'preparations.jsonl').read_text().splitlines()))
    if any(value != sha(directory/'results.jsonl') for value in [manifest['resultsSha256'], review['resultsSha256'], decision['resultsSha256']]) or manifest['outcomesSha256'] != sha(directory/'case-outcomes.jsonl') or manifest['preparationsSha256'] != sha(directory/'preparations.jsonl') or decision['reviewSha256'] != sha(directory/'review.json'):
        raise ValueError('Changed complete referent-screen bytes')
    labels = validate(rows, manifest, review, outcomes, preps, decision, directory)
    description_file = read(directory/'description-reviews.json')
    expected_descriptions = [dict(arm=row['arm'], id=row['id'], hypothesisId=key, rowSha256=rowhash(row), **item)
                             for row in rows for key, item in (labels[(row['variant'], row['id'])]['descriptionReview'] or {}).items()]
    if description_file['resultsSha256'] != sha(directory/'results.jsonl') or description_file['fields'] != expected_descriptions:
        raise ValueError('Changed complete public-description review')
    references = {(r['arm'], r['id']): r for r in map(json.loads, (ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
    control_pairs = read(directory/'control-budget-comparison.json')['records']
    current = {(r['arm'], r['id']): r for r in rows}
    seen = set()
    for pair in control_pairs:
        key = (pair['arm'], pair['id'])
        row, old = current[key], references[key]
        projected = json.loads(json.dumps(row['request']))
        projected['max_tokens'] = 128
        if key[0] not in ['baseline', 'balanced'] or key in seen or projected != old['request'] or row['prepared'] != old['prepared']:
            raise ValueError('Control changes undeclared request/source fields')
        expected = dict(arm=row['arm'], id=row['id'], rowSha256=rowhash(row),
                        referencePath='benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl', referenceRowSha256=rowhash(old),
                        sameRequestExceptTokenBudget=True, samePrepared=True,
                        sameRawChoice=row['response']['choices'][0]['message']['content'] == old['response']['choices'][0]['message']['content'],
                        sameEditedSource=row['editedSource'] == old['editedSource'])
        if pair != expected:
            raise ValueError('Changed control-budget comparison')
        seen.add(key)
    if len(control_pairs) != 202 or seen != {(arm, row['id']) for arm in ['baseline', 'balanced'] for row in rows if row['arm'] == arm}:
        raise ValueError('Incomplete current control/reference comparison')
    return rows, labels


def collect():
    rows, labels = checked()
    groups = {}
    for arm in pipeline.ARMS:
        subset = [r for r in rows if r['arm'] == arm]
        scores = [assessment(r) for r in subset]
        judgments = [labels[(r['variant'], r['id'])] for r in subset]
        groups[arm] = dict(cases=len(subset), conforming=sum(s['conforms'] for s in scores), harmful=sum(s['harmful'] for s in scores),
                           missed=sum(s['missed'] for s in scores), requiredRepairs=sum(s['required'] for s in scores), repaired=sum(s['repaired'] for s in scores),
                           protocolFailures=sum(r['renderError'] is not None for r in subset),
                           descriptionCases=sum(j['descriptionReview'] is not None for j in judgments),
                           descriptionConforming=sum(j['descriptionConforms'] is True for j in judgments),
                           descriptionIncorrect=sum(j['descriptionConforms'] is False for j in judgments),
                           descriptionUncertain=sum(j['descriptionReview'] is not None and j['descriptionConforms'] is None for j in judgments),
                           jointConforming=sum(j['choiceConforms'] and (arm not in pipeline.EXPLANATORY or j['descriptionConforms'] is True) for j in judgments),
                           requestMedianMs=statistics.median(r['generationMs'] for r in subset),
                           requestP95Ms=sorted(r['generationMs'] for r in subset)[math.ceil(.95*len(subset))-1],
                           outputTokensMedian=statistics.median(r['response']['usage']['completion_tokens'] for r in subset),
                           sampledRssMaxBytes=max(r['residentRssBytes'] for r in subset))
    controls = read(RUN/'control-budget-comparison.json')['records']
    descriptions = read(RUN/'description-reviews.json')['fields']
    explained_labels = [labels[(r['variant'], r['id'])] for r in rows if r['arm'] in pipeline.EXPLANATORY]
    public = dict(cases=len(explained_labels), fields=len(descriptions),
                  conformingFields=sum(item['conforms'] is True for item in descriptions),
                  failedFields=sum(item['conforms'] is False for item in descriptions),
                  failedCases=sum(item['descriptionConforms'] is False for item in explained_labels),
                  correctChoiceWithFailedDescription=sum(item['choiceConforms'] and item['descriptionConforms'] is False for item in explained_labels),
                  observedDescriptionFirstCases=sum(item['descriptionOrderConforms'] is True for item in explained_labels))
    return dict(newGenerations=len(rows), sourcePreparations=457, sourceTreatments=1828, skippedSourceTreatments=1424, finalBodies=0,
                groups=groups, selection=read(RUN/'selection.json'), controlBudgetPairs=len(controls),
                controlRawParity=sum(p['sameRawChoice'] for p in controls), controlEditedParity=sum(p['sameEditedSource'] for p in controls),
                publicDescriptions=public, timingScope='Observed serial selector request times; p95 uses nearest rank. Startup/Go/formatter/native insertion/ASR contention excluded. No full-chain latency or min16 acceptance.',
                limits=read(HERE/'protocol.json')['limits']), rows, labels


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    summary, rows, labels = collect()
    args.out.mkdir(parents=True, exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
    lines = ['# Public referent-description selector comparison', '', 'All actual model responses, public descriptions and source candidates reviewed. Description correctness and choice correctness are separate; no final-body or release quality credit.', '']
    for row in rows:
        label = labels[(row['variant'], row['id'])]
        lines += [f"## {row['arm']} / {row['id']} / {row['scene']}", '', '```json', json.dumps({key: row[key] for key in ['source', 'context', 'request', 'selection', 'editedSource', 'edits']}, ensure_ascii=False, indent=2), '```', '', '```text', row['response']['choices'][0]['message']['content'], '```', '', json.dumps(label['descriptionReview'], ensure_ascii=False), '', label['note'], '']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
