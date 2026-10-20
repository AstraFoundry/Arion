"""Archive completed/rejected development evidence without model execution."""
import hashlib,json,shutil
from pathlib import Path
root=Path('/Users/steven/code/Arion');scratch=root/'.workspace/rewrite-stage-preservation';out=root/'benchmarks/reproduction/rewrite-stage-preservation-v1/evidence/rejected-v1';out.mkdir(parents=True,exist_ok=False)
files={};mappings={}
sha=lambda b:hashlib.sha256(b).hexdigest()
def copy(source,destination):
 if source.is_dir():
  for f in sorted(source.rglob('*')):
   if f.is_file() and '__pycache__' not in f.parts:copy(f,destination/f.relative_to(source))
  return
 destination.parent.mkdir(parents=True,exist_ok=True);shutil.copyfile(source,destination);files[str(destination.relative_to(out))]=sha(destination.read_bytes());mappings[str(source.resolve())]=str(destination.relative_to(out))
copy(scratch/'matrix-1',out/'matrix-1')
copy(scratch/'fresh-contrast-matrix',out/'fresh-contrast-matrix')
copy(scratch/'dev-baseline',out/'rejected-trials/workflow-initial-baseline')
for name in ['development-full-1','development-full-2','development-selected','discovery-baseline','v2-calibration','v3-preflight']:
 for source in sorted(scratch.glob(name+'.*')):copy(source,out/'rejected-trials'/source.name)
copy(scratch/'discovery-baseline-status.json',out/'rejected-trials/discovery-baseline-status.json')
for name in ['baseline-source','attempt-2-source','selected-source','v1-overlay','v2-overlay','v3-overlay']:
 copy(scratch/name,out/'source-snapshots'/name)
for source in sorted(scratch.glob('run-fresh-contrast*')):copy(source,out/'execution-sources'/source.name)
for name in ['development-report-reviewed','validation-report-reviewed','native-report-reviewed','fresh-contrast-report-reviewed']:
 copy(scratch/name,out/'reports'/name)
for name in ['discovery.json','development.json','controls.json','validation.json','user-case.json','native-pairs.json','fresh-prior-controls.json','protocol.json','stress.json','stress-preflight-original.json','stress-preflight-correction.json','v2-calibration.json','report.py','test_report.py','run-matrix.mjs']:
 source=root/'benchmarks/reproduction/rewrite-stage-preservation-v1'/name
 if source.exists():copy(source,out/'execution-sources'/name)
copy(root/'scripts/rewrite-stage-command.mjs',out/'execution-sources/rewrite-stage-command.mjs')
copy(Path('/tmp/arion-review-validation.py'),out/'execution-sources/serialize-validation-review.py')
copy(Path('/tmp/arion-review-native.py'),out/'execution-sources/serialize-native-review.py')
copy(Path('/tmp/arion-review-fresh.py'),out/'execution-sources/serialize-fresh-review.py')
copy(Path(__file__),out/'execution-sources/archive-rejected.py')
readme='''# Rejected and exploratory rewrite experiments

This immutable evidence snapshot retains completed matrix-1 development, validation, and native text-stage arms; the completed fresh prior contrast matrix; cancelled discovery; fixed-gap formatter failures; prompt-only V2/V3 exploratory trials; source snapshots; exact native completion requests/responses; logs; and hash-bound assistant reviews. V1 and V3 are rejected. Aggregate repair gains do not override newly harmful current-intent changes. V3 ambiguous-prior timeout remains visible as a failed, unassessable phase.

Reports are self-contained under `reports/`. Reviewer judgments came from a separate assistant task that read every recorded stage, not independent human or blind review. Automated substring/count checks are proxies. All inputs are authored development or synthetic TTS/Vision fixtures, not the user's original failure recording. Native input pairs begin from the last recorded ASR draft, not an actual LLM preview event. Separate real Coordinator preview/stop evidence and installed-worker evidence are already archived in this study's `native/` and `installed-v1/` directories and are not duplicated here.

`manifest.json` records the SHA-256 and original-to-archive mapping for every copied file. Original overlay files and execution metadata intentionally retain their original absolute paths; use `pathMappings` to rebase those references for replay. Weights, native runtime environments, packaged worker executables, caches and personal recordings are omitted; per-run identity records retain observed runtime/model hashes. Executable replay requires separately provisioned matching native artifacts. Matrix arms include exact source snapshots and effective configuration. The first exploratory fixed-layout trial predates the available attempt-2 source snapshot; its raw requests/responses are retained, but no claim of exact compiled first-trial source recovery is made. Review serialization scripts record the manually inspected judgments, not automated model quality tests.

Verify this archive from the repository root:

```sh
python3 benchmarks/reproduction/rewrite-stage-preservation-v1/report.py --verify benchmarks/reproduction/rewrite-stage-preservation-v1/evidence/rejected-v1
```

No report here promotes the rejected experiments or replaces the later chosen-source tests.
'''
(out/'README.md').write_text(readme);files['README.md']=sha((out/'README.md').read_bytes())
(out/'manifest.json').write_text(json.dumps({'schema':'rewrite-stage-rejected-evidence-v1','files':files,'pathMappings':mappings,'decisions':{'v1':'rejected','v3':'rejected'},'scope':'Completed/rejected/cancelled experimental evidence; no runtime/model binary archive or quality-pass declaration'},indent=2)+'\n')
print(out);print('files',len(files),'bytes',sum((out/p).stat().st_size for p in files))
