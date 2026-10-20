"""Archive chosen-source completed stage evidence; no model execution."""
import hashlib,json,shutil
from pathlib import Path
root=Path('/Users/steven/code/Arion');scratch=root/'.workspace/rewrite-stage-preservation';out=root/'benchmarks/reproduction/rewrite-stage-preservation-v1/evidence/final-v5';out.mkdir(parents=True,exist_ok=False)
sha=lambda b:hashlib.sha256(b).hexdigest();files={};mappings={}
def copy(source,target):
 if source.is_dir():
  for f in sorted(source.rglob('*')):
   if f.is_file() and '__pycache__' not in f.parts:copy(f,target/f.relative_to(source))
  return
 target.parent.mkdir(parents=True,exist_ok=True);raw=source.read_bytes();target.write_bytes(raw);assert sha(source.read_bytes())==sha(raw),f'source changed while archiving: {source}';files[str(target.relative_to(out))]=sha(raw);mappings[str(source.resolve())]=str(target.relative_to(out))
for name in ['final-validation-candidate','final-validation-baseline-derived','final-intent-baseline','final-intent-baseline-2','final-development-baseline','final-development-no-prior','final-development-no-screen','final-development-candidate']:
 copy(scratch/name,out/'runs'/name)
for name in ['final-validation-review.json','final-validation-semantic-summary.json','final-development-review.json','final-development-semantic-summary.json']:
 copy(scratch/name,out/'reviews'/name)
for name in ['final-validation-report-reviewed','final-development-report-reviewed']:copy(scratch/name,out/'reports'/name)
for name in ['final-code-review.md','final-code-review-tests.log']:copy(scratch/name,out/'source-review'/name)
for name in ['run-final-followups.py','final-followups-progress.json','final-followups-2-progress.json']:
 if (scratch/name).exists():copy(scratch/name,out/'execution-sources'/name)
study=root/'benchmarks/reproduction/rewrite-stage-preservation-v1'
for name in ['final-validation.json','final-validation-provenance.json','final-intent-controls.json','fresh-prior-controls.json','development.json','report.py','test_report.py']:
 copy(study/name,out/'execution-sources'/name)
copy(root/'scripts/rewrite-stage-command.mjs',out/'execution-sources/rewrite-stage-command.mjs')
copy(Path('/tmp/arion-final-review.py'),out/'execution-sources/serialize-final-validation-review.py')
copy(Path('/tmp/arion-final-dev-review.py'),out/'execution-sources/serialize-final-development-review.py')
copy(Path(__file__),out/'execution-sources/archive-final.py')
text='''# Chosen V5 stage evidence

This archive contains the completed chosen-source 24-case current-intent comparison and four-arm eight-case development ablation, their exact source/configuration/runtime identity snapshots, raw native selector/formatter traces, logs, and hash-bound assistant reviews. The retained architecture uses original model prompts with qualified earlier-source nomination, same-letter compounds, approved-memory continuity, mechanical forced keep for all-ambiguous choices, and independently keyed preview-result reuse. A stage-window result does not certify full-pipeline semantic safety.

The 24-case comparison contains 16 exposed earlier contrasts plus eight authored controls frozen before chosen-source generation. Candidate observation review finds all nine supported repair finals and all 15 preservation finals correct. All 48 phases are reviewed as correct within this authored scope. The baseline has 39/48 correct phases, missing nine supported repairs. Baseline24 is an explicitly documented composition: original fresh baseline16 raw observation lines plus completed new8 baseline2 raw lines, retained verbatim with original streams, per-line/source hashes and lineage metadata. No load event or generation was fabricated. These are separately timed runs; nonrandom arm order and host drift limit causal latency attribution.

Development assistant review counts are baseline12/16, no-prior14/16, no-screen10/16 and candidate15/16. The surviving candidate development failure is an already-wrong first native ASR draft, Chapter fly. Candidate has no manually judged correct-preview-to-wrong-final entity regression in these eight pairs. Case-only canonicalization of chatterfly to Chatterfly is semantically correct despite an exact lowercase substring proxy failing.

The first new8 baseline attempt failed at compilation because current candidate-specific ambiguity tests referenced a helper absent from the frozen baseline engine. It performed zero model inference. Its complete sources/logs/status remain under `runs/final-intent-baseline/`. Baseline2 uses an explicit test-only Go overlay replacing that candidate-specific test file with a package declaration; the actual native stage harness and baseline production source remain active. Both runner revisions are preserved in the respective source snapshots.

Reviews were written by a separate assistant task that inspected all actual observations, not independent human/blind evaluation. No natural-speech held-out accuracy, UI-delivery acceptance or global semantic pass follows. Prior rejected V1/V3 experiments remain in the neighboring immutable rejected-v1 archive. Actual Coordinator, full-audio, cache and still-failing native semantic scenarios are archived separately under `native/final-v5`; inspect them before interpreting this stage-only result. Known native06 invented-negation and native07 unresolved-reviewer-name risks remain material.

Verify this archive using the study report tool with `--verify` and this directory. `manifest.json` preserves copied bytes, every SHA-256 and original-to-archive path mapping. Rebase original absolute overlay paths with those mappings for replay. Native runtime/worker binaries and weights remain separately provisioned and are not copied.
'''
(out/'README.md').write_text(text);files['README.md']=sha((out/'README.md').read_bytes())
(out/'manifest.json').write_text(json.dumps({'schema':'rewrite-stage-final-evidence-v1','files':files,'pathMappings':mappings,'scope':'Chosen-source completed stage contrasts and development ablations; full native semantic safety is not certified'},indent=2)+'\n')
print(out);print('files',len(files),'bytes',sum((out/p).stat().st_size for p in files))
