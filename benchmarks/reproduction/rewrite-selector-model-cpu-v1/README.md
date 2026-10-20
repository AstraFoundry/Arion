# Granite CPU-layer selector comparison

This explicit follow-up examines checkpoint quality after the [recorded Metal-path failure](../rewrite-selector-model-v1/README.md). The same official Granite4.0-1b Q4 export is used with requested GPU layers0 and CPU4; actual native logs must establish0/41 layer offload. Default operation offload remains enabled. It is not an automatic fallback or accelerated deployment candidate.

Before inference,347 files are frozen. The small entry-point adapter injects a GPU-layers0 session factory into the unchanged frozen comparison executor. All202 requests exactly match the corresponding actual Qwen baseline/balanced control requests; no rule/example/schema/budget/source edits occur. The factory source, executor, model, tokenizer and runtime are all bound. Cross-model comparisons also differ in provider, tokenizer and architecture; they cannot establish a parameter-count or checkpoint-only causal effect.

All457 source preparations are replayed from the prior actual Go execution. This run executes zero new Go preparations. Each of two arms retains457 source outcomes, including356 no-hypothesis skips. Expected total scope is202 actual choices and914 source outcomes, with zero formatter or final-body generations. Natural120 sources remain among the zero-hypothesis preparation scope; there is no new natural final-body generation.

The frozen101-case evaluator remains outside model input. Existing user permission allows raw spelling or same-letter space/case normalization; it does not permit unsupported cash-to-cache substitution. A keep may preserve source fidelity while missing a required correction. All outputs must be personally reviewed; final-body usefulness is never inferred from JSON success.

The native run completes202 responses, all with stop finish reasons and no protocol errors. Every raw choice was personally read in two complete101-row passes. All18 unique failed source/output pairs and all30 changed Qwen/Granite full-source/context/choice pairs were reviewed separately.

| Configuration | Conforming /101 | Harmful choice cases | Missed repair cases | Required items repaired /47 |
| --- | ---: | ---: | ---: | ---: |
| Existing Qwen Metal / baseline |92|3|6|41|
| Granite CPU-layer / baseline |84|16|1|46|
| Existing Qwen Metal / balanced |94|4|3|44|
| Granite CPU-layer / balanced |86|13|2|45|

Thirty choices differ from the exact-request controls: seven improve and23 regress. Increased positive correction does not compensate for wrong-domain and wrong-object substitutions. Both Granite arms introduce unsupportedreadCache despite explicit cash meaning. All protocol-valid output remains visible; no semantic rejection or source fallback is added. Neither arm passes; no full-chain/fresh expansion or production activation occurs. Passing this screen only permits a complete457-source chain; fresh/native/currentASR+Vision/lifecycle/offline/concurrent/independent/actual16GB acceptance remain mandatory. No4B or olderQwen inference is added.

## Commands

- `pnpm rewrite:selector-model-cpu-v1:run --out <new-private-directory>` performs the fixed CPU-layer diagnostic.
- `pnpm rewrite:selector-model-cpu-v1:report --out <new-directory>` replays completed evidence without inference.
- `pnpm rewrite:selector-model-cpu-v1:test` checks full source/request/review coverage and rejects invented delivery.
- `pnpm rewrite:selector-model-cpu-v1:verify` checks archive identities without loading weights.
