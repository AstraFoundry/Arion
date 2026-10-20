#!/usr/bin/env python3
"""Verify frozen study bytes without models, native runtimes or inference."""
import hashlib
import json
from pathlib import Path


def verify(directory):
    directory = Path(directory).resolve()
    manifest = json.loads((directory / 'study-manifest.json').read_bytes())
    files = manifest['files']
    for relative, expected in files.items():
        path = (directory / relative).resolve()
        if not path.is_relative_to(directory) or not path.is_file():
            raise ValueError(f'missing or unsafe study path: {relative}')
        if hashlib.sha256(path.read_bytes()).hexdigest() != expected:
            raise ValueError(f'study hash mismatch: {relative}')
    actual = {str(p.relative_to(directory)) for p in directory.rglob('*')
              if p.is_file() and '__pycache__' not in p.parts and p != directory / 'study-manifest.json'}
    if actual != set(files):
        raise ValueError(f'study inventory mismatch; added={sorted(actual-set(files))}, missing={sorted(set(files)-actual)}')
    print(f'Verified {len(files)} study files; no inference executed.')


if __name__ == '__main__':
    verify(Path(__file__).parent)
