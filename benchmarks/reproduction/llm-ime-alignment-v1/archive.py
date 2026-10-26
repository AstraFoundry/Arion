"""Freeze actual observations and the source bytes required for replay."""
import argparse, shutil
from pathlib import Path
from study import ROOT, read, write, sha, report, schedule_report

p=argparse.ArgumentParser();p.add_argument('--workspace',required=True);a=p.parse_args();work=Path(a.workspace)
runs=ROOT/'benchmarks/runs/llm-ime-alignment-v1';results=ROOT/'benchmarks/results/llm-ime-alignment-v1';repro=ROOT/'benchmarks/reproduction/llm-ime-alignment-v1'
runs.mkdir(exist_ok=False)
for name in ['pilot','gated']:
    src=work/name;dest=runs/name;dest.mkdir()
    for n in ['requests.jsonl','provenance.json','environment.json','driver.log']:shutil.copyfile(src/n,dest/n)
    shutil.copytree(src/'inference',dest/'inference')
    write(results/(name+'-summary.json'),report(dest))
for name in ['schedule.json','schedule-observations.jsonl','schedule-test.log','schedule-observations-v2.jsonl','schedule-test-v2.log']:shutil.copyfile(work/name,runs/name)
write(results/'schedule-summary.json',schedule_report(runs/'schedule.json',runs/'schedule-observations-v2.jsonl'))
write(results/'schedule-preliminary-summary.json',schedule_report(runs/'schedule.json',runs/'schedule-observations.jsonl'))
# Historical entry files have independently recorded source hashes.
for name in ['pilot','gated']:
    for path,digest in read(runs/name/'environment.json')['sources'].items():
        source=ROOT/path
        if name=='pilot' and path.endswith('/ranking/ranking.go'):source=work/'recovered/ranking-pilot.go'
        if path.endswith('/arion-ime-rank-eval/main.go'):source=work/f'recovered/driver-{name}.go'
        if sha(source)!=digest:raise ValueError('Historical source hash mismatch: '+path)
        target=repro/'historical-source'/name/path;target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(source,target)
# Current source overlay; unchanged dependencies are pinned by the Git base and modules.
paths=['packages/core/domain/keyboard_ranking.go','packages/core/contracts/keyboard_ranking.go','packages/core/domain/ports/keyboard_ranking.go','packages/core/application/keyboard_ranking.go','packages/core/application/keyboard_ranking_test.go','apps/desktop/cmd/arion-ime-rank-eval/main.go','apps/desktop/cmd/arion-llm/main.go','apps/desktop/infrastructure/llm/protocol/protocol.go','apps/desktop/infrastructure/llm/client/client.go','apps/desktop/infrastructure/llm/gguf/engine.go','apps/desktop/infrastructure/llm/gguf/completion.go','apps/desktop/infrastructure/cloudpolish/completion.go','apps/desktop/infrastructure/cloudpolish/completion_test.go','apps/desktop/infrastructure/cloudpolish/service.go','apps/desktop/go.mod','apps/desktop/go.sum','go.work','scripts/llm-runtime.json']
paths += [str(p.relative_to(ROOT)) for p in (ROOT/'apps/desktop/infrastructure/llm/ranking').glob('*.go')]
for path in paths:
    target=repro/'source'/path;target.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(ROOT/path,target)
write(results/'source-snapshot.json',{'scope':'Current source overlay plus historical entry files with recorded hashes. Historical binary bit-for-bit reproduction is not claimed; current source reruns the validated policy.','files':{path:sha(ROOT/path) for path in paths},'historicalSource':'historical-source/{pilot,gated}/ retains all four entry files hashed in each run environment record.'})
print('Archived runs and replayed reports')
