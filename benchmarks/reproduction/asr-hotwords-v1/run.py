"""Run a frozen development request plan through the actual Go production finalizer."""
import argparse
import hashlib
import json
import os
import pathlib
import platform
import shutil
import subprocess
import sys
import time

ROOT = pathlib.Path(__file__).resolve().parents[3]

def sha(path):
    h = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(8 * 1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()

def capture(args):
    result = subprocess.run(args, cwd=ROOT, text=True, capture_output=True)
    stdout = result.stdout
    if args[:2] == ['system_profiler', 'SPHardwareDataType']:
        stdout = '\n'.join(line for line in stdout.splitlines() if not any(key in line for key in ['Serial Number', 'Hardware UUID', 'Provisioning UDID'])) + '\n'
    return {'command': args, 'exitCode': result.returncode, 'stdout': stdout, 'stderr': result.stderr}

def main():
    parser = argparse.ArgumentParser()
    for name in ['worker', 'model', 'driver', 'out']:
        parser.add_argument('--' + name, required=True)
    parser.add_argument('--input', default='benchmarks/corpora/asr-hotwords-v1/requests.json')
    args = parser.parse_args()
    out = pathlib.Path(args.out).resolve()
    out.mkdir(parents=True, exist_ok=False)
    artifacts = []
    def pin(path, role):
        path = pathlib.Path(path).resolve()
        artifacts.append({'role': role, 'path': str(path), 'sha256': sha(path), 'bytes': path.stat().st_size})
    worker, model, driver = [pathlib.Path(v).resolve() for v in [args.worker, args.model, args.driver]]
    catalog = json.loads((ROOT / 'models/catalog.json').read_text())
    artifact = next(a for a in catalog['artifacts'] if a['id'] == 'qwen3-asr-1.7b-mlx-8bit')
    for name, expected in artifact['files'].items():
        path = model / name
        if sha(path) != expected['sha256']:
            raise ValueError('Model digest mismatch: ' + name)
        pin(path, 'catalog-model-file')
    pin(driver, 'production-port-driver')
    for path in sorted(worker.parent.rglob('*')):
        if path.is_file():
            pin(path, 'worker-runtime')
    sources = []
    for base in ['apps/desktop/infrastructure/asr', 'apps/desktop/infrastructure/audio', 'packages/core/domain', 'packages/core/contracts', 'apps/desktop/native/mlx-asr']:
        sources.extend(p for p in (ROOT / base).rglob('*') if p.is_file() and (p.suffix in ['.go', '.swift'] or p.name == 'Package.resolved'))
    sources += [ROOT / p for p in ['go.work', 'apps/desktop/go.mod', 'apps/desktop/go.sum', 'packages/core/go.mod', 'models/catalog.json', 'benchmarks/corpora/asr-hotwords-v1/requests.json', 'benchmarks/corpora/asr-hotwords-v1/evaluation.json', 'benchmarks/reproduction/asr-hotwords-v1/driver.go', 'benchmarks/reproduction/asr-hotwords-v1/run.py']]
    custom_input = ROOT / args.input
    if custom_input not in sources:
        sources.append(custom_input)
    for path in sources:
        destination = out / 'source' / path.relative_to(ROOT)
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(path, destination)
    command = [str(driver), '-model', str(model), '-input', args.input]
    metadata = {'version': 1, 'status': 'running', 'design': 'paired-development-ablation', 'stage': 'production-asr.Offline.Finalize', 'model': artifact, 'artifacts': artifacts, 'command': command, 'timer': 'Go time.Since monotonic; wall compute finalization, excluding model load and WAV read', 'pipeline': {'memory': 'controlled canonical spellings, no aliases', 'rewrite': False, 'replacement': False, 'audioReplay': 'offline complete utterance', 'temperature': 0, 'maxTokens': 128}, 'environment': {'platform': platform.platform(), 'machine': platform.machine(), 'python': sys.version, 'hardware': capture(['system_profiler', 'SPHardwareDataType']), 'power': capture(['pmset', '-g', 'custom']), 'battery': capture(['pmset', '-g', 'batt']), 'go': capture(['go', 'version']), 'revision': capture(['git', 'rev-parse', 'HEAD']), 'dirtyState': capture(['git', 'status', '--short'])}, 'sourceFiles': [{'path': str(p.relative_to(ROOT)), 'sha256': sha(p)} for p in sorted(sources)], 'limitations': ['One development host with ordinary background activity; no isolated power/performance guarantee.', 'Model weights and binary dependencies are identified but remain private, not copied into the research archive.', 'No new model download occurred.']}
    (out / 'run.json').write_text(json.dumps(metadata, indent=2) + '\n')
    env = dict(os.environ, ARION_ASR_WORKER=str(worker))
    start = time.monotonic()
    with (out / 'raw.jsonl').open('x') as stdout, (out / 'stderr.log').open('x') as stderr:
        result = subprocess.run(command, cwd=ROOT, env=env, stdout=stdout, stderr=stderr)
    metadata.update(status='complete' if result.returncode == 0 else 'failed', exitCode=result.returncode, wallElapsedMs=(time.monotonic()-start)*1000, rawSha256=sha(out/'raw.jsonl'), stderrSha256=sha(out/'stderr.log'))
    (out / 'run.json').write_text(json.dumps(metadata, indent=2) + '\n')
    print(json.dumps({'status': metadata['status'], 'out': str(out), 'exitCode': result.returncode}))
    return result.returncode

if __name__ == '__main__':
    sys.exit(main())
