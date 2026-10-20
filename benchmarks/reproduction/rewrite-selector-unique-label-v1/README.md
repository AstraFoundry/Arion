# Unverified unique-label exposure

This study removes only `status:unique` from every model-facing hypothesis in the ten original examples and current user message. It retains `status:ambiguous`, all spelling options, `kind`, full source/OCR, scene, original/replacement comparison data, trusted rules and assistant demonstration decisions. In particular, the original system's unique-is-not-correct clause remains. The actual candidate producer and its prepared output do not change; omitted metadata is not silently changed into a confirmed spelling.

Seven payload checks include exact restoration against both original controls across all457 frozen preparations. Reinserting the missing unique field before `kind` restores complete original request bytes. The334-file freeze precedes inference. Actual1,881,825,088-element Qwen3.5-2B Q4 uses b11734/Metal99/CPU4/context4096/parallel1/greedy42/reasoningoff/cachetrue/128tokens/no-warmup. All16 scene workers actually offload25/25 layers.

457 fresh actual Go speech preparations are reused across four arms.1,828 source-treatment outcomes include1,424 no-hypothesis skips, with404 actual model choices. Natural120 sources per arm remain skipped, not new natural-body inference. There are zero newASR, Vision or formatter generations; no4B or olderQwen inference. Source identity/enum/exactUTF8 application remains, without semantic output rejection or source fallback.

| Arm | Conforms /101 | Harmful cases | Missed repair cases | Required repairs /47 |
| --- | ---: | ---: | ---: | ---: |
| Baseline |92|3|6|41|
| No unique label |91|4|6|41|
| Balanced |94|4|3|44|
| No unique label + balanced |91|7|3|44|

Every one of404 raw responses and edited sources was personally read. All13 unique failed complete-source/context/hypothesis/output tuples and all4 changed within-run pairs were separately reviewed. Every response finishes normally; protocol failures are zero. Concurrent controls reproduce202/202 previous raw choices and edited sources. Every new request restores exactly after unique-label reinsertion. Each arm retains12 original-only ambiguity schemas; protocol success there is not an autonomous semantic decision.

## Decision and interpretation

Reject both label-omission variants. All4 changed choices are regressions, with no improved choice or additional required repair. The baseline variant newly changes a customer-interaction word using unrelated travel OCR. The balanced variant newly changes nursery cultivation into software configuration with plain or imperative wrong-domain OCR; two of these are duplicate source/context cases. Four changes therefore do not mean four independent utterances.

The same-letter read cash to readCash normalization, or keeping the source, is accepted. Unsupported readCache remains a failure. Neither removing a metadata field nor returning well-formed JSON establishes faithful correction. These cases do not fail because structured writing differs from the original: no new formatter or final body is generated.

This tests the literal unique-label field only. Singleton option counts still reveal candidate cardinality, and the system's unique clause remains; it does not remove all confidence cues or identify a general causal effect of metadata. Tokenization/cache consequences are not independently controlled. No benefit, production adoption, full-chain/fresh expansion or release qualification follows. Observed serial selector timing in the summary excludes startup/Go/formatter/native insertion/concurrentASR and is not an end-to-end or minimum16GB SLO.

Complete457-source rewriting, fresh/currentASR+Vision, cross-script terminology, native lifecycle/offline/concurrent stop-to-insertion, independent meaning/style review and actual Apple Silicon16GB acceptance remain open on the M4/32GB host. Existing Chinese baseline/balanced prompts remain experimental references.

## Reproduction

Use `pnpm rewrite:selector-unique-label-v1:run --out <new-private-directory> --cleanup-worker <pinned-worker>`; `:payload-test`; `:test`; `:verify`; `:report --out <new-directory>`. Weights/runtime/executable stay private. The owned adapter injects its declared pipeline/freeze into the unchanged four-arm actual-Go/Metal executor. Internal executor/worker labels remain selector-screen-v1. Omission-arm prompt identity points to the composite layout recipe with pinned original system/example sources and complete actual message hashes.
