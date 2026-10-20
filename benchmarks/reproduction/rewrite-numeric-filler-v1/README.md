# Numeric and hesitation formatter verification

This focused development check responds to spoken Chinese numbers and decimal points remaining as Han text, and sentence-internal hesitation surviving organization. Natural-layout-v2 preserves numeric value and precision rather than requiring original spoken spelling, and adds shared script-routed quantity and hesitation examples. The selector, acoustic recognition, cleanup rules and delivery checks are unchanged.

`cases.json` contains assistant-authored exposed text probes, not user recordings or held-out examples. `TestNativeNumericFillers` runs the actual packaged worker with each source as live and final requests. Source and expected checks remain separate; expected output is never passed to inference. Assertions check specified numeric spelling and protected phrases; full outputs require semantic review. Historical natural-layout checks remain separate.

Raw baseline, failed attempts, final observations, source snapshots, provenance and hashes are retained here. The 12 authored probes run once with each live/final flag (24 observations): baseline passes 10/24 checks, attempt 1 passes 20/24 with negative/percentage failures, attempt 2 passes 24/24 but fails the separate explicit-list regression. Attempt 3 changes example ordering alone and still fails lists. The accepted version also renders shared examples with list layout when that scene is selected and passes 24/24. Live and final outputs match within each accepted pair. Existing native pipeline checks plus an added numeric-list probe pass; all 14 layout source/candidate pairs were reviewed for scope, negation, scene and preserved facts. The assistant reviewed all accepted outputs: numeric values/precision, negation, conditions, referents, idioms, clock wording and quoted text are retained. These are exposed development observations, not a quality percentage for real dictation.

The M4/32GB run without concurrent ASR records accepted worker requests below the existing 10-second final-rewrite budget; raw elapsed timings include no ASR or editor work and do not establish a latency improvement. The final numeric harness enforces the existing 10-second request deadline. Shared examples add prompt tokens; concurrent ASR and long-input performance remain unqualified.

Run `python3 benchmarks/reproduction/rewrite-numeric-filler-v1/verify.py` to check recorded text assertions and SHA-256 bytes without model inference. Native tests do not capture audio, OS context or editor insertion, establish a held-out error rate, or qualify latency or lower-memory hardware. A live worker flag is not a real streaming audio/session acceptance test.

Reproduce with the pinned local artifacts and a new output path:

```sh
CGO_ENABLED=0 go build -o apps/desktop/resources/darwin-arm64/llm/arion-llm ./apps/desktop/cmd/arion-llm
ARION_GGUF_RUNTIME="$PWD/apps/desktop/resources/darwin-arm64/llm" \
ARION_GGUF_MODEL="$HOME/Library/Application Support/Arion/models/qwen3.5-2b-q4-k-m-metal" \
ARION_GGUF_OBSERVATIONS="$PWD/.workspace/numeric-filler-new.jsonl" \
go test ./apps/desktop/infrastructure/llm/gguf -run TestNativeNumericFillers -count=1 -v
```
