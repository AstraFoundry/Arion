"""Create an immutable manifest after an experiment archive is complete."""
from study import ROOT, write, sha
roots=[ROOT/'benchmarks'/kind/'llm-ime-alignment-v1' for kind in ['reproduction','runs','results']]
out=roots[0]/'archive.json'
if out.exists():raise FileExistsError(out)
files={str(p.relative_to(ROOT)):sha(p) for root in roots for p in sorted(root.rglob('*')) if p.is_file() and '__pycache__' not in p.parts and p!=out}
write(out,{'version':1,'experiment':'llm-ime-alignment-v1','scope':'Synthetic fixtures, actual observations, source snapshots and exact report replay; no weights or credentials.','files':files})
print('Frozen',len(files),'files')
