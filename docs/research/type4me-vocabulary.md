# Type4Me vocabulary comparison

## Scope and source identity

This report compares Type4Me's vocabulary representation, editing, matching and recognition-to-rewrite flow with Arion's approved correction memory. It records static source evidence and the selected Arion alignment; source inspection alone does not establish recognition improvement. Separate Arion observations are linked below.

| Source | Revision | Evidence boundary |
| --- | --- | --- |
| Type4Me checkout, `/Users/steven/code/type4me` | `86d03d39a46d97b37ee213cc5576441a6717bd06` | Clean checkout during inspection; source paths and line numbers below refer to this revision. No application launch, recording, test run, model inference or personal vocabulary database inspection. |
| Arion baseline | `a1f9db1ce995c0835e1c03da5155ccc9ec39a700` | Baseline comparison uses committed source. The alignment described below belongs to the accompanying change set; concurrent working-tree implementation is not evidence of completed native acceptance. |

Type4Me source references below are checkout-relative paths and line numbers at the pinned revision, written as inline code because the external checkout is not part of Arion. Its own `AGENTS.md` was read, and the checkout was not modified. No Type4Me implementation, user data, credentials, model weights or recordings were copied into Arion.

## Verified Type4Me behavior

### Correct terms and mistaken variants are separate stores

Type4Me has two concepts:

- **Hotwords** are correct-only strings, persisted as a JSON string array in `hotwords.json`. `HotwordStorage.loadEffective()` returns the user store through `load()`; it does not merge built-in words at this revision, despite older comments describing a merge. See `Sources/Type4Me/Services/HotwordStorage.swift:105`, `Sources/Type4Me/Services/HotwordStorage.swift:177` and `Sources/Type4Me/Services/HotwordStorage.swift:213`.
- **Snippets** are flat pairs with `trigger` and `replacement`, persisted in `snippets.json`. Many triggers can share one replacement, but there is no canonical record containing an alias array. See `Sources/Type4Me/Services/SnippetStorage.swift:35`. At this revision, global rule compilation calls `load()`, which reads the user file; comments describing built-in merging should not be read as proof that built-in rules execute (`Sources/Type4Me/Services/SnippetStorage.swift:235`, `Sources/Type4Me/Services/SnippetStorage.swift:384`).

A correct-only term is therefore representable as a hotword. A snippet group needs at least one trigger: the add flow requires an accepted trigger, and a bulk line containing only the replacement is discarded. Neither storage nor the UI makes every snippet replacement a hotword automatically. The correction-learning path does explicitly add the corrected text to hotwords while saving the mistaken-to-correct pair (`Sources/Type4Me/Services/CorrectionLearning.swift:506`).

### The UI groups flat pairs by their correct replacement

`Sources/Type4Me/UI/Settings/VocabularyTab.swift:852` builds groups using exact replacement string equality, preserving first group occurrence. Search matches either the replacement or any trigger using case-insensitive and diacritic-insensitive substring matching. This is list filtering, not recognition-time retrieval.

The group UI supports changing the replacement of every pair in the group, removing the whole group, removing one trigger and adding a trigger (`Sources/Type4Me/UI/Settings/VocabularyTab.swift:1052`). The add form stages several triggers, splitting on comma, Chinese comma or newline, then saves one pair per accepted trigger (`Sources/Type4Me/UI/Settings/VocabularyTab.swift:1226`, `Sources/Type4Me/UI/Settings/VocabularyTab.swift:1268`).

Bulk editing renders `replacement, trigger1, trigger2, ...` and expands each line back into flat pairs. Saving replaces the selected scope's list. The parser splits literal commas, has no quoted-field handling, ignores blank fields and drops lines with fewer than two fields (`Sources/Type4Me/UI/Settings/VocabularyTab.swift:1359`, `Sources/Type4Me/UI/Settings/VocabularyTab.swift:1443`). This is useful workflow evidence for grouping; it is not a robust interchange format to adopt unchanged.

### Runtime matching is explicit replacement, without fuzzy recall

`Sources/Type4MeTransferCore/SnippetPattern.swift:9` removes all whitespace from a trigger, escapes each character, inserts `\s*` between characters and adds ASCII alphanumeric lookarounds. Compiled expressions are case-insensitive. Consequently, a stored trigger can match spacing variants; this does not establish pronunciation similarity, acoustic confidence or semantic equivalence.

Global rules execute in stored order over the evolving output. App-specific rules suppress global rules with the same compiled pattern and run afterward. Later rules can match earlier replacements, so rewrites can chain. The same application pass records which trigger, replacement and scope actually fired (`Sources/Type4Me/Services/SnippetStorage.swift:456`).

No runtime vocabulary retrieval using fuzzy distance, phonetics, embeddings or relevance ranking was found in the inspected vocabulary, recognition, LLM and Intelli Sense source paths. Type4Me does have spelling and pronunciation similarity in `Sources/Type4Me/Services/CorrectionAffinityAnalyzer.swift:16`: Latin Damerau-Levenshtein, Mandarin homophone comparison and cross-script transliteration assess the confidence of a proposed observed correction. Its call from `Sources/Type4Me/Services/ImmediateCorrectionAnalyzer.swift:162` is learning analysis, not automatic fuzzy vocabulary replacement.

### Recognition receives hotwords; the LLM receives the corrected transcript

The traced flow is:

1. The recognition session loads effective hotwords into ASR connection options (`Sources/Type4Me/Session/RecognitionSession.swift:1425`). Local Qwen3 receives a file synchronized from those terms (`Sources/Type4Me/Services/SenseVoiceServerManager.swift:23`).
2. The Qwen3 server reads all nonempty hotwords and constructs a space-separated `context` string. A source comment explicitly avoids a `Vocabulary:` label because it can be echoed into the transcript (`Sources/qwen3-asr-server/server.py:385`); transcription passes this context into the ASR session (`Sources/qwen3-asr-server/server.py:225`). This is an implementation and rationale, not measured leak prevention in this inspection.
3. On the streaming stop path, snippets rewrite the final recognized text before it is passed to the LLM (`Sources/Type4Me/Session/RecognitionSession.swift:2188`, `Sources/Type4Me/Session/RecognitionSession.swift:2214`).
4. The synchronous path likewise uses the snippet-corrected text. Intelli Sense can refresh its target and recompute snippets from the raw transcript for that target before processing (`Sources/Type4Me/Session/RecognitionSession.swift:2286`, `Sources/Type4Me/Session/RecognitionSession.swift:2426`).

The ordinary dictation prompt builder supplies mode instructions, captured context and an optional expression profile; it does not retrieve a vocabulary alias block (`Sources/Type4Me/Session/RecognitionSession.swift:3465`). The traced paths therefore show corrected spellings in the transcript, not a general mechanism exposing mistaken aliases as trusted LLM spelling evidence. Typed/manual input explicitly skips snippet application in the completion path.

### Budgets and conflict handling vary by entry point

Type4Me's inspected Qwen3 context construction joins the complete hotword list. Other providers apply their own limits: Cartesia takes the first 100 nonempty terms (`Sources/Type4Me/Protocol/CartesiaProtocol.swift:35`); ElevenLabs keeps the first 50 terms of at most 20 characters (`Sources/Type4Me/Protocol/ElevenLabsProtocol.swift:50`). These examples do not establish a shared relevance budget across all providers.

| Entry point | Verified handling |
| --- | --- |
| Add several triggers in Settings | Case-insensitive duplicate sources are silently skipped; accepted remaining triggers are saved. The operation can partially accept a submitted group. |
| Add one trigger to an existing group | A case-insensitive duplicate source anywhere in that scope is silently rejected. |
| Vocabulary URL command | Existing source with the same replacement returns `alreadyExists`; another replacement returns `conflict` (`Sources/Type4Me/Services/VocabularyCommands.swift:274`). |
| Bulk edit | The parser does not reject duplicate or conflicting triggers before replacing the list. |
| Correction learning | An existing case-insensitive source is repointed to the new corrected text. If saving mappings fails after hotwords were saved, rollback of hotwords is attempted (`Sources/Type4Me/Services/CorrectionLearning.swift:515`). This is not a single transaction across both files. |

## Arion baseline and selected alignment

Arion already stores encrypted `From`/`To` correction pairs with per-pair IDs, origin, usage/recall counts and timestamps in [the vocabulary store](../../apps/desktop/infrastructure/vocabulary/vocabulary.go). [ADR 014](../decisions/014-edit-learning.md) governs learned-pair expiry, undo, manual-entry protection and one-pass replacement. The baseline [domain vocabulary](../../packages/core/domain/vocabulary.go) matches the longest approved heard or written form at each position, and replacement does not chain. These safeguards remain applicable when several mistaken forms share one written form.

At the baseline revision, `Vocabulary.Terms` emitted `From → To` and deduplicated pair identities. Two aliases for the same correct word could occupy two prompt slots, and matching the correct form could select an arbitrary stored alias to accompany it. The downstream spelling-candidate consumer can interpret identifiers from terms as spelling evidence, so including the mistaken form gives it a role beyond the approved replacement relationship. Type4Me's traced pre-LLM substitution does not justify that exposure.

The selected alignment is:

1. **Retain pair storage and provenance.** Grouping is a view of entries sharing the same written form. Do not migrate to a group-owned ID or combine learning counts, expiry and undo across aliases.
2. **Present one written form with several mistaken forms.** The native vocabulary UI should make the relationship visible and accept several mistaken variants in one explicit edit. Preserve per-pair removal and learning provenance through the grouped view.
3. **Validate the entire explicit edit before persistence.** Trim and validate every value, reject unsafe or identity pairs, and deduplicate equivalent submitted aliases. If any source is already approved for a different target, reject the whole edit with a clear conflict result. A valid partial subset must not be saved. Existing identical pairs may remain idempotent; learning policy remains governed separately by ADR 014.
4. **Recall canonical spellings only.** `Vocabulary.Terms` should return each relevant correct `To` once, whether recall matched an alias or the correct spelling. Mistaken aliases remain replacement evidence and must not become candidate spellings merely because they were serialized in `Terms`.
5. **Preserve the final-source ordering and budget.** Recall current recognized-source matches first, then relevant earlier rule-corrected recognition from this session, deduplicating canonical spellings across both sources within the existing eight-entry recalled-memory budget. Generated rewrite text must never supply recall evidence. See [CTX-06 and LRW-12](../../specs/009-contextual-rewrite.md) and [termsForFinal](../../packages/core/application/live.go). This budget describes recalled vocabulary, not every independently bounded context section or explicitly supplied policy term.

These choices align the useful grouped editing workflow while preserving Arion's encryption, one-pass replacement and learning lifecycle. Implementation and test results belong to the accompanying change report; this source review alone does not verify the native UI, storage-fault behavior or model output after alignment.

## Evaluation boundaries

- Do not copy Type4Me's whitespace-stripping regex, sequential chaining or fuzzy learning heuristics into automatic replacement without a separate evaluation. Arion's existing explicit boundary matching and non-chaining replacement remain the reference behavior.
- Plain correct-only hotwords and ASR contextual bias are a distinct capability from correction pairs and canonical rewrite terms. Type4Me proves a separate store and ASR injection path exist; it does not prove a safe benefit for Arion's selected recognizer, languages or budget. Provisioning, recall policy, false-bias controls and real audio evaluation are required before adoption. See [recognition memory optimization](user-modify-asr-optimization.md) and [keyboard/voice vocabulary evaluation](keyboard-voice-lexicon-plan.md).
- This report does not establish Type4Me's runtime accuracy, latency, hotword leakage rate, installed release identity, every provider's behavior or correctness of unexercised error paths. Source comments that disagree with the traced call path are identified rather than treated as execution evidence.
- The grouped UI and atomic multi-alias edit require focused domain/store tests, both interface locales and native UI acceptance. Canonical recall needs alias/canonical hits, repeated aliases, overlapping matches, unrelated text, shared final/previous-source deduplication and the combined cap. Passing source review is not a substitute for those checks.

## Implemented validation

The selected grouped editor, atomic alias addition and canonical recall changes are implemented. See [vocabulary alignment results](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/vocabulary-alignment-v1/README.md`) for controlled baseline comparison, real local model observations, native macOS UI evidence and explicit limits.

## ASR hotword alignment

The subsequent ASR change adds recognition-time context independently of the eight-term rewrite recall. Qwen3-ASR exposes `context` as system-side recognition context in its official [inference implementation](https://github.com/QwenLM/Qwen3-ASR/blob/main/qwen_asr/inference/qwen3_asr.py). It is a hint, not a deterministic replacement. The pinned MLX and speech.cpp mappings and capability checks are documented in [ADR 020](../decisions/020-native-qwen-asr.md).

| Concern | Type4Me inspected source | Arion implemented alignment |
| --- | --- | --- |
| Correct-only terms | Separate hotword JSON file | Explicit hotword records in the existing encrypted vocabulary file |
| Learned corrections | Accepted learning saves corrected text and pair across two stores | Existing immediate learning with undo; canonical term eligibility and pair share one atomic encrypted write |
| Manual correction pairs | Do not automatically add hotwords | Remain independent; Settings supports explicit correct-only hotwords |
| Removal | Independent stores | Removing a learned pair removes its derived eligibility; a separately saved explicit hotword survives |
| Model context | Space-joined complete list, refreshed through service restart | Bounded canonical snapshot per recording, sent with every preview/final acoustic request; empty context explicitly resets the request |
| Limits | No shared Qwen3 selection budget found | Product bounds of 64 terms / 2,048 code points plus model token-position validation; these are not official Qwen3 hotword count limits |

Neither system promotes incorrect spellings into recognition hints. Arion selects correct terms before any transcript exists, so this is bounded memory selection, not transcript-based fuzzy retrieval. Later rule replacement and canonical rewrite retrieval retain their own behavior. See MEM-07 and SPH-11/12.

The [paired ASR hotword evaluation](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/results/asr-hotwords-v1/README.md`) records actual local MLX inference, unrelated-context controls, silence, per-case outputs and context-reset observations. It evaluates an exposed development vocabulary and does not establish held-out personalization gains or Windows inference quality. The [native runtime smoke archive](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/reproduction/asr-hotwords-v1/runtime-smoke/README.md`) separately verifies request handling and the model context bound.
