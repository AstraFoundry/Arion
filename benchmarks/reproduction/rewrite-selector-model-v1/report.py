"""Validate failed execution and every completed diagnostic response separately."""
import hashlib
import json
import pathlib
import argparse

ROOT = pathlib.Path(__file__).resolve().parents[3]
RUN = ROOT/'benchmarks/runs/rewrite-selector-model-v1'


def read(path):
    return json.loads(path.read_text())


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rowhash(value):
    return hashlib.sha256(json.dumps(value, ensure_ascii=False, sort_keys=True).encode()).hexdigest()


def checked(directory=RUN):
    failed = read(directory/'failed-schema/manifest.json')
    if failed['status'] != 'failed' or failed['completedGenerations'] != 0 or (directory/'failed-schema/results.jsonl').read_bytes():
        raise ValueError('Failed Granite run reclassified as generated output')
    if read(directory/'terminal-initial.json')['exitCode'] != 1:
        raise ValueError('Changed initial native failure')
    error = 'Unexpected empty grammar stack after accepting piece: 8 (23)'
    if error not in (directory/'failed-schema/selector-model-v1-baseline-work-server.log').read_text():
        raise ValueError('Missing raw native failure')
    failure = read(directory/'failure.json')
    if failure['rawRequestReference']['id'] != 'history-0095':
        raise ValueError('Wrong first attempted source identity')
    if sha(directory/'freeze-initial.json') != failed['freezeSha256'] or failed['model'] != read(directory/'freeze-initial.json')['model']:
        raise ValueError('Changed failed model/freeze identity')
    results_path = directory/'diagnostic-responses/results.jsonl'
    rows = list(map(json.loads, results_path.read_text().splitlines()))
    manifest = read(directory/'diagnostic-responses/manifest.json')
    review = read(directory/'diagnostic-responses/review.json')
    if manifest['resultsSha256'] != sha(results_path) or review['resultsSha256'] != sha(results_path) or manifest['reviewSha256'] != sha(directory/'diagnostic-responses/review.json'):
        raise ValueError('Changed diagnostic projection or review')
    raw = {}
    errors = [dict(id='initial-full-screen', request=read(directory/'attempted-request.json'), error=failed['error'], nativeLog='benchmarks/runs/rewrite-selector-model-v1/failed-schema/selector-model-v1-baseline-work-server.log')]
    for group in ['diagnostic', 'precision']:
        path = directory/group/'results.json'
        key = 'benchmarks/runs/rewrite-selector-model-v1/'+group+'/results.json'
        if manifest['sourceFiles'][key] != sha(path):
            raise ValueError('Changed native diagnostic source')
        for index, item in enumerate(read(path)):
            identity = (key, index)
            if item['response'] is None:
                errors.append(dict(id=group+'-'+str(index), request=item['request'], error=item['httpError'], nativeCommand=item.get('nativeCommand', item.get('command'))))
            else:
                raw[identity] = item
    labels = {(j['variant'], j['id']): j for j in review['reviews']}
    if manifest['status'] != 'complete' or manifest['generations'] != 7 or len(rows) != 7 or len(labels) != 7 or len(raw) != 7 or len(errors) != 6:
        raise ValueError('Missing actual response or HTTP failure')
    seen = set()
    for row in rows:
        ref = row['rawReference']
        key = (ref['path'], ref['index'])
        actual = raw[key]
        if key in seen or ref['rowSha256'] != rowhash(actual):
            raise ValueError('Duplicate or changed diagnostic identity')
        for name in ['request', 'response', 'generationMs', 'residentRssBytes']:
            if row[name] != actual[name]:
                raise ValueError('Projected response differs from actual bytes')
        last_input = row['request']['messages'][-1]['content']
        if actual['mode'] == 'simple-schema':
            expected_source, expected_context = last_input, {}
        else:
            payload = json.loads(last_input)
            expected_source, expected_context = payload['source'], payload['context']
        if row['source'] != expected_source or row['context'] != expected_context:
            raise ValueError('Diagnostic source/context differs from actual input')
        if row['stage'] != 'runtime-probe' or row['actualNativeCommand'] != actual.get('nativeCommand', actual.get('command')):
            raise ValueError('Diagnostic reclassified as body or changed command')
        label = labels[(row['variant'], row['id'])]
        expected = {'e0': 'keep' if actual['mode'] == 'simple-schema' else 'replace'}
        try:
            conforms = json.loads(row['response']['choices'][0]['message']['content']) == expected
        except json.JSONDecodeError:
            conforms = False
        if row['expectedDiagnosticChoice'] != expected or label['rowSha256'] != rowhash(row) or label['probeConforms'] != conforms or label['faithful'] is not None or label['sceneUseful'] is not None:
            raise ValueError('Changed diagnostic task or invented fidelity/style judgment')
        seen.add(key)
    return dict(completedResponses=7, diagnosticAttempts=12, diagnosticHttpErrors=5, initialFullScreenHttpErrors=1,
                initialPlannedGenerations=202, initialCompletedGenerations=0,
                conformingDiagnosticTasks=sum(j['probeConforms'] for j in labels.values()), errors=errors,
                finding='GPU layers0 returns valid diagnostic JSON; Metal returns repeated8 without schema. f32KV/flash-off do not recover output. No NaN measurement or exact kernel cause established.'), rows, labels


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--out', type=pathlib.Path, required=True)
    args = parser.parse_args()
    summary, rows, labels = checked()
    args.out.mkdir(parents=True, exist_ok=False)
    (args.out/'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2)+'\n')
    lines = ['# Granite native-runtime diagnosis', '', 'Zero completed responses from the initial full-screen attempt. All seven subsequent completed diagnostic responses reviewed; six total HTTP errors retained.', '']
    for row in rows:
        lines += [f"## {row['id']}", '', '```json', json.dumps(row, ensure_ascii=False, indent=2), '```', '', labels[(row['variant'], row['id'])]['note'], '']
    lines += ['## HTTP errors', '', '```json', json.dumps(summary['errors'], ensure_ascii=False, indent=2), '```']
    (args.out/'review.md').write_text('\n'.join(lines)+'\n')
    print('Replayed seven diagnostic responses and six HTTP errors')


if __name__ == '__main__':
    main()
