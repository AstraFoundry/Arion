# Candidate sentence previews

This experiment changes selector data only. Every unverified candidate includes its spelling and the complete source after that one exact UTF-8 substitution. The semantic-v2 system, demonstrations/answers, hypotheses, schema, decoding and latest four-scene formatter remain fixed. Removing the preview fields from every actual selector request reconstructs the frozen baseline request exactly, including demonstrations. No evaluator/reference text or omitted context is added.

The260-file freeze precedes inference and includes the complete prior semantic-v2 raw/labels/manifest/selection/archive. The same actual1.88B Qwen3.5-2B Q4 model runs with Metal/CPU4/4096 context and no thinking. All558 current responses (101 choices/457 final bodies) were personally read. There are457 actual Go cleanup attempts:401 historical sources,24 old controls and32 role controls. No4B, older Qwen, new ASR or new OCR inference was added.

| Paired development result | Semantic v2 | Candidate preview |
| --- | ---: | ---: |
| Historical useful bodies /401 |346|347|
| Old typo repairs /8 |7|8|
| Related-context role repairs /4 |0|3|
| Old-control faithful bodies /24 |24|22|
| Old-control useful bodies /24 |21|20|
| Harmful role substitutions /32 |3|10|
| Role faithful bodies /32 |29|22|
| Role useful bodies /32 |16|16|
| Total useful bodies /457 |383|383|
| Total unfaithful bodies /457 |3|12|

The candidate is rejected. Fourteen intermediate choices change, all from keep to replace: five reviewed corrections and nine harmful substitutions. All14 corresponding final bodies change;443 other formatter requests are identical. Overall usefulness conceals much worse source fidelity. Natural120 bodies are unchanged:119 confirmed/one conservative ASR-source uncertainty/75 useful. Same-letter readCash normalization remains correct; no unsupported readCache substitution is introduced.

## Failure analysis

The preview helps the model repair `工时通记` and three related-context role errors. It also lets unrelated OCR replace correct source objects: bookshelf `书架` becomes holiday `暑假`, and legal `合法` becomes drinking method `喝法`. Domain-conflicting candidates change server configuration into cultivation, deadlines into implication and import lists into rhetorical relevance.

A separate error affects the forbidden alternative in the nursery sentence: the producer's actual candidate is the later `配置` span, not the earlier cultivation noun. The selector changes `别改成系统配置` into `别改成系统培植`. Checking only whether the main noun or an evaluator's forbidden-replacement substring changed would miss this failure. The full source, byte span, candidate preview and final body were reviewed together. Blanket exclusion of every negated noun would also block valid spelling repair in prohibited actions; that is not adopted.

All these failures originate in actual spelling choices, then propagate through the isolated formatter. They are not correct rewrites rejected by a strict semantic validator: no semantic output rejection or source fallback is active. A produced near-homophone and a fluent replacement sentence do not establish that the replacement preserves the referent or meaning.

One-sided replacement changes are observed. Extra target repetition and after-only framing are possible mechanisms, not proven causes. A balanced original/alternative representation without redundant target repetition is a next controlled hypothesis. An explicit entity-name/ordinary-word distinction and another eligible small selector model are also possible directions; no such improvement is claimed here.

The eight fixed role utterances across four contexts are not32 independent samples. Exact source wording inside some instructions suppresses hypotheses, so those context conditions do not isolate instruction effects alone. The previously frozen fresh set was not consumed because the joint development gate fails.

## Evidence and limits

The v3 executor records actual frozen template ownership and message hashes. Replay verifies complete current requests, source preparation, conditional choices, unfiltered final bodies, coverage and both source/raw identities in every baseline pair. Actual previews are visible in the living HTML, alongside their source spans and raw decisions. All101 choices have valid protocol; only88 conform to the reviewed correction requirements.

Median selector input increases by355 tokens; maximum input is1740 tokens, within the actual4096 context. This is a paired input-token observation, not a latency claim. Unprimed text-chain p95 is approximately2018ms for281 synthetic historical sources and1573ms for120 natural sources. A read-only midrun load observation is retained; it does not identify the cause of timing drift or prove absence of throttling. Native integration, ASR plus Vision, concurrency, lifecycle/once-only/offline, independent ratings and actual Apple Silicon/16GB acceptance remain open. Production prompts/validators are unchanged.

The initial derived report before supplementary observations is retained with an explicit evolution record. Raw responses, semantic labels and scores are unchanged.

## Commands

- `pnpm rewrite:selector-preview-v1:verify`
- `pnpm rewrite:selector-preview-v1:payload-test`
- `pnpm rewrite:selector-preview-v1:test`
- `pnpm rewrite:selector-preview-v1:report --out .workspace/preview-report-replay`
- `pnpm rewrite:selector-preview-v1:run --out <new-private-directory> --cleanup-worker <pinned-worker>`

Weights, runtime and executable workers remain private; reusable exact inputs, outputs, reviews and reproduction sources are archived under benchmarks.
