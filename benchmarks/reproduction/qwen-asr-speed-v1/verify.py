"""Verify the completed diagnostic archive without models or inference runtimes."""
import hashlib,json,pathlib
root=pathlib.Path(__file__).resolve().parents[3]
manifest=json.loads((root/'benchmarks/results/qwen-asr-speed-v1/archive.json').read_text())
assert manifest['status']=='complete' and manifest['files']
for entry in manifest['files']+manifest['sharedInputs']:
 p=root/entry['path']
 assert not p.is_symlink() and p.resolve().is_relative_to(root),entry['path']
 data=p.read_bytes()
 assert len(data)==entry['sizeBytes'] and hashlib.sha256(data).hexdigest()==entry['sha256'],entry['path']
print('Verified',len(manifest['files']),'archive files and',len(manifest['sharedInputs']),'shared inputs; no weights required.')
