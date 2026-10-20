"""Replay demo-order-only requests and complete paired selector evidence."""
import argparse
import importlib.util
import json
import math
import pathlib
import statistics
import sys

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[2]
RUN = ROOT/'benchmarks/runs/rewrite-selector-demo-order-v1/run'


def module(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    value = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(value)
    return value


before = list(sys.path)
sys.path.insert(0, str(HERE.parent))
try:
    pipeline = module('demo_order_report_pipeline', HERE/'pipeline.py')
    core = module('demo_order_core_evidence', HERE.parent/'rewrite-selector-screen-v1/report.py')
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
        raise ValueError('Changed complete demo-order evidence')
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
    pairs = read(directory/'demo-order-comparison.json')['records']
    seen = set()
    for pair in pairs:
        key = (pair['arm'], pair['id'])
        row = current[key]
        control_arm = pipeline.control_arm(row['arm'])
        control = current[(control_arm, row['id'])]
        if key in seen or key[0] not in pipeline.AUGMENTED or pipeline.restore_request(row['request'],row['arm']) != control['request']:
            raise ValueError('Treatment changes more than existing demonstration order')
        if len(row['request']['messages']) != 22 or len(control['request']['messages']) != 22:
            raise ValueError('Changed original ten-example count')
        expected = dict(arm=row['arm'],id=row['id'],rowSha256=rowhash(row),controlArm=control_arm,controlRowSha256=rowhash(control),
                        unchangedMessageCount=True,sameRequestAfterInversePermutation=True,pairPermutation=layout['pairPermutation'],
                        samePrepared=row['prepared']==control['prepared'],changedChoice=row['selection']!=control['selection'])
        if pair != expected or not pair['samePrepared']:
            raise ValueError('Changed exact demonstration-order comparison claim')
        seen.add(key)
    if len(pairs)!=202 or seen!={(a,r['id']) for a in pipeline.AUGMENTED for r in rows if r['arm']==a}:
        raise ValueError('Incomplete demonstration-order comparison')
    native = directory.parent/'preflight'
    status = read(native/'status.json')
    budget = list(map(json.loads, (native/'requests.jsonl').read_text().splitlines()))
    if status['status'] != 'complete' or status['modelGenerations'] != 0 or status['templateRequests'] != 404 or status['tokenizationRequests'] != 404 or status['model'] != manifest['model'] or status['requestsSha256'] != sha(native/'requests.jsonl'):
        raise ValueError('Changed zero-generation native budget evidence')
    original_preps = {p['sample']['id']:p for p in map(json.loads, (ROOT/'benchmarks/runs/rewrite-selector-screen-v1/run/preparations.jsonl').read_text().splitlines())}
    seen = set()
    for item in budget:
        key = (item['arm'], item['id'])
        row = current[key]
        tokens = item['tokenizeResponse']['tokens']
        if item['sourcePreparationReference'] != dict(path='benchmarks/runs/rewrite-selector-screen-v1/run/preparations.jsonl',rowSha256=rowhash(original_preps[row['id']])) or item['promptTokens'] != row['response']['usage']['prompt_tokens']:
            raise ValueError('Native budget differs from prepared source or actual inference token usage')
        if key in seen or item['request'] != row['request'] or item['requestSha256'] != rowhash(row['request']) or item['modelGenerations'] != 0 or item['promptTokens'] != len(tokens) or not all(isinstance(t, int) for t in tokens) or item['outputReserve'] != 128 or item['contextLimit'] != 4096 or len(tokens)+128 > 4096:
            raise ValueError('Invalid complete-prompt token budget')
        if item['tokenizeRequest'] != dict(add_special=False, parse_special=True, with_pieces=False) or not all(message['content'] in item['applyTemplateResponse']['prompt'] for message in row['request']['messages']):
            raise ValueError('Missing complete native prompt content')
        if key[0] in ['baseline','balanced']:
            reference = references[key]
            expected_reference = dict(path='benchmarks/runs/rewrite-selector-screen-v1/run/results.jsonl', rowSha256=rowhash(reference), actualPromptTokens=reference['response']['usage']['prompt_tokens'])
            if item['controlReference'] != expected_reference or len(tokens) != expected_reference['actualPromptTokens']:
                raise ValueError('Native tokenization differs from actual control usage')
        elif item['controlReference'] is not None:
            raise ValueError('Invented prior model output for new examples')
        seen.add(key)
    if seen != set(current) or len(budget) != 404 or status['maximumPromptTokens'] != max(i['promptTokens'] for i in budget):
        raise ValueError('Incomplete native budget coverage')
    if 'offloaded 25/25 layers to GPU' not in (native/'token-budget-server.log').read_text():
        raise ValueError('Missing actual native budget GPU placement')
    return rows, labels


core.checked = checked


def collect():
    summary, rows, labels = base_collect()
    controls = read(RUN/'control-parity.json')['records']
    pairs = read(RUN/'demo-order-comparison.json')['records']
    summary.update(nativeBudget=dict(templateRequests=404,tokenizationRequests=404,modelGenerations=0,maximumPromptTokens=read(RUN.parent/'preflight/status.json')['maximumPromptTokens']),controlPairs=len(controls), controlRawParity=sum(p['sameRawChoice'] for p in controls),
                   controlEditedParity=sum(p['sameEditedSource'] for p in controls),
                   demoOrderPairs=len(pairs), changedChoices=sum(p['changedChoice'] for p in pairs),
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
    lines = ['# Existing demonstration pair ordering', '', 'Every actual choice personally reviewed. Full decoded inputs, rules, all ten exact demonstration pairs, final-source/context and request settings retained; only pair order changes. No new final bodies or release acceptance.', '']
    for row in rows:
        lines += [f"## {row['arm']} / {row['id']} / {row['scene']}", '', '```json', json.dumps({key: row[key] for key in ['source', 'context', 'request', 'selection', 'editedSource', 'edits']}, ensure_ascii=False, indent=2), '```', '', '```text', row['response']['choices'][0]['message']['content'], '```', '', labels[(row['variant'], row['id'])]['note'], '']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print(json.dumps(summary, ensure_ascii=False))


if __name__ == '__main__':
    main()
