# Scene rewrite alignment experiment

The [alignment and acceptance proposal](../../../docs/research/scene-rewrite-alignment.md)
defines the desired product behavior and separates client evidence from unknown
competitor services. This is actual local Qwen3 1.7B INT4 text inference on
48 single-author synthetic cases, not a release benchmark or competitor output
comparison. The 24 calibration pairs are separately authored validation inputs.
No labels, constraints or references are supplied to either model.

`protocol.json`, templates, corpus and calibration were frozen before generation.
`freeze.json` also pins generation/build/scoring code. Development runs and
calibration precede a frozen selection decision; test family generation follows
that decision. Test language siblings stay in the same content-family split.
`study.py` records all completed, rejected, truncated, timed-out and malformed
observations with explicit run status. No failed observation is replaced.

Two generation treatments (installed production, isolated scene-aware overlay)
share weights/provider/threads/greedy decoding and engine budgets. Three delivery
treatments (strict, envelope, envelope plus local semantic verifier) are evaluated
on each identical raw generation. Envelope-only is a diagnostic arm, not a
proposed safe default. All local model verifier results remain fallible evidence.

`build.py` creates a Go overlay in `.workspace/` that adds explicit experimental
template loading, actual scene tags and closed context sections. The audit worker
also bypasses speech cleanup for its structured source/candidate payload, so
neither text is silently altered before comparison. No production source, app
installation or product preferences are changed. Frozen overlay files and their
hashes are retained in the run archive; weights and binaries remain private.

## Reproduction

Provision the catalog-pinned model under the standard Arion model directory and
the installed native worker/runtime. No command downloads weights or a runtime.
This inference workflow targets the documented Apple Silicon macOS runtime;
model-free reports and verification need only Python's standard library.
Use the root package manager pinned in `package.json`. `EVAL_PYTHON` can select
the separately provisioned Python interpreter.

```sh
pnpm rewrite:alignment:build
pnpm rewrite:alignment:run --split dev --out .workspace/alignment-repeat-dev
pnpm rewrite:alignment:run --split test \
  --selection .workspace/alignment-repeat-dev/selection.json \
  --out .workspace/alignment-repeat-test
pnpm rewrite:alignment:report \
  --dev .workspace/alignment-repeat-dev \
  --test .workspace/alignment-repeat-test \
  --out .workspace/alignment-repeat-report
pnpm rewrite:alignment:verify
pnpm rewrite:alignment:test
```

Build and run directories must be new. The build directory is explicitly
`.workspace/scene-rewrite-alignment-v1/build`; retain it as evidence or move it
aside explicitly before rebuilding. Do not replace archived results.
Make exposes the same operations as `rewrite-alignment-build`,
`rewrite-alignment-bench SPLIT=... OUT=... SELECTION=...`,
`rewrite-alignment-report DEV=... TEST=... OUT=...`,
`rewrite-alignment-verify`, and `rewrite-alignment-test`.

`freeze.py` is the original fixture construction source. It refuses to overwrite
its outputs and is not part of ordinary replay. Its initial freeze is extended
with `study.py` before the first inference. Reporting/verification code is
archived separately and does not retune the generation policy.

## Evidence and limitations

The [raw archive](../../runs/scene-rewrite-alignment-v1) retains both splits,
calibration, loads/stderr, source/model/runtime/binary/fixture/row identities,
effective overlay source, complete dev selection, summaries and report replay.
The report includes 288 blinded delivered-output review tasks, a separate private
arm mapping and an unrated rubric requiring at least three independent raters.
No independent ratings are fabricated.

Fact patterns and layout checks are explicitly proxies. They can miss invented
claims, actor-role relationships, condition scope or an answer containing required
words. Inspect actual outputs and the failure notes; do not call these counts
semantic accuracy. The same checkpoint generates and audits, so correlated errors
remain possible. Timings are monotonic worker request observations on a shared
host; background apps are uncontrolled. Cold model/prefix load is recorded but
not included in warm final times. There are no native stop-to-insertion, live
preview, microphone, OCR, ASR-contention or power-isolated performance claims.
