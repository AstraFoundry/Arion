"""Capture the native prompt protocol smoke with immutable request/output bytes."""
import argparse
import base64
import gzip
import hashlib
import json
import os
from pathlib import Path
import platform
import queue
import shutil
import subprocess
import threading
import wave

ROOT = Path(__file__).resolve().parents[4]


def sha(path):
    digest = hashlib.sha256()
    with path.open('rb') as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b''):
            digest.update(block)
    return digest.hexdigest()


def main():
    parser = argparse.ArgumentParser()
    for name in ['worker', 'model', 'out']:
        parser.add_argument('--' + name, required=True)
    args = parser.parse_args()
    out = Path(args.out).resolve()
    out.mkdir(parents=True, exist_ok=False)
    worker, model = Path(args.worker).resolve(), Path(args.model).resolve()
    fixture = ROOT / 'benchmarks/corpora/asr-window-v1/ocr-work-term.wav'
    corpus = ROOT / 'benchmarks/corpora/asr-window-v1/corpus.json'
    description = json.loads(corpus.read_text())
    sample = next(s for s in description['samples'] if s['id'] == 'ocr-work-term')
    assert sha(fixture) == sample['audio']['sha256']
    artifact = next(a for a in json.loads((ROOT / 'models/catalog.json').read_text())['artifacts'] if a['id'] == 'qwen3-asr-1.7b-mlx-8bit')
    model_files = []
    for name, expected in artifact['files'].items():
        file = model / name
        digest = sha(file)
        assert digest == expected['sha256'], name
        model_files.append({'name': name, 'sha256': digest, 'bytes': file.stat().st_size})
    provenance = {'schema': 1, 'status': 'running', 'kind': 'native-protocol-smoke', 'platform': platform.platform(), 'machine': platform.machine(), 'command': [str(worker), 'worker', str(model)], 'worker': {'sha256': sha(worker), 'bytes': worker.stat().st_size}, 'modelFiles': model_files, 'fixture': {'path': str(fixture.relative_to(ROOT)), 'sha256': sha(fixture), 'license': description['license'], 'consent': description['consent'], 'sample': sample}, 'generation': {'temperature': 0, 'seed': 11, 'maxTokens': 128}, 'contextLimit': {'loadedConfigPath': 'thinker_config.text_config.max_position_embeddings', 'loadedValue': json.loads((model / 'config.json').read_text())['thinker_config']['text_config']['max_position_embeddings'], 'check': 'actual tokenized complete prompt including audio tokens + generation budget <= loaded model maxPositionEmbeddings'}, 'overflowPrompt': {'construction': "'x ' * 70000", 'utf8Bytes': 140000, 'sha256': hashlib.sha256(('x ' * 70000).encode()).hexdigest()}, 'limitations': ['Protocol/runtime smoke only; no accuracy or latency acceptance claim.', 'Existing synthetic development fixture; no held-out speech or Windows host evidence.', 'Weights and executable remain external and hash-identified; no downloads.']}
    for file in [Path(__file__), Path(__file__).with_name('Worker.swift'), Path(__file__).with_name('Package.swift'), Path(__file__).with_name('Package.resolved')]:
        shutil.copyfile(file, out / file.name)
    (out / 'provenance.json').write_text(json.dumps(provenance, ensure_ascii=False, indent=2) + '\n')
    messages = queue.Queue()
    raw = (out / 'worker.stdout.jsonl').open('xb')
    stderr = (out / 'worker.stderr.log').open('xb')
    requests = gzip.open(out / 'requests.jsonl.gz', 'xb')
    process = subprocess.Popen(provenance['command'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=stderr, env={**os.environ, 'HF_HUB_OFFLINE': '1'})

    def read():
        for line in process.stdout:
            raw.write(line)
            raw.flush()
            messages.put(json.loads(line))
        messages.put(None)

    reader = threading.Thread(target=read, daemon=True)
    reader.start()

    def receive():
        response = messages.get(timeout=180)
        assert response is not None, ('worker exited', process.poll())
        return response

    def send(message):
        line = (json.dumps(message, ensure_ascii=False) + '\n').encode()
        requests.write(line)
        requests.flush()
        process.stdin.write(line)
        process.stdin.flush()

    try:
        ready = receive()
        assert ready['type'] == 'ready' and ready['protocol'] == 3 and 'prompt' in ready['capabilities'], ready
        with wave.open(str(fixture), 'rb') as audio:
            assert (audio.getnchannels(), audio.getsampwidth(), audio.getframerate()) == (1, 2, 16000)
            pcm = base64.b64encode(audio.readframes(audio.getnframes())).decode()
        checks = []
        for identity, prompt in [('contextual', 'MLX Swift'), ('empty', ''), ('overflow', 'x ' * 70000)]:
            send({'type': 'chunk', 'id': identity, 'seq': 0, 'pcm': pcm})
            send({'type': 'transcribe', 'id': identity, 'sample_rate': 16000, 'prompt': prompt})
            response = receive()
            assert response.get('id') == identity, response
            assert response['type'] == ('error' if identity == 'overflow' else 'end'), response
            if identity != 'overflow':
                assert response.get('stop') == 'complete', response
            checks.append({'id': identity, 'passed': True})
        provenance.update(status='complete', checks=checks)
    except BaseException as error:
        provenance.update(status='failed', failure=repr(error))
        raise
    finally:
        process.stdin.close()
        try:
            process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()
        reader.join(timeout=10)
        requests.close()
        raw.close()
        stderr.close()
        provenance['workerExitCode'] = process.returncode
        (out / 'provenance.json').write_text(json.dumps(provenance, ensure_ascii=False, indent=2) + '\n')
        identities = {p.name: {'sha256': sha(p), 'bytes': p.stat().st_size} for p in sorted(out.iterdir()) if p.is_file()}
        (out / 'sha256.json').write_text(json.dumps(identities, indent=2) + '\n')
    print(json.dumps({'status': provenance['status'], 'checks': provenance['checks'], 'out': str(out)}))


if __name__ == '__main__':
    main()
