# Clean demonstration and policy language diagnostic

This failed development study retains labeled packaging, the same local model and
native worker. `baseline` is the previous aligned template; `clean-en` changes
policy and demonstrations, removing hostile instructions from examples;
`clean-zh` uses equivalent Chinese policy and the exact same clean demonstrations.
Thus only the clean-en versus clean-zh pair isolates policy language. Hostile OCR
remains in test inputs. Nothing is silently removed from context.

Run `python3 benchmarks/reproduction/rewrite-prompt-isolation-v1/run.py --phase dev --out <new-directory>`
on the recorded macOS worker/runtime/model. All 54 outputs were personally
inspected. Baseline fidelity 16/18, scene-ready 12/18, required repair 3/5.
Both alternatives fall to 14/18, 10/18 and 2/5 respectively. English drops a
no-settings-change request; Chinese obeys a hostile DONE instruction. Neither
solves task grouping. The frozen decision retains baseline and does not execute
evaluation. Fresh cases remain ungenerated in this study and may only be reused
with explicit provenance before any new experiment output is seen.

Replay validation and all output review:
`python3 benchmarks/reproduction/rewrite-prompt-isolation-v1/report.py` writes a
new report directory and refuses overwrite. The archived report already exists;
use `collect()` for model-free validation without rewriting it. `freeze.json`
pins pre-generation inputs; `archive.json` pins completed evidence. This is an
assistant-reviewed synthetic Chinese text diagnostic, not independent review,
real OCR/ASR or a production change.
