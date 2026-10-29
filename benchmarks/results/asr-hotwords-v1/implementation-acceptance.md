# Hotword implementation acceptance

These checks are separate from inference-quality observations in [the paired report](README.md). Commands below were executed against the accompanying working tree; this is an observation summary, not a raw terminal log.

| Requirement | Implementation and observed evidence |
| --- | --- |
| Correct-only approved ASR terms, bounded deterministic ranking | `packages/core/domain/hotwords.go`; domain tests and race run pass |
| Learned-term linkage, legacy compatibility, independent manual hotword, atomic encrypted persistence | `infrastructure/vocabulary/hotwords_test.go`, `variants_test.go`; domain/store race run passes; bootstrap action tests pass |
| One recording owns one immutable snapshot | `application/hotwords_test.go`: both polishing modes, next-session refresh, defensive copying and segment finalization; application race run passes |
| Preview and final native transport | `infrastructure/asr/hotwords_test.go`: MLX and speech.cpp capability fixtures, empty reset, invalid context and unsupported worker errors, retained streaming snapshot and offline finalizer; ASR race run passes |
| Existing cancellation/lifecycle invariants | Full ASR and application suites pass under race detector; native worker changes reviewed for ownership/cancellation |
| Real model context | [Native smoke](../../reproduction/asr-hotwords-v1/runtime-smoke/README.md) and [paired production inference](README.md); source/contract checks alone are not counted as model quality |
| UI in both locales | Full `ui/app` suite passes; rendered and native evidence is retained under `benchmarks/runs/asr-hotwords-v1/ui` |

Executed passing commands (Go cache: `/private/tmp/arion-hotword-go-cache`):

```sh
go test -race ./packages/core/application ./packages/core/domain ./apps/desktop/infrastructure/asr ./apps/desktop/infrastructure/vocabulary
go test ./apps/desktop/bootstrap
go test ./apps/desktop/infrastructure/cloudpolish
go test ./apps/desktop/infrastructure/modelruntime ./apps/desktop/cmd/arion-eval-driver ./apps/desktop/ui/entities/vocabulary ./apps/desktop/ui/shared/copy
go vet ./packages/core/... ./apps/desktop/infrastructure/asr ./apps/desktop/infrastructure/vocabulary ./apps/desktop/bootstrap ./apps/desktop/ui/features/vocabulary
python3 benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/verify.py
```

The modelruntime, evaluation-driver, entity and copy packages were observed passing in a combined invocation whose cloudpolish package failed to bind localhost in the sandbox. Cloudpolish subsequently passed with escalation. Bootstrap likewise required escalation for its existing local HTTP fixture. Initial session-snapshot tests incorrectly expected generation increments of one even after cancellation; they failed, were corrected to use the actual request generation, and then passed. No production fallback was added for any of these failures.

Limitations: native Windows inference/window acceptance was not run on this macOS host. Speech.cpp context support is verified against pinned source and the shared Go protocol fixture. Development inference timing is offline compute, not microphone-to-insertion latency. Product hotword caps are bounded policy, not a measured optimal vocabulary size. Learned expiry retains the existing prune-on-learning lifecycle.
