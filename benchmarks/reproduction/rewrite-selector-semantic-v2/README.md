# Concise entity-positive semantic selector

This repeats the full457-source candidate chain:401 historical inputs,24 old tone controls and32 semantic-role controls. The same1.88B Qwen3.5-2B Q4 model, Metal runtime, tone-insensitive producer and latest four-scene formatter are retained. Only the selector system instruction and four extra unrelated demonstrations change. No4B or older Qwen was run.

The247-file freeze precedes inference. All558 actual responses (101 choices/457 final bodies) were personally read. Baselines are previous complete archived observations, explicitly paired by raw identity and not counted as new inference. This is exposed development data; no held-out acceptance or causal timing comparison is claimed.

| Observation | Original tone-neutral selector | Semantic v1 | Semantic v2 |
| --- | ---: | ---: | ---: |
| Historical useful bodies /401 |344|338|346|
| Corrected old typo opportunities /8 |3|1|7|
| Useful old controls /24 |16|15|21|
| Harmful role substitutions /32 |6|2|3|
| Relevant-context role repairs /4 |0|0|0|

The nine entity repairs lost in v1 are all restored. Historical questions using `会议记要` and `验受报告` improve in two cases; another `会议记要` case remains unresolved. No historical body changes beyond those two compared with the original tone-neutral candidate. Historical fidelity remains400 confirmed/one ASR-source uncertainty, natural120 utility remains75, and all natural bodies are unchanged. The published-reference restoration observation for `af` → `afraid` remains separate from conservative source-text fidelity.

All24 old controls are source-faithful; seven of eight authored typo opportunities recover. `工时通记` still fails to become `工时统计`. `read cash` → `readCash` remains allowed and correct; unsupported `readCache` is not accepted.

## Why this is still rejected

The role controls still produce three wrong-domain substitutions when the screen contains an instruction: deployment `配植清单` → cultivation `培植清单`, implication `按时` → unrelated `按实`, and customer contact `大理` → unrelated travel `大里`. Four related-context positive controls still keep their incorrect source terms. The policy fails its preregistered joint positive/negative development gate; untouched fresh data was not consumed.

The examples show that removing an overly conservative rule can restore positive repairs, but a more permissive selector still follows some misleading screen text. A phonetic candidate is neither confirmed spelling nor proof of the same referent. The next experiment should make the actual before/after local substitution visible and test referent consistency, while preserving supported name and identifier normalization. It must retain every positive and negative case, not declare success on the subset that improved.

The32 role controls represent eight fixed utterances across four contexts. Context instructions can also change producer availability when they contain the exact source spelling; the import/writing groups are not pure instruction-only comparisons. No universal causal or independent-sample claim follows.

## Provenance and report interpretation

This is the first actual chain run using the versioned v3 executor. Every stage records its explicit frozen template path/hash and the hash of actual messages; the shared replay helper reconstructs and verifies complete requests and delivery. The v1 metadata defect remains in its original evidence and is not silently rewritten.

The living report now distinguishes intermediate choice conformance from the original `faithful` field. In v1 some source-preserving choices had `faithful=true` but `choiceConforms=false` because a supported repair was missed. They must display as choices not meeting requirements. Raw fields and original scores are retained; legacy studies that used `faithful` to score choice correctness have that basis explicitly identified. Filtering applies choice conformance to intermediate choices and fidelity/usefulness to final bodies.

The101 current choices include92 conforming and nine failures: one old typo, one historical term, four related-context role misses and three harmful role substitutions. Protocol validity alone does not establish correctness.

Unprimed text-chain p95 is approximately2480ms for281 historical synthetic inputs and1127ms for120 natural sources. These are not ASR stop-to-insertion, concurrent or16GB acceptance. Native integration, actual ASR plus Vision, focus/revision/cancellation/once-only/offline delivery, independent ratings and actual Apple Silicon/16GB validation remain open. Production prompts/validators are unchanged.

## Commands

- `pnpm rewrite:selector-semantic-v2:verify`
- `pnpm rewrite:selector-semantic-v2:test`
- `pnpm rewrite:selector-semantic-v2:report --out .workspace/semantic-v2-report-replay`
- `pnpm rewrite:selector-semantic-v2:run --out <new-private-directory> --cleanup-worker <pinned-worker>`

Weights, runtime and worker executables remain private; reusable inputs, exact responses, reviews, source provenance and failed outcomes remain under benchmarks.
