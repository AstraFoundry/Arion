# Trusted instruction-language comparison

This study changes only `messages[0].content` from the existing Chinese system rules to an English translation. All nine Chinese clauses have an authored alignment; their concatenation exactly reproduces the prior system. The English clauses form the new instruction. Translation equivalence is assistant review, not an independent linguistic judgment. No rule, example, source, complete OCR, candidate, assistant example decision, output schema or decoding setting is added or removed. Source speech and examples remain Chinese/mixed; this does not test translation of dictated content.

Two eligible pinned configurations run separately: actual1,881,825,088-element Qwen3.5-2B Q4 and1,461,538,368-element Granite4.0-h-1b Q4. Both use nativeb11734/Metal99/CPU4/context4096/parallel1/greedy42/reasoningoff/cachetrue/128tokens, with no warmup. Qwen runs before hybrid; the models are never simultaneously resident. Actual offloads are25/25 and41/41; all16 scene workers match their recorded model chat templates. No4B or olderQwen inference occurs.

The411-file freeze precedes inference.457 original actual Go preparations are replayed across four model/arms; no cleanup is regenerated.1,828 complete source outcomes include1,424 no-hypothesis skips. Natural120 sources per arm remain in skipped scope, not new natural rewrite observations. All404 actual selector responses finish normally with zero protocol failure. No newASR,Vision,formatter or final-body inference occurs.

| Model / arm | Chinese conforms /101 | English conforms /101 | Chinese harmful | English harmful | English missed cases | English required repairs /47 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Qwen / baseline |92|88|3|10|3|44|
| Qwen / balanced |94|87|4|12|2|45|
| hybrid / baseline |81|80|20|21|0|47|
| hybrid / balanced |82|79|19|22|0|47|

Every actual request restores exactly to its same-model prior Chinese control after replacing only the first system message. Chinese controls are prior actual observations, not regenerated or newly counted. All404 raw outputs/edited sources were personally read. All24 unique failed full source/context/output tuples and23 changed within-model pairs were separately reviewed. Qwen has19 changes:4 improve,15 regress. Hybrid has4 changes, all regressions. These are repeated exposed development cases, not independent blind/generalization evidence. Each model/arm retains12 forced-original ambiguity schemas; their protocol success is not an autonomous semantic decision.

## Decision

Reject all four English-system configurations. Increased positive repair in Qwen does not compensate for unsupported object/domain substitutions. Failures include nursery cultivation becoming software settings, a place becoming an action, and an explicit non-animal definition becoming an animal. Hybrid still misreads the explicit cash/not-cache boundary; balanced English additionally changes purpose to a musical instrument. Literal letter normalization such as read cash to readCash, or retaining the source, remains accepted. This is not overlap-based rejection.

Keep the prior Chinese candidates as experimental references; they are not newly release-qualified. No formatter/fresh expansion or production activation occurs. Existing identity/enum/exact-span application remains; no semantic output rejection or source fallback is introduced. Language only identifies these two translations/configurations on this exposed corpus. Tokenization/prefill consequences, cache/order and wording nuances are not independently controlled, so do not infer that English instructions are generally worse or claim causal latency benefits.

Observed serial request medians are244/342ms for Qwen and605/608ms for hybrid in baseline/balanced order; nearest-rank p95 is374/547ms and1113/1051ms. Startup, preprocessing, final formatting, native insertion and concurrentASR are excluded. These are not full-chain or actual16GB measurements. Cross-script candidate coverage, complete-chain/fresh/currentASR+Vision, native lifecycle/offline/concurrent, independent meaning/style and actual Apple Silicon16GB acceptance remain open on the M4/32GB host.

## Reproduction

Use `pnpm rewrite:selector-instruction-language-v1:run --model-key qwen|hybrid --out <new-private-directory>` for one202-call model partition. `:payload-test` checks all457 source projections and unchanged non-system messages; `:test` checks complete paired evidence and corruption boundaries; `:verify` checks the archive; `:report --out <new-directory>` replays both partitions without inference. Weights/runtime remain private.

The driver derives from the immutable model-comparison executor, with a model-keyed freeze and an explicit system-only request projection. Historical controls and verifier sources remain unchanged. The versioned language evidence checker validates real English requests and original Chinese references separately; it never rewrites historical rows or declares them exact-request matches.

Earlier prompt-isolation language studies also changed demonstration content relative to their original baseline and used another deployment configuration. This study preserves every original demonstration and tests the current small configurations; old results are not reclassified.

The next untested diagnosis concerns reference-data boundaries rather than instruction language. Current best Chinese controls still fail on several wrong-domain imperative OCR strings. A separately frozen context role/serialization experiment must preserve all OCR, positive repairs and the original release scope; no benefit is claimed before actual inference.
