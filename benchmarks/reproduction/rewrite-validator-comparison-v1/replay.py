import argparse
import json
import pathlib
import subprocess

parser = argparse.ArgumentParser(description='Replay authored production-validator probes.')
parser.add_argument('--validator', type=pathlib.Path, required=True)
args = parser.parse_args()
root = next(p for p in pathlib.Path(__file__).resolve().parents if (p/'models/catalog.json').exists())
cases = json.loads((pathlib.Path(__file__).parent/'probes.json').read_text())
recorded = json.loads((root/'benchmarks/runs/rewrite-validator-comparison-v1/results.json').read_text())
expected = {r['case']['id']: r['validation']['verdict'] for r in recorded['rows']}
for case in cases:
    request = dict(operation='validate', mode='product', text=case['text'], candidate=case['candidate'])
    result = json.loads(subprocess.check_output([str(args.validator.resolve())], input=json.dumps(request), text=True))
    assert result['verdict'] == expected[case['id']], (case['id'], result)
    print(case['id'], result['verdict'])
