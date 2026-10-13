# ADR 008: Evaluate contextual rewriting as a separate local capability

## Status

Proposed. This records the investigation boundary and experiment design, not an adopted runtime, checkpoint or enabled product feature.

## Context

ASR reconstructs speech; candidate ranking chooses among producer-owned alternatives; conservative correction repairs bounded mistakes. Scenario-aware rewriting changes expression and may introduce semantic errors even when the result sounds fluent. The installed competitors expose post-recognition refinement and scene/context surfaces, but their server models and prompts are not available from client artifacts.

Arion needs a local implementation that preserves the speaker's meaning, input ownership and privacy. A generative token stream is not evidence of incremental speech understanding, safe text replacement or suitable latency on a particular device.

## Proposed design

- Keep verbatim dictation, deterministic correction and optional contextual rewriting as separate policies. Rewriting produces a reviewable candidate; it never fabricates ASR n-best or bypasses the commit coordinator.
- Select a declared scene (generic, chat, email, document or technical) from an explicit user preference and consent-gated app/field metadata. App identity alone cannot establish intent; uncertain classification keeps the declared generic policy visible.
- Snapshot only bounded, opted-in field context and approved memory. Foreign context is data, not an instruction channel. No automatic clipboard, full-screen content or conversation scraping is introduced by this research.
- Separate final rewriting after stop from any future speculative draft rewriting. Speculation uses immutable source revisions, cancels superseded generations, has a bounded work queue and never writes to a foreign target. The first implementation gate is final-candidate evaluation; speculative scheduling requires its own streaming experiment.
- Build prompts from versioned policy and the checkpoint's pinned chat template. Use explicit non-thinking inference where supported; pin tokenizer, stop tokens, context/output limits and conversion settings. Budget overflow is an explicit rejected capability/result, not silent truncation.
- Validate identity, decoding completion, structural output and declared preservation constraints before preview. Names, numbers, URLs, negation and technical identifiers require both automatic proxies and blind human intent review. A rejected candidate preserves the separately identified source/rule result and does not silently switch to cloud inference.
- Compare existing ONNX Runtime GenAI CPU packages against reference inference before selecting a native C/C++ bridge. Accelerator providers and model exports are separate treatments. Verify compatibility with sherpa's resident ONNX Runtime; an isolated worker is an option if shared-library or crash isolation requires it, with authenticated bounded local IPC specified before implementation.

## Consequences

Rewriting has separate model provisioning and capability status. Recognition remains usable while the optional rewriting model is missing, rejected or cancelled. Model selection requires quality, native preview latency, memory and concurrent-ASR evidence on each declared OS/hardware class; ASR bundle rankings do not select an LLM.

The evaluator requires real token/preview/lifecycle observations and reviewed labels. Automatic overlap and protected-string metrics remain diagnostic proxies. A human review import is required before claiming meaning preservation or style quality. Missing runtime drivers and missing human ratings are reported as missing evidence.

## Alternatives considered

- A single undifferentiated correction stage hides semantic changes and makes ablation and failure ownership ambiguous.
- Running a large reasoning model for every ASR partial adds unbounded work and unstable output; a token stream alone cannot justify this schedule.
- Copying competitor prompts or server endpoints does not provide a reproducible local model and is outside the implementation plan.
- Extending Laya to generate rewritten text conflates its candidate selection contract with generation.

## References

- [Contextual rewrite spec](../../specs/009-contextual-rewrite.md).
- [Chatterfly shipped pipeline](../research/chatterfly.md).
- [Typeless and Doubao rewrite paths](../research/competitor-rewrite.md).
- [ONNX/Hugging Face model candidates](../research/onnx-rewrite-models.md).
- [Rewrite evaluation](../research/rewrite-evaluation.md).
