"""Explicit, digest-checked development provisioning. Never runs during ordinary checks."""
import argparse
import hashlib
import importlib.util
import json
import pathlib
import subprocess
import tarfile

ROOT = pathlib.Path(__file__).resolve().parents[3]


def checksum(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        while block := f.read(1024 * 1024):
            h.update(block)
    return h.hexdigest()


def fetch(asset, destination):
    destination.parent.mkdir(parents=True, exist_ok=True)
    candidate = destination if destination.exists() else destination.with_name(destination.name + '.partial')
    if not destination.exists():
        subprocess.run(['curl', '-sS', '--fail', '-L', '--retry', '3', '-C', '-', '-o', str(candidate), asset['url']], check=True)
    if candidate.stat().st_size != asset['sizeBytes']:
        raise ValueError('Downloaded size mismatch: ' + asset['path'])
    digest = checksum(candidate)
    if asset.get('sha256') and digest != asset['sha256']:
        raise ValueError('Downloaded digest mismatch: ' + asset['path'])
    if not asset.get('sha256') and asset.get('gitBlobSha1'):
        content = candidate.read_bytes()
        blob = hashlib.sha1(f'blob {len(content)}\0'.encode() + content).hexdigest()
        if blob != asset['gitBlobSha1']:
            raise ValueError('Git blob digest mismatch: ' + asset['path'])
    if candidate != destination:
        candidate.rename(destination)
    return dict(path=str(destination.resolve()), sha256=digest, sizeBytes=destination.stat().st_size, source=asset['url'])


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--models', required=True)
    parser.add_argument('--ids', required=True)
    parser.add_argument('--asr', action='store_true')
    args = parser.parse_args()
    out = pathlib.Path(args.models).resolve()
    if not out.is_relative_to(ROOT / '.workspace'):
        raise ValueError('Weights and traces must stay in the private .workspace directory')
    catalog = json.loads((ROOT / 'benchmarks/catalogs/qwen-diagnostic-models.json').read_text())
    ids = args.ids.split(',')
    index = {m['id']: m for m in catalog['models']}
    if not ids or any(i not in index for i in ids):
        raise ValueError('Unknown model selection')
    artifacts = []
    for model_id in ids:
        for asset in index[model_id]['assets']:
            artifact = fetch(asset, out / model_id / asset['path'])
            artifact.update(modelId=model_id, license=index[model_id]['license'], repository=index[model_id]['repository'], revision=index[model_id]['revision'])
            artifacts.append(artifact)
    for name in ['onnxruntime_genai', 'onnxruntime']:
        package = importlib.util.find_spec(name)
        if package is None:
            raise ValueError('Provision the pinned diagnostic environment first')
        for file in pathlib.Path(package.submodule_search_locations[0]).rglob('*'):
            if file.is_file() and (file.suffix in ['.so', '.dylib'] or file.name in ['__init__.py', '_dll_directory.py']):
                artifacts.append(dict(path=str(file.resolve()), sha256=checksum(file), sizeBytes=file.stat().st_size, runtime=name, license='MIT'))
    if args.asr:
        asset = dict(catalog['qwenAsrArchive'], path='qwen3-asr.tar.bz2')
        archive = out / asset['path']
        fetch(asset, archive)
        with tarfile.open(archive) as source:
            source.extractall(out, filter='data')
    (out / 'artifacts.json').write_text(json.dumps(artifacts, indent=2) + '\n')
    print('Provisioned and verified selected checkpoints and native runtime files')


if __name__ == '__main__':
    main()
