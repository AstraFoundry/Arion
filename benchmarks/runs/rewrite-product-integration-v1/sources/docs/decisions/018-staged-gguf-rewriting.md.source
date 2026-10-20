# 018: Staged GGUF rewriting on macOS Apple Silicon

## Status

Accepted for product integration by the user. This replaces the ONNX/live-v2 rewrite implementation of ADR 011; ASR selection, once-only insertion and session ownership are unchanged. Hardware/release evidence remains scoped to actual measurements.

## Decision

Use the pinned Qwen3.5-2B Q4_K_M artifact and llama.cpp b11734 on macOS Apple Silicon. Retain the pipe worker contract, with a private Unix-socket native child and process-group cleanup. The packaged runtime contains no weights; existing verified provisioning installs the catalog artifact for offline inference.

Use the exact baseline selector and work-scope-v4 formatter templates. Source-relative pronunciation hypotheses are unverified; the model selects keep/replace under a bounded ID schema, then the formatter receives only the edited source. Preserve work/email/coding-prompt/chat scenes and route language instructions/examples by actual source script.

Remove production semantic rejection based on overlap, wording, numbers, negation counts and OCR credit. The user accepted the baseline's three harmful choices and six misses in the exposed 101-case development set, rather than requiring case-specific rules or a perfect score. Keep protocol/UTF-8/completeness/control-markup checks, explicit failure outcomes, recognized-original availability and cancellation/focus/source-revision ownership. This is an accepted product tradeoff, not a guarantee of semantic correctness or a held-out error-rate estimate.

## Consequences

- The independent speech ONNX runtime and optional rewriting remain separate capabilities.
- Work and prompt scenes no longer collapse into chat/list before model invocation.
- Fixed pypinyin/CMU resources ship with provenance and licenses; the Go producer and payload construction are checked against frozen experimental fixtures.
- Preparation checks native template/token budgets but does not compute KV state or sample tokens. The native slot is erased at session boundaries; save/restore APIs are never called.
- Retired research weights can be removed while preserving raw observations, source hashes and reproduction artifacts. Retired inference requires downloading the pinned weights again.
- Actual integration uses M4/32GB. Apple Silicon/16GB, supported older macOS versions, concurrent speech load and native editor delivery remain distinct acceptance scopes. Intel/Windows have no selected runtime.

See [local rewrite configuration](../operations/local-rewrite.md), [contextual rewriting](../../specs/009-contextual-rewrite.md), and [product integration evidence](../../benchmarks/reproduction/rewrite-product-integration-v1/README.md).
