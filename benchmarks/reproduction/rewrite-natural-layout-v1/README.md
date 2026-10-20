# Natural-layout rewrite verification

The user requested conservative alignment with Doubao and Chatterfly after ordinary dictation was being turned into structured content. The shipped `selector-screen-v1-baseline+natural-layout-v1` retains the baseline spelling selector and replaces the formatter's aggressive work/prompt demonstrations. Scene identity alone does not authorize lists or headings. Ordinary requests and questions stay prose; clear enumeration retains its markers/order or list layout; explicit list requests and the `list` scene use numbered lines. The list scene has a separate recipe and does not receive the default prose policy.

This is Arion's product policy, not either competitor's recovered prompt. Existing client evidence establishes separate cleanup/organization and original/enhanced result handling. Exact server prompts, layout thresholds and current-version behavior remain unknown ([Doubao](../../../docs/research/competitor-disfluency.md), [Chatterfly](../../../docs/research/chatterfly.md)).

## Actual observations

The final `accepted/observations.jsonl` contains 13 authored development probes, one new-session repeat, one generation-free preparation, one deliberately cancelled request and one successful request after cancellation. The actual packaged Go worker and pinned Qwen3.5-2B Q4_K_M / llama.cpp b11734 run locally on Apple M4 / 32GB. Full source requests and responses are retained; no mock inference was substituted. The search probe is a worker-formatting observation; application-level `FinishScene` punctuation handling is outside this harness.

All 13 source/candidate pairs were reviewed by the assistant: facts, actors, questions, negation, uncertainty, scope and corrected values are retained. Ordinary prose probes have no unsolicited bullets or category labels. Both explicit-list probes produce numbered lines. The Chinese enumeration becomes numbered lines; the English enumeration retains its original First/Second markers and order, which is permitted by the conservative policy. Email greeting separation remains functional. See `review.json` and the final native `test.log`.

These probes are exposed authored development examples, many also present in the demonstrations. They are not independent blind review, held-out quality estimates, audio/ASR tests, competitor runtime comparisons, latency qualification, concurrent-ASR checks, 16GB-host acceptance or target-editor insertion evidence.

## Repository and packaging checks

`go test` passed for the LLM packages, rewriting/provisioning, models and portable core. `go vet` passed for LLM and core packages. `pnpm docs:check` passed. `make build desktop` compiled and packaged the native macOS arm64 app; package verification confirmed bundled local rewriting, architectures, resources/licenses, Developer ID signatures, hardened runtime, notarization and Gatekeeper acceptance. This build does not install or launch the app and does not establish real-editor dictation acceptance.

## Failed attempts and policy conflict

All five attempts retain their original JSONL bytes and exact formatter snapshots:

- `attempt-1`: 13 completed probe outputs; the explicit `list` scene incorrectly remains prose. The native test failed that assertion. Its terminal result was captured by the coding session; no separate log file was saved.
- `attempt-2`: adds an explicit-list exception to the general prose policy; the same explicit-list assertion still fails. Original test log retained.
- `attempt-3`: clarifies the list instruction, but the shared prose policy still conflicts with list formatting. The same assertion fails. Original test log retained.
- `attempt-4`: separates the default prose policy from the explicit-list recipe. Both explicit-list probes now work. The test still fails because it incorrectly requires numbered lines for an English First/Second enumeration, even though preserving the original enumeration is permitted. The output and test failure are retained.
- `accepted`: reruns the same final formatter as attempt 4 with the assertion corrected to accept preserved enumeration markers as well as list lines. All layout and lifecycle checks pass. This changes no generation policy relative to attempt 4.

Each attempt also retains its new-session repeat, preparation, cancellation and recovery observations. Failed tests do not have completed-test credit.

## Reproduction and integrity

`source/` snapshots the final formatter construction, native harness and worker implementation. `provenance.json` records the revision/dirty state, host/power context, decoding settings and tested model/worker/runtime hashes. Weights and runtime binaries remain local; their pins are unchanged. `sha256.json` covers every retained file except itself.

From the repository root, with the pinned model provisioned and the packaged runtime staged:

```sh
CGO_ENABLED=0 go build -o apps/desktop/resources/darwin-arm64/llm/arion-llm ./apps/desktop/cmd/arion-llm
ARION_GGUF_RUNTIME="$PWD/apps/desktop/resources/darwin-arm64/llm" \
ARION_GGUF_MODEL="$HOME/Library/Application Support/Arion/models/qwen3.5-2b-q4-k-m-metal" \
ARION_GGUF_OBSERVATIONS="$PWD/.workspace/natural-layout-new-observations.jsonl" \
go test ./apps/desktop/infrastructure/llm/gguf -run TestNativeSelectedPipeline -count=1 -v
```

Choose a new observation path for every run; the test refuses to overwrite an existing file. The native test is opt-in and otherwise reports a skip. Ordinary package tests verify selector parity, source/context isolation, script routing and the absence of conflicting prose policy in the list scene; those are not model-quality evidence. Historical work-scope-v4 templates, outputs and payload fixtures remain unchanged in their original reproduction directories.
