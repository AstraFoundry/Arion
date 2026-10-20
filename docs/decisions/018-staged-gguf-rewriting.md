# 018: Staged GGUF rewriting on macOS Apple Silicon

## Status

Accepted for product integration by the user. This replaces the ONNX/live-v2 rewrite implementation of ADR 011; ASR selection, once-only insertion and session ownership are unchanged. Hardware/release evidence remains scoped to actual measurements.

## Decision

Use the pinned Qwen3.5-2B Q4_K_M artifact and llama.cpp b11734 on macOS Apple Silicon. Retain the pipe worker contract, with a private Unix-socket native child and process-group cleanup. The packaged runtime contains no weights; existing verified provisioning installs the catalog artifact for offline inference.

Use the exact baseline selector. The original integration adopted work-scope-v4 formatting; the user-requested conservative alignment replaces it with natural-layout-v2. Source-relative pronunciation hypotheses are unverified; the model selects keep/replace under a bounded ID schema, then the formatter receives only the edited source. Preserve work/email/coding-prompt/chat scenes and route language instructions/examples by actual source script.

Remove production semantic rejection based on overlap, wording, numbers, negation counts and OCR credit. The user accepted the baseline's three harmful choices and six misses in the exposed 101-case development set, rather than requiring case-specific rules or a perfect score. Keep protocol/UTF-8/completeness/control-markup checks, explicit failure outcomes, recognized-original availability and cancellation/focus/source-revision ownership. This is an accepted product tradeoff, not a guarantee of semantic correctness or a held-out error-rate estimate.

## Consequences

- The independent speech ONNX runtime and optional rewriting remain separate capabilities.
- Work and prompt scenes no longer collapse into chat/list before model invocation.
- Fixed pypinyin/CMU resources ship with provenance and licenses; the Go producer and payload construction are checked against frozen experimental fixtures.
- Preparation checks native template/token budgets but does not compute KV state or sample tokens. The native slot is erased at session boundaries; save/restore APIs are never called.
- Retired research weights can be removed while preserving raw observations, source hashes and reproduction artifacts. Retired inference requires downloading the pinned weights again.
- Actual integration uses M4/32GB. Apple Silicon/16GB, supported older macOS versions, concurrent speech load and native editor delivery remain distinct acceptance scopes. Intel/Windows have no selected runtime.

See [local rewrite configuration](../operations/local-rewrite.md), [contextual rewriting](../../specs/009-contextual-rewrite.md), and [product integration evidence](../../benchmarks/reproduction/rewrite-product-integration-v1/README.md).

## Conservative formatting follow-up

The user requested alignment with Doubao and Chatterfly after observing excessive structuring. Preserve the separate cleanup, spelling-selection and formatting stages, and recognized-original availability. Default to minimal copyediting and natural prose in work/prompt scenes. Multiple tasks or restrictions alone do not imply list intent. Lists require clear enumeration, an explicit list-layout request or the explicit list scene. Remove demonstrations that add unsolicited bullets or category labels; retain explicit-list demonstrations with owner/deadline/condition scope.

This is Arion's policy choice informed by client workflow evidence. Neither competitor's complete server prompt or list-formatting threshold was recovered. Do not describe this template as a copied or behaviorally equivalent competitor prompt. See [focused verification](../../benchmarks/reproduction/rewrite-natural-layout-v1/README.md).

## Numeric and hesitation follow-up

The formatter preserves numeric value and precision rather than requiring spoken Chinese numeral spelling. Natural-layout-v2 adds shared script-routed editing examples for quantities, decimal points, signs, percentages and sentence-internal hesitation. Preserve meaningful referents, times, idioms and literal quotations. This is LLM formatting, not a global numeral replacement rule or a guarantee of recovery from acoustic errors. See [focused observations](../../benchmarks/reproduction/rewrite-numeric-filler-v1/README.md).

## Technical spelling follow-up

Natural-layout-v3 permits source-grounded canonical English spellings for established phonetic technical terms in an unambiguous technical context. Keep surrounding Chinese, proper names, literal text and explicit Chinese naming. This changes the formatter policy and script guidance, not the frozen selector or its pronunciation producer. See [scoped native observations](../../benchmarks/reproduction/rewrite-technical-spelling-v1/README.md).

## Preview-to-final spelling follow-up

The stage-preservation investigation reproduced correct LLM previews followed by an incorrect final result on the same synthetic audio: the whole-utterance acoustic finalizer changed the source spelling and relevant memory disappeared. The selected source pipeline adds exact-letter compound nominations, independent-context-confirmed earlier-recognition nominations, bounded approved-memory continuity and exact-request preview reuse. Selector instructions and natural-layout-v3 generation policy remain unchanged. Earlier recognition is not generated model output or a new persistent memory source. A fixed-gap formatter trial was rejected after actual duplication/protocol failures; no hard entity lock or competitor-equivalent server policy is claimed. See [protocol and evidence](../../benchmarks/reproduction/rewrite-stage-preservation-v1/README.md) and [competitor boundaries](../research/rewrite-stage-preservation.md).
