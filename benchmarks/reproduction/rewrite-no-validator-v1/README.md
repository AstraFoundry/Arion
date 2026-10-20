# Prompt optimization and text-validator ablation reproduction

This research experiment evaluates improved clause/condition/intent preservation
and complete **text** validation removal. Production source/installed behavior
is unchanged. See [results](../../results/rewrite-no-validator-v1/README.md),
[all ungated evaluation text](../../runs/rewrite-no-validator-v1/report/without-validator.md),
and [all policy deliveries](../../runs/rewrite-no-validator-v1/report/review.md).

`freeze.json` pins pre-generation templates, development/regression/new cases,
evaluator-only intended references, protocol, runner and prior-template
identities. It does not pin later report implementations as inference inputs;
`archive.json` separately pins those implementations and all completed evidence.
Do not overwrite runs, refreeze inputs or tune from exposed test output.

Sixteen development cases compare minimal/anchored examples. Freeze selection
only after inspecting all source/context/proposal/cleanup pairs. Then run both
the selected variant and the declared prior-candidate composite on 48 cases.
References and judgments never enter the worker request. The original v1
worker handles speech cleanup, the real production engine and native model;
`policies.py` applies strict/envelope/none to the **same** recorded output.

Inference requires the pinned macOS installed runtime/model and
[v1 isolated worker build](../scene-rewrite-alignment-v1/README.md):

```sh
pnpm rewrite:ungated:run --phase dev --out .workspace/ungated-new/dev
pnpm rewrite:ungated:run --phase evaluation --selection .workspace/ungated-new/dev/selection.json --out .workspace/ungated-new/evaluation
```

Between commands, inspect every dev pair and write a complete `review.json`
and `selection.json` using archived schemas, row/results/review/freeze hashes.
The selected candidate may be frozen for diagnosis without qualification for
release. Evaluation also requires complete explicit proposal and cleanup review.

Model-free replay and integrity checks:

```sh
pnpm rewrite:ungated:verify
pnpm rewrite:ungated:test
pnpm rewrite:ungated:report --dev benchmarks/runs/rewrite-no-validator-v1/dev --evaluation benchmarks/runs/rewrite-no-validator-v1/evaluation --out .workspace/ungated-report-replay
```

Reports validate complete unique coverage, raw/input/template/scene identities,
review-row binding, paired model/build identities and frozen dev selection.
Tests exercise text-gate removal for content, empty, protocol and truncated
outputs, retained deadline/cancellation semantics, and missing/changed review.
All 128 proposals and cleanup values were inspected by the assistant; independent
human and native application behavior are not established. The 384 deliveries
are deterministic replays, not 384 inference calls.

Make exposes `rewrite-ungated-bench` (`PHASE`, `OUT`, optional `SELECTION`),
`rewrite-ungated-report` (`DEV`, `EVALUATION`, `OUT`), `rewrite-ungated-verify`
and `rewrite-ungated-test`. `EVAL_PYTHON` selects the executable. A portable
command wrapper does not establish Windows model/runtime support.
