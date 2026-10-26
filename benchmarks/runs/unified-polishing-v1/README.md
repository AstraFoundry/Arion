# Unified polishing verification

Both cloud and native adapters call `infrastructure/llm/rewrite.Pipeline.Run`. This run verifies the extraction with the pinned local Qwen3.5-2B model and retains a reproducible timeout. It does **not** establish complete product acceptance.

| Check | Result |
| --- | --- |
| `pnpm desktop:check` | Passed: desktop Go packages, `go vet`, native hold and keyboard checks. Opt-in native model and product E2E are separate. |
| `pnpm exec node scripts/check-go.mjs core` | Passed: core/model package tests and `go vet`. |
| Actual staged context corpus | 24 authored cases, preview + final: 48 observations, 47 delivery-format acceptances and one cancellation at 10 s. Completion of the recording is not a quality pass. |
| Isolated timeout recheck | The same final phase cancelled again at 10 s; subsequent native session erase failed. This run is failed. |
| Actual rebuilt worker, `TestNativeSelectedPipeline` | Passed: 14 rewrite cases, new-session parity, generation-free preparation, expected cancellation and successful subsequent request; 18 recorded calls. |
| Real cloud provider | Not run. Cloud adapter/context isolation is covered by local fake-provider tests only. |
| Complete microphone → context capture → rewrite → editor insertion | Not run. Authored context was supplied to model tests. |

## Remaining failure

`final-intent-02/final` exceeded the existing 10-second stage-test deadline in both runs. The isolated recheck additionally reported `private GGUF request failed` during `EndSession`. The worker cancellation recovery test passes on its own input, so this cannot be generalized to all requests. Root cause is unresolved; neither the timeout nor session cleanup policy was relaxed to obtain a passing result.

The staged harness records literal checks for diagnostic use; those checks are not semantic quality scores and some target final-phase text. No blind review, held-out accuracy or cloud latency claim is made.

## Evidence and reproduction

- `stages/`: unchanged original observations, source snapshot, model/runtime identities, host/power metadata and manifest from `.workspace/unified-polish-native-v1`.
- `timeout-recheck/`: unchanged failed recheck from `.workspace/unified-polish-native-recheck-v1`. Its corpus is the `final-intent-02` object selected from the first run's frozen corpus.
- `worker/`: actual pipe-worker observations and log from `.workspace/unified-polish-worker-v1`, plus worker hash and source/build references. Executables and model weights stay in `.workspace`.
- `checks/`: completed desktop/core check logs.
- `manifest.json`: SHA-256 and byte size of every archived file. Original nested manifests keep their exact bytes and historical paths.

Reproduce the context run with a new output directory:

```sh
pnpm rewrite:stages:run --arm candidate --cases benchmarks/runs/unified-polishing-v1/stages/cases.json --out .workspace/unified-polish-next --runtime apps/desktop/resources/darwin-arm64/llm --model .workspace/rewrite-small-models-v1
```

Run the worker integration after rebuilding `arion-llm` into an isolated directory with the staged native libraries; set `ARION_GGUF_RUNTIME`, `ARION_GGUF_MODEL` and a new `ARION_GGUF_OBSERVATIONS` path, then run `go test ./apps/desktop/infrastructure/llm/gguf -run '^TestNativeSelectedPipeline$' -count=1`. This uses real inference and does not replace the installed app.
