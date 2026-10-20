# Scene task wrapper reproduction

This study keeps the exact aligned templates and demonstrations from packaging-v1
and compares labeled data with one extra scene-task instruction immediately before
the dictated message. Examples and actual inputs use the same serializer. Complete
sanitized context remains present. The same rebuilt worker, local 1.7B INT4 model,
greedy decoding, four threads and absolute 384-new-token budget serve both arms.
The overlay does not modify production files. Relative timing is within this
study only; input and prefill lengths intentionally differ.

Build: `python3 benchmarks/reproduction/rewrite-task-wrapper-v1/build.py --out .workspace/rewrite-task-wrapper-v1/build`.
Run: `python3 benchmarks/reproduction/rewrite-task-wrapper-v1/run.py --phase dev --out <new-directory>`.
Review all 36 outputs and independent cleanup before writing the hash-bound
selection. Then run `--phase evaluation --selection <dev>/selection.json --out <new-directory>`.
Evaluation has 24 known context regressions and 24 previously ungenerated probes
frozen in prompt-isolation-v1; that study rejected both alternatives at development
and never used its fresh set. This provenance is explicit rather than claiming
newly authored data. `freeze.json` pins inputs before any output in this study.
Never overwrite existing directories or refreeze published data.

Model-free replay: `python3 benchmarks/reproduction/rewrite-task-wrapper-v1/report.py --dev benchmarks/runs/rewrite-task-wrapper-v1/dev --evaluation benchmarks/runs/rewrite-task-wrapper-v1/evaluation --out <new-directory>`.
Run `test_report.py` for evidence-integrity guards. All source/output judgments are
assistant inspection, not independent human ratings. Assigned scenes do not test
automatic scene detection. Authored text is not actual OCR/ASR/native acceptance.
Production activation requires separate qualification and macOS validation.

The development owner success has an explicitly archived review-clarification:
its success note and scores were correct, but one stale issue tag remained when
adapting a row. The clarification removes only that tag; the original review and
selection hashes remain unchanged. It does not change any scores or treatment.
