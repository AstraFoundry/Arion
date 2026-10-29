# Vocabulary grouping and canonical recall

Arion now groups correction pairs by correct spelling, adds multiple aliases atomically, rejects conflicts without partial writes, and recalls only distinct correct terms into spelling-selection context. Pair IDs, learned provenance, counters, expiry and undo remain separate. This aligns the useful Type4Me grouping/pre-LLM correction behavior while retaining Arion's bounded on-device retrieval. See the [source comparison](../../../docs/research/type4me-vocabulary.md).

## Controlled production-path results

Final deterministic run: [recall-04](../../runs/vocabulary-alignment-v1/recall-04/manifest.json), [replay report](recall-04/report.json). Seventeen authored fixtures include several aliases, word boundaries, no chaining, irrelevant words, earlier recognition, budget exhaustion and a 2,000-pair vocabulary.

| Measure | Pre-change index/recall adapter | Updated production path |
| --- | ---: | ---: |
| Exact corrected text | 17/17 | 17/17 |
| Expected canonical terms recalled | 25/26 | 26/26 |
| Unexpected canonical terms | 0 | 0 |
| Slots exposing incorrect aliases | 33 | 0 |
| Duplicate canonical slots | 8 | 0 |
| Maximum slots | 8 | 8 |
| Warm correction + recall p99 | 9.833 µs | 4.666 µs |
| 2,000-entry probe p99 | 3.375 µs | 2.542 µs |

One prior-source fixture exhausts the old budget with aliases for the same term, losing another relevant term. Canonical deduplication restores that term. Latency is descriptive on the recorded macOS arm64 host, 100 warmed repetitions per case, with baseline first; it is not an independent statistical or natural-speech comparison. The full machine/power/timer/compiler identity is in the run provenance. The baseline is Arion's old code, not an execution of Type4Me.

## Actual local model acceptance

[Native run 03](../../runs/vocabulary-alignment-v1/native-03/manifest.json), [evaluated observations](native-03/report.json): nine cases, each with two harness phases, passed all 18 exact allowed-output, delivery, canonical-context and formatter-isolation checks. Qwen3.5-2B Q4_K_M uses the catalog-verified artifact and pinned local llama.cpp runtime. No cloud calls or mock inference were used.

The selector case recalls `writeValue` from prior recognized text, offers it for the source span `write value`, and returns `检查 writeValue。`. The actual selector wire contains `Terms: "writeValue"`; the formatter sees only `<source>检查 writeValue。</source>`. Incorrect aliases `wrongValue` and `writeValley` enter neither stage's context. Previous-only memory does not insert an unrelated term into `Check the result.`. The model keeps Claude Code and Arion in the other vocabulary cases.

These are synthetic development text checks, not microphone/ASR accuracy, broad fuzzy recall, independent human meaning ratings or Windows acceptance. Final harness phases reuse a warm model cache, so their latency is not representative of a fresh session. No improvement to acoustic recognition is claimed.

## Tests and UI evidence

- Core checks and vet pass; domain/application/storage/shared-rewrite race tests pass.
- Bootstrap race tests pass outside the sandbox; the sandboxed attempt was blocked by an existing localhost account test.
- Full UI app, catalog and vocabulary entity suites pass; focused tests cover multiline submission, grouping/search, per-alias removal, errors, retained drafts and language switching.
- Actual macOS `TestVocabularyNativeGroups` passes in English and Simplified Chinese. Content captures were visually inspected; these are native Content renders, not OS compositor screenshots. The sandboxed attempt could not connect to desktop services; the isolated-profile rerun passed outside the sandbox.
- Storage checks cover authenticated old-format restart, atomic invalid/conflict failures, duplicate approval, learned promotion, write/authentication failure, per-alias removal/expiry, and matcher-consistent Unicode identity.
- Repository/i18n checks, four recall-evaluator tests and archive/report replay verification pass.

Native captures are local review artifacts under `.workspace/vocabulary-alignment-ui/`, excluded from Git. No screenshot was added to the feature branch. Windows native execution, physical microphone dictation and signed installed-app acceptance were not run for this change.

## Retained attempts

- `recall-01`: completed original 15-case population.
- `recall-02`: failed historical-overlay compilation after the shared source comparator was added; candidate observations retained.
- `recall-03`: completed 16-case population with budget exhaustion.
- `recall-04`: completed final 17-case population, including selector-memory input.
- `native-01`: failed sandboxed GGUF loading; no completed inference claim.
- `native-02`: completed original eight-case model set outside the sandbox.
- `native-03`: completed final nine-case model set.

Each run retains its own immutable status, hashes, source snapshot and observations. [Reproduction commands](../../reproduction/vocabulary-alignment-v1/README.md) replay reports without inference and describe a new native run.
