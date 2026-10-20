# OCR alignment prompt correction reproduction

This follow-up fixes the distinction between a legitimate ASR spelling repair
and a factual conflict with OCR. It compares direct/staged wording using the same
examples, freezes a development choice, then compares the selected prompt with
the no-validator-v1 anchored prompt. [Results](../../results/ocr-alignment-v2/README.md)
include all gains, failures and limitations. No production source is changed.

`freeze.json` pins every pre-inference corpus/reference/protocol/template/runner
and prior dependency. Do not refreeze published inputs or overwrite runs.
`diagnosis.json` links the previous exact regression bytes to observed behavior
and distinguishes hypotheses from measured changes. `parity-plan.json` and
`parity_probe.py` are a separately frozen post-experiment diagnostic, not a
retuned prompt treatment. Report and archive hashes pin all later artifacts.

The 16 development inputs are deliberately exposed. Inspect every raw proposal
and completed cleanup before writing `review.json`/`selection.json` using the
archived schemas. Evaluation uses 24 known context regressions plus 24 newly
authored probes, including required repair, correct text, unrelated/corrupt/
ambiguous OCR and hostile context. Evaluator-only intended text never enters
the worker request. Scene metadata is fixture-assigned, not automatic detection.

Inference requires the pinned macOS native runtime/model and the isolated
[v1 production-engine overlay](../scene-rewrite-alignment-v1/README.md):

```sh
pnpm rewrite:ocr:run --phase dev --out .workspace/ocr-new/dev
pnpm rewrite:ocr:run --phase evaluation --selection .workspace/ocr-new/dev/selection.json --out .workspace/ocr-new/evaluation
pnpm rewrite:ocr:parity --out .workspace/ocr-new/parity
```

The selection command requires completed development review between the first
two commands. Review every evaluation and supplementary parity output before
reporting. `policies.py` from the prior frozen experiment is reused directly;
384 strict/envelope/none deliveries are deterministic replays of 128 main
inferences, not 384 model calls. The supplementary parity probe adds 12 real
calls. Authored text does not establish actual ASR/OCR or native performance.
All content review is assistant inspection, not independent human ratings.

Model-free verification and replay:

```sh
pnpm rewrite:ocr:verify
pnpm rewrite:ocr:test
pnpm rewrite:ocr:report --dev benchmarks/runs/ocr-alignment-v2/dev --evaluation benchmarks/runs/ocr-alignment-v2/evaluation --parity benchmarks/runs/ocr-alignment-v2/parity --out .workspace/ocr-report-replay
```

Validation rejects incomplete/duplicate generation or review, changed raw/input/
scene/template identities, unpaired model/runtime builds and changed frozen
selection evidence. Parity validation checks all 12 observations/reviews and
recomputes output equality. Tests check missing repair judgments, raw/review
corruption, scene provenance and every exact exported policy text.

Make exposes `rewrite-ocr-bench` (`PHASE`, `OUT`, optional `SELECTION`),
`rewrite-ocr-report` (`DEV`, `EVALUATION`, `PARITY`, `OUT`),
`rewrite-ocr-verify` and `rewrite-ocr-test`. `EVAL_PYTHON` selects the executable.
The wrapper is portable; recorded model/runtime inference support is macOS only.
