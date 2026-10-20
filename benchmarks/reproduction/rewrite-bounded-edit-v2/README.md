# Bounded-edit v2

All36 exposed regressions receive two actual native calls: source-relative spelling choice with full OCR, then complete visible edited source without OCR. Frozen [protocol](protocol.json) and [source identities](freeze.json) precede all72 observations. Qwen3.5-2B Q4/nativearm64b11734/CPU4/GPU0/cache off,128+256 output-token caps. Original v1 source remains immutable; v2 pins/reuses its candidate/payload helpers and fixed general pronunciation resources. The versioned runner and verifier snapshots retain v1 lineage without modifying frozen evidence.

Changes: source English2..6-word windows fix embedded English/mixed subspans; source-letter-compatible case/space normalization precedes other homophones; selector examples include positive English/ordinary-word/recipient repairs. Full corrected names/identifiers stay visible to the formatter instead of private tokens. This is a whole-recipe comparison, not an isolated prompt intervention or production adoption.

All72 raw responses, hypotheses, edits, source/context and36 final bodies personally inspected. Selector36/36 conform; final34/36 faithful and33/36 scene-ready under the frozen explicit-contrast requirement. All11 local spelling repairs survive, but only10/11 repaired messages are entirely faithful. One formatter adds records to a function-investigation request, narrowing scope. Another keeps four seconds but omits explicit not-forty-seconds emphasis. That latter issue concerns the experiment's emphasis criterion, not an incorrect value; [source-bound caveat](../../runs/rewrite-bounded-edit-v2/run/review-caveats.json) preserves original scores while a user preference is pending. Oral filler remains; quotes' payload stays exact while delimiters can change outside verbatim mode. Candidate not selected, fresh24 unused, production unchanged.

```sh
pnpm rewrite:bounded-v2:report --out .workspace/bounded-v2-report-repeat
pnpm rewrite:bounded-v2:test
pnpm rewrite:bounded-v2:verify
```

Make aliases mirror these commands. New inference needs separately provisioned exact weights/runtime and a new output directory via `pnpm rewrite:bounded-v2:run --out ...`. Report/verification load no model.

Actual CPU two-call text p50≈8.50s,p95≈11.15s; maximum post-response sampled workerRSS3,378,495,488 bytes, coordinator190,062,592 bytes. Not peak/full-app memory, minimum AppleSilicon16GB, continuous ASR/native stop-to-insertion performance or a causal timing comparison. Current machine M4/32GB. Independent natural ASR/OCR, languages, other scenes, lifecycle and production integration remain open.
