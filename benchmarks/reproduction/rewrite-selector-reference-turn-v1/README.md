# Complete-reference message separation

This experiment moves the complete `context` object from each of the ten existing demonstration user messages and the current input into a preceding `tool` message. Source text, every OCR byte, scene, hypotheses, schema, system rules and assistant demonstration choices remain unchanged. Exact restoration of every request is verified against the concurrent baseline or balanced control, including all457 source preparations. The balanced arm retains both original and replacement sentence data.

Four actual native `/apply-template` requests precede the339-file inference freeze. They perform zero generation. The pinned Qwen template renders each tool message as a user turn containing `tool_response` markers; it does not provide a privileged role or prove instruction resistance. No tools execute, and no new screen image is acquired. Native serialization requests, responses, model identity, command and logs remain archived under the run's `preflight/` directory.

The pinned actual1,881,825,088-element Qwen3.5-2B Q4 uses b11734/Metal99/CPU4/context4096/parallel1/greedy42/reasoningoff/cachetrue/128tokens/no-warmup. All16 scene workers actually offload25/25 layers.457 fresh actual Go speech preparations are reused across four arms.1,828 source-treatment outcomes include1,424 no-hypothesis skips, with404 actual model choices. Natural120 inputs per arm remain in skipped scope, not new natural-body inference. Zero new4B/olderQwen, ASR, Vision or formatter generations occur.

| Arm | Conforms /101 | Harmful cases | Missed repair cases | Required repairs /47 |
| --- | ---: | ---: | ---: | ---: |
| Baseline |92|3|6|41|
| Reference turn |89|9|3|44|
| Balanced |94|4|3|44|
| Reference turn + balanced |91|8|2|45|

All404 responses finish normally with zero protocol failure. Every raw response and edited source was personally read, followed by all14 unique failed complete-source/context/hypothesis/output tuples and16 changed within-run pairs. Original controls reproduce202/202 raw choices and edited sources. Across the16 changed pairs,5 improve and11 regress. These include repeated source/context cases and are not16 independent utterances.12 original-only ambiguity schemas per arm remain separately limited: their protocol success is not autonomous semantic selection.

## Decision and diagnosis

Reject both reference-turn configurations. Source fidelity is different from required-repair conformance: fewer missed repairs do not offset increased unsupported substitutions. Separating context repairs `工时通记` to `工时统计`, deployment spelling and a meeting-minutes case. It also turns nursery cultivation into software configuration and changes a customer-interaction word using unrelated travel text. New errors occur with both plain wrong-domain OCR and imperative OCR. The current failure cannot be attributed only to obeying explicit commands in OCR.

The balanced reference-turn arm stops one imperative `按实` substitution but still makes the same substitution when OCR contains only an unrelated schedule. Tool-response markers therefore do not establish a stable relevance/meaning boundary in this configuration. This is a packaging comparison that changes turn structure, relative placement and tokenization together; it does not isolate an abstract role-authority cause. Same-letter `read cash` to `readCash` or source preservation is allowed; unsupported `readCache` remains a failure.

Observed serial request medians are245/1102/496/1230ms and nearest-rank p95 is374/2018/961/2153ms in table order. Median prompt tokens rise1247→1357 and1718→1828; median cached tokens fall1158→854 and1589→1361. Recorded prefill medians rise141→972ms and340→1054ms. These observations identify prefill/cache work as a latency contributor in this run, not an isolated causal benchmark or end-to-end SLO. Startup, Go preparation, formatting, native insertion and concurrentASR are excluded.

Existing candidate identity/enum/UTF8 application remains. No semantic output rejection, source fallback, final-body expansion, fresh acceptance or production adoption occurs. Choice correctness does not establish structured writing quality. Complete457-source rewriting, fresh/currentASR+Vision, cross-script terminology, native lifecycle/offline/concurrent stop-to-insertion, independent meaning/style review and actual Apple Silicon16GB acceptance remain open on the M4/32GB development host.

## Reproduction

Use `pnpm rewrite:selector-reference-turn-v1:run --out <new-private-directory> --cleanup-worker <pinned-worker>`; `:payload-test`; `:test`; `:verify`; and `:report --out <new-directory>`. Weights/runtime/executable remain private. The owned adapter injects its declared pipeline/freeze into the unchanged four-arm actual-Go/Metal executor. Original internal executor and worker labels remain `selector-screen-v1`; actual transformed prompt identity points to this study's composite layout recipe, with pinned original system/example sources and complete actual message hashes.
