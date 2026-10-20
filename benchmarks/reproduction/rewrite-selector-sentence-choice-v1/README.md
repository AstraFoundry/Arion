# Counterbalanced whole-sentence selector

This development study tests a combined whole-sentence task/data/output framing. The new system compares the original source with full sentences produced by exact local substitutions. Existing balanced comparison records are reversibly encoded as A/B sentence lists. Ten inherited example source tuples and semantic decisions remain unchanged; only their output labels are translated. Source/context, candidate alternatives, decoding and model remain fixed. `sentence-ab` assigns original=A, replacement=B; `sentence-ba` reverses them in all examples/current data and output schemas. This is not an isolated system-text or label-only comparison against the controls.

The final335-file freeze precedes inference. All arms use the same actual1,881,825,088-parameter Qwen3.5-2B Q4, nativeb11734, Metal99/actual25-of25GPU layers, CPU4, context4096, greedy42, nonthinking/cachetrue, maximum128tokens. Actual Go cleanup runs457 times and is reused across four arms, yielding1,828 outcomes and1,424 no-hypothesis skips. The natural120 sources remain within skipped scope, not new natural rewrite inference. No4B or olderQwen is tested.

| Arm | Conforming /101 | Harmful cases | Missed repair cases | Required items /47 | Non-forced conforming /89 |
| --- | ---: | ---: | ---: | ---: | ---: |
| baseline |92|3|6|41|80|
| balanced |94|4|3|44|82|
| sentence-ab |81|19|1|46|69|
| sentence-ba |91|5|5|42|79|

Each arm inherits12 ambiguous cases whose schema permits only the original operation. They are actual responses and protocol observations, but not free semantic decisions. Their alternatives remain visible; the application never silently chooses an ambiguous replacement. The other89 cases have at least one two-option field. All404 responses finish normally without a protocol failure.

## Manual review and decision

All404 actual raw choices and edited sources were personally read. All202 A/B cases were read with complete source/context in explicit ranges. All28 unique failed source/context/output tuples and32 changed balanced-control pairs were separately reviewed. Current baseline/balanced requests and prepared data match old actual requests exactly; raw choices and edited sources reproduce202/202. Manual review is the assistant's judgment, not independent blind acceptance.

All101 A/B request pairs match after swapping only labels/option order and corresponding example/schema labels. Normalized operations and edited sources agree77/101. Of24 changed pairs,20 differ in conformance;4 are both correct but make different permitted letter-normalization choices. Both arms are correct on76 cases and harmful on5. Letter-normalization differences are not mislabeled as semantic failure. These are repeated development pairs, not independent sources or a significance estimate.

The framing is rejected in both orders. A/B improves several required repairs but adds19 harmful cases; B/A still has5 harmful cases and5 missed repairs. Errors include legality becoming a tea-drinking method, a purpose becoming an instrument despite an explicit denial, a destination becoming a customer-interaction verb, and nursery cultivation becoming software configuration. Strong label/position sensitivity is observed across the full example/current encoding. This does not prove a fixed preference for one letter or identify hidden reasoning.

Raw A/B labels, normalized keep/replace, exact edits and review judgments remain separately bound to each actual row. No semantic output rejection or source fallback is introduced; normalization only translates the protocol and existing exact-span application. No final bodies, ASR or Vision generations occur. Production prompts/provider/validators/native integration are unchanged. Neither arm passes the prerequisite for full457-source chain/fresh expansion.

Observed serial request medians are245/384/655/492ms in arm order; nearest-rank p95 is375/764/1278/1109ms, with7 median output tokens. Startup, Go, formatter, native insertion and concurrentASR are excluded. Arm order/cache/load were not reversed independently; these are not causal latency/SLO or minimum16GB measurements.

## Reproduction and failures

Use `pnpm rewrite:selector-sentence-choice-v1:payload-test`, `:test`, `:verify`, or `:report --out <new-directory>`. `:run --out <new-private-directory> --cleanup-worker <pinned-executable>` requires private pinned weights/runtime/executable. Replay and hash verification need no inference.

A pre-inference projection test caught an adapter bug that expanded a keep-only ambiguous schema into both A/B options. The failed source/test/freeze/protocol/host snapshot and cause are retained under the run's `preflight-schema-projection`; it produced zero model responses. The corrected adapter preserves enum cardinality and passes complete457-source projection. This failure is not a native model failure or response count.

Reusable raw results, preparations/skips, manual review ranges/tuples, actual prompt ownership and paired comparisons are archived. Post-launch official bounded model-catalog metadata is separate research evidence and never enters these frozen inference requests; no alternative model was generated in this study. Weights/runtime/executable remain private. Full-chain/fresh/currentASR+Vision/cross-script/native lifecycle/offline/concurrent/independent/actual Apple Silicon16GB acceptance remains open on the M4/32GB development host.

The next evidence-driven direction is a distinct eligible small architecture comparison rather than adopting either A/B order. Official metadata for Granite4.0-h-1b indicates1,461,538,368 elements, Apache2 and Chinese support; quality/runtime remain untested. This hybrid checkpoint is distinct from the previously rejected dense Granite4.0-1b. Its provisioning/preflight, if executed, needs its own frozen experiment and actual outputs.
