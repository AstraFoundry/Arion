"""Model-free immutable-archive validation and exact report regeneration."""
from pathlib import Path
from study import ROOT, read, sha, report, schedule_report
study=ROOT/'benchmarks/reproduction/llm-ime-alignment-v1'
manifest=read(study/'archive.json')
for path,digest in manifest['files'].items():
    if sha(ROOT/path)!=digest:raise ValueError('Archive hash mismatch: '+path)
run=ROOT/'benchmarks/runs/llm-ime-alignment-v1';result=ROOT/'benchmarks/results/llm-ime-alignment-v1'
for name in ['pilot','gated']:
    if report(run/name)!=read(result/(name+'-summary.json')):raise ValueError('Report replay mismatch: '+name)
    for path,digest in read(run/name/'environment.json')['sources'].items():
        if sha(study/'historical-source'/name/path)!=digest:raise ValueError('Historical source mismatch')
for raw,out in [('schedule-observations.jsonl','schedule-preliminary-summary.json'),('schedule-observations-v2.jsonl','schedule-summary.json')]:
    if schedule_report(run/'schedule.json',run/raw)!=read(result/out):raise ValueError('Schedule replay mismatch')
print(f"Verified {len(manifest['files'])} frozen files and four exact report replays")
