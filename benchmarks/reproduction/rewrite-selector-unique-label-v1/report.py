"""Replay unique-label-only requests and complete paired selector evidence."""
import argparse
import importlib.util
import json
import math
import pathlib
import statistics
import sys

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
RUN = ROOT/'benchmarks/runs/rewrite-selector-unique-label-v1/run'


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


before = list(sys.path)
sys.path.insert(0, str(HERE.parent))
try:
    pipeline = module('unique_label_report_pipeline', HERE/'pipeline.py')
    core = module('unique_label_core_evidence', HERE.parent/'rewrite-selector-screen-v1/report.py')
finally:
    sys.path[:] = before
read, sha, rowhash = core.read, core.sha, core.rowhash
core.HERE, core.RUN, core.pipeline = HERE, RUN, pipeline
base_collect = core.collect


def checked(directory=RUN):
    rows = list(map(json.loads, (directory/'results.jsonl').read_text().splitlines()))
    manifest, review, decision = [read(directory/name) for name in ['manifest.json', 'review.json', 'selection.json']]
    outcomes = list(map(json.loads, (directory/'case-outcomes.jsonl').read_text().splitlines()))
    preps = list(map(json.loads, (directory/'preparations.jsonl').read_text().splitlines()))
    if any(value != sha(directory/'results.jsonl') for value in [manifest['resultsSha256'], review['resultsSha256'], decision['resultsSha256']]) or manifest['outcomesSha256'] != sha(directory/'case-outcomes.jsonl') or manifest['preparationsSha256'] != sha(directory/'preparations.jsonl') or decision['reviewSha256'] != sha(directory/'review.json'):
        raise ValueError('Changed complete unique-label evidence')
    labels = core.validate(rows, manifest, review, outcomes, preps, decision, directory)
    layout = read(HERE/'layout.json')
    for source in layout['systemAndExampleSources'].values():
        if sha(ROOT/source['path']) != source['sha256']:
            raise ValueError('Changed actual system/example source')
    references = {(r['arm'], r['id']): r for r in map(json.loads, (ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl').read_text().splitlines())}
    current = {(r['arm'], r['id']): r for r in rows}
    controls = read(directory/'control-parity.json')['records']
    seen = set()
    for pair in controls:
        key = (pair['arm'], pair['id'])
        row, prior = current[key], references[key]
        if key in seen or key[0] not in ['baseline', 'balanced'] or row['request'] != prior['request'] or row['prepared'] != prior['prepared']:
            raise ValueError('Changed plain control request/source')
        expected = dict(arm=row['arm'], id=row['id'], rowSha256=rowhash(row),
                        referencePath='benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl', referenceRowSha256=rowhash(prior),
                        sameRequest=True, samePrepared=True,
                        sameRawChoice=row['response']['choices'][0]['message']['content'] == prior['response']['choices'][0]['message']['content'],
                        sameEditedSource=row['editedSource'] == prior['editedSource'])
        if pair != expected:
            raise ValueError('Changed plain-control parity claim')
        seen.add(key)
    if len(controls) != 202 or seen != {(a, r['id']) for a in ['baseline', 'balanced'] for r in rows if r['arm'] == a}:
        raise ValueError('Incomplete plain-control comparison')
    pairs = read(directory/'unique-label-comparison.json')['records']
    seen = set()
    for pair in pairs:
        key = (pair['arm'], pair['id'])
        row = current[key]
        control_arm = 'balanced' if key[0] == 'no-unique-label-balanced' else 'baseline'
        control = current[(control_arm, row['id'])]
        if key in seen or key[0] not in pipeline.OMITTED or pipeline.transform(row['request'], omit=False) != control['request']:
            raise ValueError('Treatment changes more than unique-label exposure')
        users = [message for message in row['request']['messages'] if message['role'] == 'user']
        if any(h.get('status') == 'unique' for message in users for h in json.loads(message['content'])['hypotheses']):
            raise ValueError('Unverified unique label still exposed')
        omitted = sum('status' not in h for message in users for h in json.loads(message['content'])['hypotheses'])
        expected = dict(arm=row['arm'], id=row['id'], rowSha256=rowhash(row), controlArm=control_arm, controlRowSha256=rowhash(control),
                        omittedUniqueLabels=omitted, sameRequestAfterUniqueRestoration=True,
                        samePrepared=row['prepared'] == control['prepared'], changedChoice=row['selection'] != control['selection'])
        if pair != expected or not pair['samePrepared']:
            raise ValueError('Changed unique-label comparison claim')
        seen.add(key)
    if len(pairs) != 202 or seen != {(a, r['id']) for a in pipeline.OMITTED for r in rows if r['arm'] == a}:
        raise ValueError('Incomplete unique-label comparison')
    return rows, labels


core.checked = checked


def collect():
    summary, rows, labels = base_collect()
    controls = read(RUN/'control-parity.json')['records']
    pairs = read(RUN/'unique-label-comparison.json')['records']
    summary.update(controlPairs=len(controls), controlRawParity=sum(p['sameRawChoice'] for p in controls),
                   controlEditedParity=sum(p['sameEditedSource'] for p in controls),
                   uniqueLabelPairs=len(pairs), changedChoices=sum(p['changedChoice'] for p in pairs),
                   timingScope='Observed serial selector requests only; nearest-rank p95. No startup/Go/formatter/native insertion/concurrentASR/min16 acceptance.')
    for arm, group in summary['groups'].items():
        subset = [r for r in rows if r['arm'] == arm]
        group.update(requestMedianMs=statistics.median(r['generationMs'] for r in subset),
                     requestP95Ms=sorted(r['generationMs'] for r in subset)[math.ceil(.95*len(subset))-1],
                     outputTokensMedian=statistics.median(r['response']['usage']['completion_tokens'] for r in subset))
    return summary, rows, labels


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    summary, rows, labels = collect()
    args.out.mkdir(parents=True, exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
    lines = ['# Unverified unique-label exposure', '', 'Every actual choice personally reviewed. Full decoded inputs, instruction/example answers and request settings retained; only unique-status label exposure changes. No new final bodies or release acceptance.', '']
    for row in rows:
        lines += [f"## {row['arm']} / {row['id']} / {row['scene']}", '', '```json', json.dumps({key: row[key] for key in ['source', 'context', 'request', 'selection', 'editedSource', 'edits']}, ensure_ascii=False, indent=2), '```', '', '```text', row['response']['choices'][0]['message']['content'], '```', '', labels[(row['variant'], row['id'])]['note'], '']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
