# ADR 040: Bounded keyboard context

## Status

Accepted. Supersedes the complete-window keyboard admission policy in [ADR 036](036-two-stage-keyboard-ranking.md) and [ADR 038](038-contextual-keyboard-completion.md). Dictation retains [CTX-02](../../specs/009-contextual-rewrite.md#rewrite-context).

## Decision

After safe Accessibility acquisition and exact marked-text validation, keyboard inference selects at most 128 UTF-16 units from the end of preceding caret text and the start of following caret text. Selection preserves Unicode scalar boundaries, so it can retain fewer units rather than split a surrogate pair. Shared context sanitization runs before selection. When cursor context is off, retain and bound the native host's independently confirmed preceding context instead of replacing it with an empty AX field. Existing explicit context preferences remain authoritative.

Window context remains independently enabled. From a successful complete shared AX snapshot, retain its last 1,024 UTF-16 units in AX traversal order. This deterministic bounded slice avoids scraping a larger document or adding heuristic phrase retrieval. AX order is not guaranteed to represent recency or proximity to the focused field; useful context earlier in that order may be excluded. This is Arion's explicit screen-selection policy, not a recovered competitor algorithm. A selected window produces the content-free developer reason `context_window_selected`; selection is a successful context operation, not a read error.

The keyboard no longer rejects a safe snapshot merely because its complete window exceeds the old byte admission bound. Existing transport validation and the worker's complete-template token check remain independently enforced. Source selection does not guarantee every possible Unicode request fits the model or that a model changes the first candidate. Runtime capacity and settled deadline are unchanged.

Reuse the shared AX reader and its hard traversal/text limits. Failed, partial, secure, changed-target or canceled reads still fail explicitly; do not replace them with a cached window or a different application's text. No OCR, cloud routing, model substitution or new permission is introduced. Focus/revision changes and user navigation still invalidate proposals. Voice context is not sliced by this policy.

## Evidence and validation

The private [Doubao context investigation](https://github.com/AstraFoundry/Arion-evaluation) (`benchmarks/runs/doubao-keyboard-context-v1`) records caller-supplied nearby-context limits, a preceding-text suffix operation and separate network context fields in version 1.0.3. These are verified static instruction windows, not recovered source or observed runtime behavior; neither its server capacity nor all request paths are known.

Domain tests cover side direction, UTF-16/surrogate boundaries, screen selection and unchanged dictation context. Bootstrap tests reproduce an oversized safe window reaching the model boundary while retaining privacy checks and native confirmed context. The opt-in actual-worker smoke adds a synthetic large-window completion within the production deadline. Diagnostic export tests retain the closed selection reason without text. Native Rime/AppKit regressions separately cover generated proposal publication and user-confirmed selection; injected protocol results are not installed-editor end-to-end inference.
