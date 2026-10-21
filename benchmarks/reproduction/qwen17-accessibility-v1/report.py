"""Verify paired native window reads and hash-bound assistant output reviews."""
import argparse
import hashlib
import json
import math
import pathlib
import statistics


def sha(path):
    digest = hashlib.sha256()
    with path.open('rb') as source:
        for block in iter(lambda: source.read(1024 * 1024), b''):
            digest.update(block)
    return digest.hexdigest()


def rows(path):
    return [json.loads(line) for line in path.read_text().splitlines()]


def quantile(values, probability):
    ordered = sorted(values)
    index = (len(ordered) - 1) * probability
    low, high = math.floor(index), math.ceil(index)
    return ordered[low] + (ordered[high] - ordered[low]) * (index - low)


def normalized(text):
    return ''.join(text.split())


def report(directory):
    reviews = json.loads((directory / 'review.json').read_text())
    for name, digest in reviews['files'].items():
        if sha(directory / name) != digest:
            raise ValueError('Changed reviewed input or output: ' + name)
    cases = json.loads((directory / 'window-cases.json').read_text())
    observations = rows(directory / 'window-observations.jsonl')
    expected = {(case['id'], repeat) for case in cases for repeat in range(3)}
    actual = [(r['id'], r['repeat']) for r in observations]
    if len(actual) != len(expected) or set(actual) != expected:
        raise ValueError('Incomplete or duplicate window observations')
    if any(r['axStatus'] != 0 or r['ocrStatus'] != 0 for r in observations):
        raise ValueError('Failed window observations')
    if any('CANARY_NEVER_READ' in r['axText'] for r in observations):
        raise ValueError('Protected control leaked into context')
    output = {'status': 'complete', 'windowCases': len(cases), 'pairedWindowReads': len(observations),
              'reviewer': reviews['reviewer'], 'canaryLeaks': 0, 'sources': {}}
    outputs = {}
    for kind in ('ax', 'ocr'):
        filename = directory / (kind + '-rewrite.jsonl')
        if reviews['files'][filename.name] != sha(filename):
            raise ValueError('Changed reviewed output')
        data = rows(filename)
        if {r['caseId'] for r in data} != {c['id'] for c in cases} or len(data) != len(cases):
            raise ValueError('Rewrite coverage mismatch')
        outputs[kind] = {r['caseId']: r for r in data}
        if any(r['truncated'] or r['verdict'] != 'accepted' for r in data):
            raise ValueError('Incomplete rewrite delivery')
        for r in data:
            judged = reviews['cases'][r['caseId']][kind]
            digest = hashlib.sha256(r['text'].encode()).hexdigest()
            if judged['textSha256'] != digest or not isinstance(judged['pass'], bool):
                raise ValueError('Unbound or missing output judgment')
        durations = [r['axMs'] if kind == 'ax' else r['captureMs'] + r['ocrMs'] for r in observations]
        truth = {c['id']: c['text'] for c in cases}
        coverage = sum(normalized(truth[r['id']]) in normalized(r[kind + 'Text']) for r in observations)
        output['sources'][kind] = {'readMsP50': statistics.median(durations), 'readMsP95': quantile(durations, .95),
                                   'completeFixtureTextPresent': coverage,
                                   'reviewPass': sum(reviews['cases'][r['caseId']][kind]['pass'] for r in data)}
    controls = json.loads((directory / 'visibility-final.json').read_text())
    flags = ('visibleRangeAndCanariesPass', 'staleRejected', 'limitRejectedWithoutPartialText')
    if not all(controls.get(key) is True for key in flags):
        raise ValueError('Native visibility or stale/limit control failed')
    output['nativeControls'] = {key: controls[key] for key in flags}
    output['equalFinalOutputs'] = sum(outputs['ax'][c['id']]['text'] == outputs['ocr'][c['id']]['text'] for c in cases)
    return output


def integration_report(directory, window):
    review = json.loads((directory / 'review.json').read_text())
    for name, digest in review['files'].items():
        if sha(directory / name) != digest:
            raise ValueError('Changed integration input/output: ' + name)
    window_rows = {r['id']: r for r in rows(window / 'window-observations.jsonl') if r['repeat'] == 0}
    native = [r for r in rows(directory / 'integration-asr.jsonl') if 'response' in r and not r['warmup']]
    arms = {'old': 'xasr-160-xasr-final', 'new': 'xasr-160-qwen3asr-17-final'}
    if len(native) != 24 or len({(r['variant'], r['id']) for r in native}) != 24:
        raise ValueError('Incomplete or duplicate integration ASR')
    for kind, variant in arms.items():
        source = {r['id']: r['response']['finalText'] for r in native if r['variant'] == variant}
        corpus = json.loads((directory / ('integration-' + kind + '-corpus.json')).read_text())['samples']
        data = rows(directory / ('integration-' + kind + '-rewrite.jsonl'))
        if set(source) != set(window_rows) or len(corpus) != 12 or len(data) != 12 or {r['caseId'] for r in data} != set(source):
            raise ValueError('Integration coverage mismatch')
        for case in corpus:
            if case['sourceText'] != source[case['id']] or case['contextFixture']['screenTruth'] != window_rows[case['id']]['ocrText' if kind == 'old' else 'axText']:
                raise ValueError('Integration did not use actual ASR/window text')
        for r in data:
            judged = review['cases'][r['caseId']][kind]
            if judged['textSha256'] != hashlib.sha256(r['text'].encode()).hexdigest() or not isinstance(judged['pass'], bool):
                raise ValueError('Unbound integration review')
            if r['truncated'] or r['verdict'] != 'accepted':
                raise ValueError('Incomplete integration delivery')
    cases = review['cases']
    return {'cases': len(cases), 'reviewer': review['reviewer'],
            'oldPass': sum(c['old']['pass'] for c in cases.values()),
            'newPass': sum(c['new']['pass'] for c in cases.values()),
            'gained': [k for k, c in cases.items() if c['new']['pass'] and not c['old']['pass']],
            'regressed': [k for k, c in cases.items() if c['old']['pass'] and not c['new']['pass']]}


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--run', type=pathlib.Path, required=True)
    parser.add_argument('--out', type=pathlib.Path, required=True)
    parser.add_argument('--integration', type=pathlib.Path)
    args = parser.parse_args()
    result = report(args.run)
    if args.integration:
        result['integration'] = integration_report(args.integration, args.run)
    with args.out.open('x') as target:
        json.dump(result, target, ensure_ascii=False, indent=2)
        target.write('\n')
