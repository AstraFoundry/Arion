# Focused scene prompt v2 reproduction

This is a frozen prompt-only research experiment, not a production template
migration. [Results](../../results/scene-rewrite-prompt-v2/README.md) and the
[full review](../../runs/scene-rewrite-prompt-v2/report/review.md) include all failures.

`freeze.json` records the pre-inference corpus, protocol, template and runner
identities. `freeze.py` is retained provenance, not a command for overwriting
this published experiment. Do not refreeze or tune from exposed test results.
The v1 archive and its reproduction sources are immutable dependencies.

Use the pinned pnpm from the repository root. No model is needed to verify,
run integrity tests or replay the complete report:

```sh
pnpm rewrite:prompt:verify
pnpm rewrite:prompt:test
pnpm rewrite:prompt:report --dev benchmarks/runs/scene-rewrite-prompt-v2/dev --evaluation benchmarks/runs/scene-rewrite-prompt-v2/evaluation --out .workspace/prompt-v2-report-replay
```

For new inference, first follow the [v1 build instructions](../scene-rewrite-alignment-v1/README.md)
to prepare its isolated worker/validator and provision the pinned local weights.
The runner verifies weights and build inputs and uses the installed Arion native
runtime. New run/output directories must not exist. This recorded build requires
macOS and the installed runtime; the portable command wrapper does not establish
Windows inference support.

```sh
pnpm rewrite:alignment:build
pnpm rewrite:prompt:run --phase dev --out .workspace/prompt-v2-new-dev
```

Inspect every development output before writing `review.json` and a
`selection.json`, following the archived schemas. Hash the exact source/output
row and results file; do not manufacture ratings from validator acceptance.
Freeze selection before evaluation:

```sh
pnpm rewrite:prompt:run --phase evaluation --selection .workspace/prompt-v2-new-dev/selection.json --out .workspace/prompt-v2-new-evaluation
```

Inspect all evaluation outputs, including rejected and unchanged candidates,
and record review judgments before generating a report. `report.py` enforces
complete unique case/review coverage, frozen inputs and templates, paired model
and binary identities, selection/review hashes and each reviewed row's hash.
`test_report.py` exercises missing/duplicate review, changed output, missing
raw generation and raw-hash corruption detection. `verify.py` checks the full
archive checksum manifest. Reports expose all original and generated text.

Make provides `rewrite-prompt-bench` (`PHASE`, `OUT`, optional `SELECTION`),
`rewrite-prompt-report` (`DEV`, `EVALUATION`, `OUT`), `rewrite-prompt-verify`
and `rewrite-prompt-test`. `EVAL_PYTHON` selects the Python executable.

Each case now displays its fixture-assigned scene, effective model scene, scene
evidence and worker-reported prompt version. Scene assignment was explicit in
these experiments; application/browser scene detection was not exercised.
`scenes.json` exposes the same metadata for all cases. The structural-only run
did not record a separate model-scene field, so its direct scene argument is
reconstructed from the frozen runner and explicitly labeled as such.
The earlier rendering and generator are retained under
`report-before-scene-metadata/`; raw observations and judgments are unchanged.
