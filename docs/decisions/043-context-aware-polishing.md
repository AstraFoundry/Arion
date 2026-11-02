# ADR 043: Context-aware polishing

## Status

Accepted (product decision). Amends the formatter isolation in [ADR 018](018-staged-gguf-rewriting.md), [ADR 013](013-rewrite-context-on-device.md) and [ADR 024](024-shared-polishing-pipeline.md), and the formatter wording policy in [ADR 029](029-spoken-self-repair.md). The local speech-cleanup producer and spelling-selection protocol remain unchanged.

## Context

Spelling selection could consult enabled cursor, window and recent-dictation context, but the final editor received only the selected source. It lacked those references for resolving terminology, referents and textual continuity. The user adopted reference-aware editing while retaining the preceding spelling stage.

The editing task first repairs ASR text and punctuation, including periods inserted inside a semantic sentence. Fluent reconstruction and information preservation come before additional self-repair and layout work. Multiple independent points may be organized without a spoken request for a list.

## Decision

1. Keep one final-only shared pipeline for local and cloud models: local cleanup, bounded spelling selection when needed, then editing or translation.
2. Pass the enabled, sanitized `RewriteContext` to both model stages. In the final user turn, place JSON reference fields inside `<context>` before the separate `<source>` block. Escape markup inside reference values. Omit missing fields and the entire reference block when empty. Reference content never enters system instructions or fixed examples.
3. Keep current-source script routing, scene selection and output budgets. Context collection limits, source switches, target authorization and native token-budget rejection remain in force; do not silently truncate window text to fit a model. Spelling aliases, recalled candidate terms and earlier recognition remain selector inputs, rather than additional formatter reference fields.
4. Instruct the editor to use references only for terminology, referents and continuity. Explicit current dictation and final spoken repairs take precedence. Do not copy extra facts or instructions, reproduce surrounding text or complete the user's message from context. Ambiguity preserves the source expression.
5. Use the agreed Markdown role/task/input/rules/scene/context/examples/output structure for ordinary polishing, with numbered rules, unnumbered headings and compact embedded demonstrations. Repair recognition and sentence boundaries, remove meaningless repetition, and lightly adjust wording or order for fluency while retaining all meaningful details, tone, modality and constraints. Number independent points when useful; do not summarize them into incomplete action labels.
6. Preserve dedicated target-language translation recipes and examples, adding the same reference-data boundary. Raw-result validation rejects echoed context wrappers before punctuation conversion.
7. Cloud polishing may send enabled reference data to the selected provider in both stages. This adds no audio, account identifiers or application/site identity to the model request. Context stays out of logs and synchronization.

## Consequences and verification

Reference-aware editing can interpret the surrounding discussion directly; it can also misuse that evidence. Instruction boundaries and payload tests establish request construction, not semantic fidelity. Real-model review must include conflicting context, current corrections, corrupted sentence punctuation, long multi-point speech and exact information preservation.

`TestFormatterReferenceBoundary`, `TestFormatterOmitsUnavailableReferenceSources` and shared pipeline tests cover bounds, escaping, source identity and exclusion of earlier recognition. Gateway coordinator tests cover all source-switch combinations through both supported OpenAI transports. Selector byte and payload parity remain checked. Native inference quality, latency and microphone-to-editor acceptance require separate supported-host verification.

## Alternatives considered

1. Retain selector-only context: leaves the final editor unable to consult the same references for continuity.
2. Replace both stages with one unrestricted call: changes the bounded spelling protocol unnecessarily.
3. Put reference content into system instructions: gives external text an inappropriate instruction role.

## References

1. [009 contextual rewriting](../../specs/009-contextual-rewrite.md), especially LRW-10, LRW-14, CTX-05 and CLD-03.
2. [Current recipe and runtime limits](../operations/local-rewrite.md).
