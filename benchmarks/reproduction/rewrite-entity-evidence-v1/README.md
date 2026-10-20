# Entity evidence and actual macOS Vision reproduction

This fixed-model diagnostic changes only the entity-spelling rule: preserve an
exact relevant spelling first, otherwise adopt a unique phonetic match. Both arms
keep all examples, scene rules, full context, local model and native worker/budget.
It records 52 development generations and 48 generations fed by 24 actual Vision
observations on 12 authored images. No prompt was retuned after OCR or inference.
Both arms run the predeclared OCR diagnostic even after development fails selection.

`freeze.json` pins source/corpus/image/template identities before execution.
`render.py` created the images from exact authored text using the recorded system
font; images are already archived and must not be overwritten. Inference requires
the pinned packaging-v1 worker/model and the native OCR executable, built with
`go build -o ../../.workspace/rewrite-entity-evidence-v1/arion-ocr ./cmd/arion-ocr` from
`apps/desktop` (use the absolute output path shown by the archive mapping when
building). Changed source/toolchain artifacts require a new study identity.

- `pnpm rewrite:entity:run --phase dev --out <new-directory>`
- `pnpm rewrite:entity:ocr --out <new-directory>`
- `pnpm rewrite:entity:run --phase ocr --ocr <ocr-run>/observations.jsonl --out <new-directory>`

Inspect every output/cleanup before authoring hash-bound reviews. Do not send
intended names or evaluator references to inference. All 100 outputs and 24 OCR
observations in this archive were personally inspected by the assistant, not
independent raters. Model-free replay: `pnpm rewrite:entity:report --out <new-directory>`.
`pnpm rewrite:entity:test` checks OCR/review binding and missing coverage;
`pnpm rewrite:entity:verify` checks the archive. Make provides matching targets.

[Results](../../results/rewrite-entity-evidence-v1/README.md) explicitly distinguish
correct gold answers from evidence-supported OCR benefit. Several inputs are
exposed probes; the exact 程雨南→程语楠 pair exists in a demonstration and must not
be called held-out generalization. `identifiability.json` is a counterfactual
argument about indistinguishable inputs, not two extra independently scored cases.

Native OCR is actual Vision on fixed images, not live window capture or acoustic
ASR. The Python process runs under Rosetta; archived `host-provenance.json`
identifies Apple M4 hardware and arm64 native OCR/LLM executables. No physical
Intel claim follows from Python's architecture string. Native OCR overlapped the
initial dev model load; timing is not isolated product latency evidence.

For portable model-free replay, `path-mappings.json` maps original production
source paths to exact archived snapshots. External binary IDs remain pinned;
report replay verifies their recorded identities, not the presence of private
weights/executables. Inference still requires the actual matching bytes.
