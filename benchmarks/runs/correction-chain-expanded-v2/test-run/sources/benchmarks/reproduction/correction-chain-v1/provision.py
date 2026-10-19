"""Provision pinned public correction weights; preserve metadata and failed attempts."""
import concurrent.futures
import hashlib
import json
import pathlib
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[3]
DEST = ROOT / '.workspace/correction-chain/models'
MODELS = {
    'bart4csc': ('shibing624/bart4csc-base-chinese', '6bce2c5d39d52f542a720bfc2c612623401a2380'),
    'flanec': ('morenolq/flanec-base-cd', 'e3c8a1ea702dfb2e47ce77733a31123ea18dbe56'),
}


def acquire(name, repository, revision):
    directory = DEST / name
    directory.mkdir(parents=True, exist_ok=False)
    url = f'https://huggingface.co/api/models/{repository}/revision/{revision}?blobs=true'
    with urllib.request.urlopen(url, timeout=30) as response:
        metadata = json.load(response)
    (directory / 'upstream.json').write_text(json.dumps(metadata, indent=2))
    selected = [f for f in metadata['siblings'] if f['rfilename'] in {
        'config.json', 'generation_config.json', 'model.safetensors', 'tokenizer.json',
        'tokenizer_config.json', 'special_tokens_map.json', 'vocab.txt', 'README.md'}]
    records = []
    for item in selected:
        path = directory / item['rfilename']
        url = f'https://huggingface.co/{repository}/resolve/{revision}/{path.name}'
        with urllib.request.urlopen(url, timeout=120) as source, path.open('xb') as target:
            while block := source.read(1024 * 1024):
                target.write(block)
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        if item.get('lfs'):
            assert digest == item['lfs']['sha256'], 'LFS digest mismatch'
        else:
            data = path.read_bytes()
            blob = hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()
            assert blob == item['blobId'], 'Git blob mismatch'
        records.append(dict(file=path.name, sha256=digest, bytes=path.stat().st_size, url=url))
        print(name, path.name, path.stat().st_size, flush=True)
    manifest = dict(repository=repository, revision=revision, card=metadata.get('cardData'), files=records)
    (directory / 'manifest.json').write_text(json.dumps(manifest, indent=2))


if __name__ == '__main__':
    with concurrent.futures.ThreadPoolExecutor(2) as pool:
        jobs = [pool.submit(acquire, name, *spec) for name, spec in MODELS.items()]
        for job in jobs:
            job.result()
