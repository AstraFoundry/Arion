"""Replay authored text checks and verify archived file bytes without inference."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent


def score(name):
    cases = {case['id']: case for case in json.loads((ROOT / 'cases.json').read_text())}
    rows = [json.loads(line) for line in (ROOT / name / 'observations.jsonl').read_text().splitlines()]
    assert len(rows) == 2 * len(cases)
    seen = set()
    passed = 0
    for row in rows:
        case = cases[row['id']]
        request = row['request']
        key = (row['id'], request['Live'])
        assert key not in seen
        seen.add(key)
        assert request['Text'] == case['text'] and request['Scene'] == case['scene']
        assert not row['error'] and not row['response']['Truncated']
        output = row['response']['Text']
        passed += all(s in output for s in case['contains']) and all(s not in output for s in case['excludes'])
    return {'passed': passed, 'total': len(rows)}


def main():
    for name, expected in [('baseline', 10), ('attempt-1', 20), ('attempt-2', 24), ('accepted', 24)]:
        result = score(name)
        assert result['passed'] == expected, (name, result)
        print(name, result)
    manifest = json.loads((ROOT / 'sha256.json').read_text())
    for name, expected in manifest.items():
        assert hashlib.sha256((ROOT / name).read_bytes()).hexdigest() == expected, name
    print('Archived bytes and text checks verified; inference was not rerun.')


if __name__ == '__main__':
    main()
