# ADR 026: Spoken self-repair collapses before polish

## Status

Accepted. Amends the formatter policy in [ADR 018](018-staged-gguf-rewriting.md) from `cleanup-rules-v1` to `cleanup-rules-v2`, and the deletion producer in [ADR 011](011-live-local-rewriting.md) from `speech-spans-v2` to `speech-spans-v3`.

## Context

A speaker can revise mid-utterance: 「十二月…十三月…什么十三，没有十三…十一月」. The committed text kept the abandoned months and the repair talk.

Two checks were made:

- Commit timing. [ADR 023](023-final-only-polishing.md) still holds. Closed segments stay on screen during speech. Release runs one rewrite of the full recognized text. Partials are not inserted early.
- The text that reaches polish. `speech-spans-v2` deleted a repair only after a narrow cue (`不对`, `no wait`, `I mean`) plus a repeated frame or a matching unit. `不是 A，是 B`, `什么X`, and `没有X` were not cues. Months were not a typed unit. The formatter was also told to keep explicit negations, so 「没有十三」 looked like content to preserve, and to make minimal edits. The model therefore stayed close to the raw transcript. `ValidateRewriteDelivery` does not reject that.

The same pipeline serves macOS and Windows, local and cloud ([ADR 024](024-shared-polishing-pipeline.md)). Cloud polish already sends text, not microphone audio.

A second complaint: polish paraphrased words that were already recognized correctly. The speaker wants the original vocabulary, with three allowed changes: an ASR correction when the intended word is clear, a self-repair collapsed to the final span, and light reorganization of messy spoken order.

## Decision

- `speech-spans-v3` deletes a clear revision before the formatter sees the text. A same-type run (month, quantity, number, English month or weekday) collapses to the last value when the gaps are only separators and repair talk. `不是 A，是 B` and `not A, it's B` collapse when A repeats the preceding attempt, or when A and B are the same typed unit. A copular replacement must fill the rest of its clause.
- These stay: `四秒，不是四十秒`, `我不是张三，是李四`, conjunctions, quoted text, unfinished repair talk, negation, and uncertainty.
- `cleanup-rules-v2` states the same rule for cases the span producer leaves, and forbids synonym paraphrase. An ASR mishear fix is named a correction. The selector wire token stays `replace`. That token is the frozen baseline schema, not a license to swap synonyms.
- Both engines use this path. No new audio leaves the device.

## Consequences

- The month example, a corrected count, and `不是 A，是 B` are removed locally even when the model is unavailable. The inserted fallback is the cleaned text.
- The formatter can still paraphrase words the span producer kept. There is no synonym dictionary and no new semantic veto. The prompt and examples are the control for that case.
- Historical reports that name `speech-spans-v2` or `cleanup-rules-v1` describe the producer at the time of those runs.

## Alternatives considered

- Prompt-only. A 2B model that is told to preserve every quantity does not reliably drop 「没有十三」. The span deletion does not depend on that model.
- Re-enable overlap rejection in `ValidateRewriteDelivery`. That also blocks legitimate digit normalization, scene layout, and ASR corrections.
- Rename the selector token from `replace` to `correction`. The frozen selector baseline and JSON schema require `replace`.

## References

- [009 LRW-09, LRW-10, LRW-A14](../../specs/009-contextual-rewrite.md), [ADR 023](023-final-only-polishing.md), [ADR 024](024-shared-polishing-pipeline.md).
