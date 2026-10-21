# ADR 013: Rewrite context from the cursor, the screen and recent dictations

## Status

Accepted (product decision). Amends the consent rule of [009](../../specs/009-contextual-rewrite.md) step 2 and PRV-02 of [005](../../specs/005-privacy.md) for these three sources. Extends [ADR 011](011-live-local-rewriting.md) and [ADR 012](012-context-scene-selection.md). [ADR 018](018-staged-gguf-rewriting.md) keeps this context on the selector only; the formatter receives the edited source.

## Context

Typeless sends the focused window's visible text (from its accessibility tree), the text around the cursor and app/page identity to its server with the audio. Doubao conditions its on-device speech-LLM on cursor text, recent dialogue and hot words. Chatterfly sends cursor text and the window title ([competitor research](../research/competitor-rewrite.md#screen-and-cursor-context)). Their purpose is the same: spell names, terms and codes that recognition gets wrong, and continue the text already in the field.

Arion's rewrite saw only the dictation. 009 required a separate opt-in for screen text and history, and PRV-02 excluded broad screen capture.

## Decision

- **Sources.** Three sources, each its own switch under Settings › Model, all on by default while polishing is on:
  - Text around the cursor: up to 200 characters on each side, read through Accessibility.
  - Screen content: visible text from the captured window through macOS Accessibility or Windows UI Automation. This user-requested amendment replaces product screenshot/OCR collection. Screen Recording is no longer needed.
  - Recent dictations: the last five inserted in the same site or app, kept for 30 minutes in memory.
- **Bounds and exclusions.**
  - No window image is captured. Traversal is bounded to 512 nodes, depth 32, 32,768 UTF-16 units and a 1.5 s deadline; limit failures discard the result. Text areas use visible ranges, and secure/hidden/offscreen branches are excluded. Accessible text is retained after sanitization; the inference worker explicitly rejects requests exceeding its token budget. Context lives in session memory and is never logged or synced.
  - Secure fields read nothing, and a built-in list of password managers is never read.
  - Turning history off clears it.
- **Data, not instructions.** Context is sanitized of prompt markers and tags, placed in its own sections before the dictation, and described to the model as reference data. The validator accepts spellings supported by the context but rejects:
  - proposals that add a long run of context content;
  - context that stands in for most of the dictation (`context_copied`, capped coverage credit).
- **No screen VLM.** A local vision-language model was not evaluated; reading the screen goes through the OS recognizer.
- **Windows.** UI Automation is implemented but compile-checked only; text around the cursor is not read on Windows.

## Consequences

- Everything stays on the device. Unlike Typeless, no context reaches a server, and PRV-01 (offline, no content egress) is unchanged.
- **Delivery as context sections.** At the user's request, both preview and final rewrite requests now deliver sanitized accessible window text plus cursor/history as separate data sections. `context-v2+live-v2` permits plausible ordinary-word disambiguation from the current discussion while preserving intent and questions. Approved vocabulary remains in terms. This supersedes the earlier terms-only delivery and 1,200-character OCR cutoff; long requests fail the worker token budget explicitly. Historical benchmark results below describe the earlier templates, not an accuracy or latency guarantee for this change.
- **Measured effect** ([report](../../benchmarks/results/context-ablation-v1/README.md)): with Qwen3 4B, all sources raise the held-out context set from 9 to 16 of 30 with no worsened case, almost entirely from screen text. The shipped Qwen3 1.7B ignores context (9 of 30 with or without it). The benefit therefore needs the larger model; with 1.7B the context costs recognition time and changes little.
- Historical OCR collection cost a capture plus about 0.1–0.3 s of recognition. Current Accessibility collection runs in parallel with speaking; the final rewrite still waits at most 1.5 s for context. See the [paired diagnostic](../../benchmarks/reproduction/qwen17-accessibility-v1/README.md) for current measured scope.
- On by default means a person who never opens Settings gets screen and field text read whenever polishing is on. The source switches and Accessibility grant are the controls. A missing Accessibility provider returns an explicit unsupported result, without OCR fallback.

## Alternatives considered

- **Opt-in per source** (009 as written): safer by default, but most people would never benefit; the user chose Typeless's default.
- **Accessibility-tree visible text** (Typeless): now selected by the user. Missing text providers and large trees remain explicit coverage/limit failures.
- **Local VLM on the screenshot:** not run; weights were not downloaded for this study.
- **Context as ASR hot words** (Doubao): needs a biasing-capable recognizer and audio fixtures; a candidate follow-up.
- **Earlier context sections** (`context-v1`): the same 16 held-out passes as terms for 4B, but slower at the median (1.84 s against 1.33 s in the same batch) and a prefix swap per switch. This historical result motivated the earlier terms-only policy; the current decision prioritizes retaining discussion semantics.
