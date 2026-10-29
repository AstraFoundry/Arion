"""Create a new archive inventory after the development evidence is complete."""
import hashlib
import json
import pathlib
import shutil

root = pathlib.Path(__file__).resolve().parents[3]
manifest = root / 'benchmarks/results/asr-hotwords-v1/archive.json'
if manifest.exists():
    raise FileExistsError('Archive already exists; use a new study/version rather than overwriting retained evidence.')
paths = set()
for relative in ['tools/evaluation/archive/manifest.mjs', 'tools/evaluation/src/metrics.mjs', 'scripts/asr-hotword-bench-command.mjs']:
    destination = root / 'benchmarks/reproduction/asr-hotwords-v1/source-tooling' / relative
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(root / relative, destination)

for base in ['benchmarks/corpora/asr-hotwords-v1', 'benchmarks/reproduction/asr-hotwords-v1', 'benchmarks/runs/asr-hotwords-v1', 'benchmarks/results/asr-hotwords-v1']:
    paths.update(p for p in (root / base).rglob('*') if p.is_file() and p != manifest and '__pycache__' not in p.parts)
for name in ['requests.json', 'sensitive-isolation-requests.json']:
    for row in json.loads((root / 'benchmarks/corpora/asr-hotwords-v1' / name).read_text()):
        paths.add(root / row['audioPath'])
evaluation = json.loads((root / 'benchmarks/corpora/asr-hotwords-v1/evaluation.json').read_text())
for item in evaluation['sourceProvenance']:
    paths.add(root / item['path'])
paths.add(root / evaluation['memorySource']['path'])
for relative in ['benchmarks/corpora/asr-window-v1/corpus.json', 'benchmarks/corpora/asr-window-v1/ocr-work-term.wav']:
    paths.add(root / relative)
entries = []
for path in sorted(paths):
    data = path.read_bytes()
    relative = str(path.relative_to(root))
    entries.append({'source': relative, 'destination': relative, 'sha256': hashlib.sha256(data).hexdigest(), 'bytes': len(data)})
result = {'version': 1, 'kind': 'arion-benchmark-archive', 'status': 'complete', 'historicalRoot': str(root), 'entries': entries, 'scope': 'Paired development production ASR hotwords, separately scoped native protocol smoke, exact source snapshots, failed attempts, shared original fixtures and replay tooling. Weights and executable runtimes are identified in metadata but excluded.'}
with manifest.open('x') as stream:
    json.dump(result, stream, indent=2)
    stream.write('\n')
print(len(entries), 'archive files')
