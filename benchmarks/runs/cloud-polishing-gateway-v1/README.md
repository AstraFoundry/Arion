# Cloud polishing mock gateway verification

This is synthetic HTTP integration evidence, not model inference or product OS E2E. The gateway runs on a loopback `httptest.Server`, uses an explicit synthetic API key and never reads stored cloud credentials. No real provider is contacted.

## Coverage

All 60 leaf cases passed with Go's race detector enabled:

| Cases | Verification |
| --- | --- |
| 2 | Responses and Chat Completions: speech cleanup, contextual spelling selection, prior source, corrected spelling passed to source-only formatting |
| 16 | Both protocols × all 8 cursor/screen/history switch combinations through the production coordinator; no preview HTTP, two final stages, accepted delivery and exactly one commit |
| 6 | Coordinator HTTP failure, truncation and deadline: locally cleaned source is delivered with explicit unavailable/timeout outcome |
| 22 | HTTP 400/401, malformed JSON, illegal selector choices, truncation and refusal at selection/formatting; no subsequent stage after failure |
| 8 | Cancellation/deadline during either stage, followed by a successful rewrite using the same service |
| 4 | HTTP 429/500 followed by success: identical selector retry and one subsequent formatting call |
| 2 | No spelling candidates: one formatting request, with local filler removal |

The actual path is `Coordinator → Service.Rewrite → shared Pipeline.Run → Republic HTTP transport → mock gateway → production response parser → spelling/delivery validators → synthetic editor commit`. Direct adapter cases start at `Service.Rewrite`. The coordinator's audio, recognizer and editor ports are fixtures. The gateway selects the legal replacement and echoes the formatter source; that behavior must never be interpreted as evidence of model quality.

## Reproduction

Run from the repository root:

```sh
go test -race -json ./apps/desktop/infrastructure/cloudpolish -run '^TestOpenAIGateway' -count=1 -timeout=120s
go vet ./apps/desktop/infrastructure/cloudpolish
```

The test fixtures are `apps/desktop/infrastructure/cloudpolish/gateway_test.go` and `gateway_coordinator_test.go`. The wire envelopes follow the OpenAI Responses and Chat Completions API contracts, including terminal status, incomplete output and refusal fields. The pinned Republic adapter performs actual serialization and parsing.

## Evidence limits

Real cloud credentials, provider model quality/latency, ChatGPT OAuth/Codex transport, microphone recognition, OS context extraction and actual editor insertion are outside this test. The previously recorded local-model timeout/session-erasure failure in [unified polishing](../unified-polishing-v1/README.md) is not resolved by these results.

`checks/` retains raw test outputs; `sources/` freezes the new fixtures and relevant production sources. `manifest.json` records SHA-256 identities and source path mappings. `summary.json` records completed checks and host/build provenance. Early fixture setup required fixing the coordinator constructor return handling and supplying its mandatory ID adapter before execution passed.

## Related regression results

- `cloudpolish`, shared `llm/rewrite`, `bootstrap` and core `application`: passed.
- Complete `ui/app` package: passed with the default Go test timeout. The earlier combined run imposed a 120-second package limit and timed out during `TestAboutContributors`; its original failure log is retained. The production rewrite deadline was not changed.
- `go vet ./apps/desktop/infrastructure/cloudpolish`: passed.
- `pnpm check`: passed once the report's documentation link target existed. The initial missing-target check log is retained.
- The final race run strengthens cancellation assertions to require both an error and empty output; the earlier passing run is retained separately.
