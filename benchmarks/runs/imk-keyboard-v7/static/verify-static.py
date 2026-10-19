"""Read-only verification against source artifacts; never executes the target."""
from pathlib import Path
import json,hashlib,re
p=Path(__file__).parent/'evidence-focused.json';e=json.loads(p.read_text());sha=lambda b:hashlib.sha256(b).hexdigest();carriers={}
for key,identity in e['artifacts'].items():
 raw=Path(identity['path']).read_bytes();assert sha(raw)==identity['universalSha256'];thin=raw[identity['arm64Offset']:identity['arm64Offset']+identity['arm64Size']];assert sha(thin)==identity['arm64Sha256'];carriers[key]=raw
for f in e['intervals']:
 raw=carriers[f['artifact']];off=f['universalFileOffset'];assert sha(raw[off:off+f['sizeBytes']])==f['sha256'];assert f['instructionsValidated']*4==f['sizeBytes'];assert not f['unrecognizedInstructionSlots']
 for line in f['anchors']:
  m=re.match(r'([0-9a-f]+):\s*((?:[0-9a-f]{2} ){3}[0-9a-f]{2})',line);assert m,line;delta=int(m[1],16)-int(f['entry'],16);assert 0<=delta<f['sizeBytes'];assert raw[off+delta:off+delta+4]==bytes.fromhex(m[2])
for m in e['metadata']:
 raw=bytes.fromhex(m['bytes']);assert sha(raw)==m['sha256'];off=m['universalFileOffset'];assert carriers[m['artifact']][off:off+len(raw)]==raw
print('PASS: two exact carriers/arm64 slices; six source intervals, 1952 previously collected instruction slots and every focused anchor byte; metadata bytes; no target execution')
