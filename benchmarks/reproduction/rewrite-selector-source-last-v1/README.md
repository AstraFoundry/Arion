# Source-last selector serialization

This development study isolates top-level JSON member order. Ten example user objects and the current user object move `source` from first to last. System text, example answers, decoded input values, nested candidate order, complete OCR/context, schema and decoding stay identical. Baseline and balanced controls reproduce the earlier actual128token requests exactly. The adapter injects its pipeline/freeze directory into the unchanged screen executor; the internal executor label remains `selector-screen-v1`. Reordered rows bind the owned composite serialization recipe, its base system/example file hashes and full actual message hash.

The334-file freeze precedes inference. All four arms use the same actual1,881,825,088-parameter Qwen3.5-2B Q4, nativeb11734 Metal99/25-of25 actualGPU layers, CPU4, context4096, greedy42, nonthinking/cachetrue, maximum128tokens, candidate producer and457 source tuples. No4B or olderQwen inference is added. Each arm makes101 actual selector requests. Actual Go cleanup runs457 times and is reused across four arms; all1,828 source outcomes and1,424 no-hypothesis skips remain explicit. Natural120 sources are within skipped scope, not new natural rewrite inference.

| Arm | Conforming /101 | Harmful cases | Missed repair cases | Required repairs /47 |
| --- | ---: | ---: | ---: | ---: |
| baseline |92|3|6|41|
| source-last |93|5|3|44|
| balanced |94|4|3|44|
| source-last-balanced |88|12|1|46|

All404 actual raw choices and edited sources were personally read. All17 unique failed source/context/output tuples and17 changed within-run source/context/choice pairs were reviewed separately. No protocol failure occurs. Current controls reproduce old raw choices/edited sources202/202. All202 reordered requests match concurrent controls after restoring only source order. Seven choices change in source-last: four improve and three regress; ten change in source-last-balanced: two improve and eight regress. Source fidelity remains separate from missed normalization under the frozen development rubric; no posthoc evaluator relaxation occurs. Same-letter `read cash` to `readCash` or preservation is allowed; unsupported `readCache` fails.

## Decision and limits

Both source-last arms fail. Moving source last repairs more typos but adds unsupported substitutions: server deployment becomes nursery cultivation, a deadline's on-time becomes implication, a destination becomes a customer-interaction verb, and an explicit system-configuration prohibition becomes cultivation. Source order alone does not reliably constrain meaning under conflicting OCR. No formatter/fresh expansion or production adoption follows.

Observed serial request medians are244/265/575/489ms in arm order; nearest-rank p95 is372/387/1208/1103ms, with7 median output tokens. These exclude startup, Go cleanup, formatting, native insertion and concurrentASR. Arm order/cache/load were not independently reversed; no causal latency or minimum16GB claim follows.

No new final bodies, ASR or Vision generations occur. No semantic output rejection or source fallback is added. Existing protocol/identity/enum/UTF8-span checks remain; they do not determine semantic correctness. Exposed repeated development sources are not held-out or independent release evidence. Full-chain quality, fresh/currentASR+Vision, native lifecycle/offline/concurrent insertion, cross-script coverage, independent meaning/style review and actual Apple Silicon16GB acceptance remain open on the M4/32GB development host.

## Reproduction

Use `pnpm rewrite:selector-source-last-v1:payload-test`, `:test`, `:verify` or `:report --out <new-directory>`. `:run --out <new-private-directory> --cleanup-worker <pinned-executable>` requires the pinned private weights/runtime/Go executable. Report replay and archive verification require no model inference. All raw outputs, failures, source preparations, skips, request-order/control comparisons, manual judgments and original authoring scripts are archived with hashes.

The next untested hypothesis is to compare complete original/replacement sentence meanings with an explicit A/B decision framing while preserving bounded edits. No success is claimed for it.
