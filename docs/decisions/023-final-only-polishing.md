# ADR 023: Final-only polishing

## Status

Accepted (product decision). Supersedes the mid-speech text-model preview of [ADR 011](011-live-local-rewriting.md). Segment closure and the final rewrite remain. [ADR 022](022-cloud-polishing.md) was already final-only; local polishing now uses the same timing.

## Context

While the user was speaking, each closed recognizer segment was sent to the rewrite model and a polished preview was shown. Release then ran another rewrite and discarded that preview. Cloud polishing already sent one request after stop. The preview during speech did not become the inserted text.

## Decision

- Recognition may still close a segment after quiet or the segment limit. That text stays the recognized text. The open segment follows it.
- The session does not send closed segments to the rewrite model during speech, and it does not show a polished preview.
- When the dictation key is released, one final rewrite runs if polishing is on. Its text is the full recognized source, not a polished replacement.
- Local and cloud polishing use this timing. The polish switch still turns that rewrite on or off. Punctuation and scene formatting still apply to it.

## Consequences

- One model request per dictation. The overlay shows recognized text until release, then the validated rewrite.
- Historical live-preview measurements describe the earlier pipeline. They are not a measurement of this timing.

## Alternatives considered

- **Keep local previews and skip them only for cloud.** The preview is discarded by the final pass, so it does not change what is inserted.
- **Stop closing segments during speech.** Segmentation still bounds recognition. Only the text-model handoff during speech is removed.

## References

- [009 Contextual rewriting](../../specs/009-contextual-rewrite.md), [ADR 011](011-live-local-rewriting.md), [ADR 022](022-cloud-polishing.md).
