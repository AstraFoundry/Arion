"""Verify retained native smoke bytes without models or a native runtime."""
import gzip
import hashlib
import json
from pathlib import Path

root = Path(__file__).resolve().parents[4]
runs = root / 'benchmarks/runs/asr-hotwords-v1'
for name in ['runtime-smoke-initial-attempts', 'runtime-smoke-v1']:
    directory = runs / name
    for filename, expected in json.loads((directory / 'sha256.json').read_text()).items():
        data = (directory / filename).read_bytes()
        assert len(data) == expected['bytes'], filename
        assert hashlib.sha256(data).hexdigest() == expected['sha256'], filename
directory = runs / 'runtime-smoke-v1'
metadata = json.loads((directory / 'provenance.json').read_text())
assert metadata['status'] == 'complete' and metadata['workerExitCode'] == 0
fixture = root / metadata['fixture']['path']
assert hashlib.sha256(fixture.read_bytes()).hexdigest() == metadata['fixture']['sha256']
requests = [json.loads(line) for line in gzip.decompress((directory / 'requests.jsonl.gz').read_bytes()).splitlines()]
assert len(requests) == 6
transcribes = [request for request in requests if request['type'] == 'transcribe']
assert [request['id'] for request in transcribes] == ['contextual', 'empty', 'overflow']
assert [request['prompt'] for request in transcribes] == ['MLX Swift', '', 'x ' * 70000]
overflow = transcribes[-1]['prompt'].encode()
assert len(overflow) == metadata['overflowPrompt']['utf8Bytes']
assert hashlib.sha256(overflow).hexdigest() == metadata['overflowPrompt']['sha256']
responses = [json.loads(line) for line in (directory / 'worker.stdout.jsonl').read_bytes().splitlines()]
assert len(responses) == 4
assert responses[0]['protocol'] == 3 and 'prompt' in responses[0]['capabilities']
assert [(row['id'], row['type']) for row in responses[1:]] == [('contextual', 'end'), ('empty', 'end'), ('overflow', 'error')]
assert all(row['stop'] == 'complete' for row in responses[1:3])
assert metadata['contextLimit']['loadedValue'] == 65536
for filename in ['Worker.swift', 'Package.swift', 'Package.resolved', 'replay.py']:
    assert (directory / filename).read_bytes() == Path(__file__).with_name(filename).read_bytes()
print('Native prompt smoke archive hashes, exact requests, responses, fixture and source snapshots verified.')
