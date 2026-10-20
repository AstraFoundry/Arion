# Final source review: contextual rewriting

This is an assistant-authored source review using the code-review skill, not independent human evaluation or a semantic quality certification. It reviews the final retained pipeline, not the rejected fixed-gap or prompt-expansion variants. No product files were changed during this review. No model was loaded or inference run for this review. Actual paired generation remains the adoption evidence.

## Result

No additional must-fix product correctness defect was found in the reviewed scope. This is a bounded review, not proof of absence.

- Historical `Prepare` retains its original branches when `compounds=false`. New exact-compound boundaries, lexical intent qualification, earlier-source nomination and all-scene matching are gated behind `PrepareContextual`. Original selector messages/examples remain unchanged.
- All-ambiguous selection is mechanically forced to keep only when every status is ambiguous with at least two alternatives. The generated complete keep map passes normal domain coverage/source-span validation. Mixed or unique decisions still require selection. Unit transport tests prove no selector request and unchanged formatter input for all-ambiguous sources; malformed metadata, offsets and duplicate IDs fail explicitly.
- Cache keys cover session/generation identity, source revision, text, previous text, scene, prefill, approved terms and every context field. Mutable terms/history slices and cleanup spans are copied. Current-run/generation, cancellation, stable-source equality and pending-source guards prevent stale preview storage. The raw cached proposal is validated again for final delivery. Live display punctuation alignment does not replace raw final punctuation.
- `PreviousText` is supplied from current-session rule-corrected stable recognition, not generated preview output. It is forwarded through ports, client, protocol, worker and GGUF request. EarlierSource is excluded from ordinary screen candidate collection and cannot independently confirm itself.
- Earlier-name retrieval requires independent screen/cursor/approved-term evidence, matching complete prior token and local frame, bounded common letter edges and length ratio. Quoted source, known changed names and local correction/retraction/literal-discussion guards restrict nomination. This is still unverified retrieval, not acoustic evidence.
- Approved-memory continuity unions only existing vocabulary recalled from current and previous recognized sources, under the shared eight-entry recall cap with current-source recall first. It does not acquire memory from generated names.
- Formatter requests remain isolated to edited source; foreign context remains in the selector stage. Native token budget checks and cancellation handling remain explicit. The old generic engine ignores PreviousText; the revised producer is implemented by the selected GGUF path.

## Material limits

The lexical intent gate conservatively suppresses some genuine requests to correct spelling. Recognition/mistake cooccurrence is lexical, not full semantic negation parsing. Prior source increases selector input size and can cause explicit budget rejection. Existing ordinary formatting can still corrupt a selected name; no hard entity-preservation contract is claimed after rejecting fixed-gap formatting. Native synthetic audio reproduces the failure class, while an exact constructed `chad fly` source is not original-user-audio evidence.

## Documentation precision notes

The reviewed documentation generally distinguishes static competitor observations, Arion choices and unknown service behavior, and explicitly denies a hard fidelity guarantee. Three wording cleanups were communicated to the parent agent without changing files here:

1. The competitor research table should describe checks in the inspected delayed-write path, rather than a universal assertion that ownership prevents every stale write.
2. The historical version check should qualify its baseline source revision and its non-execution scope, since current source is V5 and separate installed-worker executions exist.
3. Architecture should distinguish unchanged pipe transport from the newly added optional PreviousText protocol field.

## Test evidence

Command: `go test ./packages/core/domain ./packages/core/application ./apps/desktop/infrastructure/llm/client ./apps/desktop/infrastructure/llm/gguf ./apps/desktop/infrastructure/llm/prompt ./models/...`

The command exited 0. Native model tests are opt-in and were not enabled. Output is retained in `final-code-review-tests.log`.

```text
ok  	arion.local/core/domain	(cached)
ok  	arion.local/core/application	(cached)
ok  	arion.local/desktop/infrastructure/llm/client	(cached)
ok  	arion.local/desktop/infrastructure/llm/gguf	(cached)
ok  	arion.local/desktop/infrastructure/llm/prompt	(cached)
ok  	arion.local/models	(cached)
```

Test log SHA-256: `d144884660cfd6c993cb81c3bdcef9e0bd566253e72ec72b0e1cc1a176ceecb8`

## Reviewed source identities

| Source | SHA-256 |
| --- | --- |
| `packages/core/domain/spelling.go` | `6c77fa728632440b9a4601da1577b5e9fe9a648aace8ed426379c10cc2d26a7d` |
| `packages/core/domain/spelling_compound_test.go` | `b8f3cee908139a20e623abf08579f7d7321a9b9a894c2976d1a95e88dddb3d81` |
| `packages/core/application/live.go` | `4b4c2711f5d78efc4bea418cd2e9a10903b8995c193532ca74698b5ca9c05a9c` |
| `packages/core/application/live_reuse_test.go` | `4f0adb03447fc3bb773e1d20f9ba2bc09223e277ddd93cd25400a4f7bee858e5` |
| `packages/core/domain/ports/ports.go` | `9dcc7aa8b3e8c0300d1d655e92874ef27100ab5b6fa09216a391d80a54d837fd` |
| `apps/desktop/infrastructure/llm/client/client.go` | `d76fd3d79a67361c1b61e3566c84dbe4e2d78016eea577e643e95bdda9b3f32f` |
| `apps/desktop/infrastructure/llm/client/client_test.go` | `1aa392064712547cea1148f67eb4c80d5dd1674bcdfbc8854afb0e1c3c0229a5` |
| `apps/desktop/infrastructure/llm/protocol/protocol.go` | `417692d9a1812686e7b5e2533f4301696b17a714e8b6f43f3f271f6f27b3c924` |
| `apps/desktop/infrastructure/llm/engine/engine.go` | `d81b90b71f26e5432e7359c8f0dd001b977d2ddb27633c2547e25be69d9539a6` |
| `apps/desktop/cmd/arion-llm/main.go` | `c41dc2ca458a4e43cf4cff36dfc1db69c40e4a501ddd7efe1f94c05acbf995cc` |
| `apps/desktop/infrastructure/llm/gguf/engine.go` | `21bf2461abb07150da5f73d78b1122e9fe62ea461afd307450dfe792bf282c27` |
| `apps/desktop/infrastructure/llm/gguf/ambiguity_test.go` | `a191ea788d64837d3431843fc6908279f14f65729bc60499a5fd94f74a49f42c` |
| `apps/desktop/infrastructure/llm/prompt/selected.go` | `439f7a4712d4065054b3e5b07c1eaa79d1cc1234271b45c9c16746196b52afaf` |
| `apps/desktop/infrastructure/llm/prompt/selection.json` | `992e89acfa462c54fa893e8319b6e031fa01dcb454ab166651abc834d2c4c35a` |
| `models/catalog.json` | `ecdc9b4bf6d3fd0498a8702b346363233d517f7590d16f60f688eb0b3e0b0d10` |
