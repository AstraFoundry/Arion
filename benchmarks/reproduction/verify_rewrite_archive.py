"""Verify an explicit research archive without loading model weights or native workers."""
import hashlib
import json
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[2]


def verify(study):
    if pathlib.Path(study).name != study:
        raise ValueError('Archive study must be a single directory name')
    archive = ROOT/'benchmarks/runs'/study/'archive.json'
    files = json.loads(archive.read_text())['files']
    for relative, expected in files.items():
        path = pathlib.Path(relative)
        if path.is_absolute() or '..' in path.parts or hashlib.sha256((ROOT/path).read_bytes()).hexdigest() != expected:
            raise ValueError('Changed research archive: '+relative)
    print('Verified', len(files), study, 'evidence files')
