# 002 Local speech and correction

## Pipeline contract

Capture → bounded frames → explicit resampling → VAD/streaming recognizer → revisioned draft → conservative correction → recognized preview or owned IME composition → final commit. Keyboard candidate ranking is a separate reorder of producer candidates (SPH-07).

- SPH-01: PCM frames carry session/generation, increasing sequence, monotonic sample offset, rate/channels and discontinuity status. The chosen adapter declares supported audio format; model metadata declares its sample rate. Do not assume every model wants the same format.
- SPH-02: Capture callbacks MUST only enqueue frames. On overload, emit a discontinuity and terminate or restart explicitly; do not conceal dropped audio.
- SPH-03: Partial output replaces only its session-owned draft. A revision number increases for every hypothesis update. ASR endpoint/final status is separate from target commit status.
- SPH-04: VAD boundaries MUST preserve complete speech including onset/tail and flush the recognizer on user stop. A nonstreaming recognizer in overlapping windows must be identified as chunked, not true streaming.
- SPH-04a: When closed segments are processed during capture ([009 LRW-02](009-contextual-rewrite.md#live-rewriting-pipeline)), each segment's audio is cut inside the pause that closed it, so no syllable is split between segments.
- SPH-05: Correction input MUST include exact source revision, allowed contextual memory and a bounded draft. Correction results cannot apply to a different revision or committed content.
- SPH-06: Names, numbers, URLs, negation and technical identifiers MUST retain intent. Acronym joining preserves quoted wording and leaves ambiguous repeated first-person `I` separate. Standalone hesitation (`嗯`, `呃`, `额`, `um`, `uh`, `erm`, `uhm`) is not a correction rule. When polishing is on, `domain.CleanSpeech` deletes those tokens before organization. Polishing may also remove a recognized stutter in that stage. Ambiguous rewrites are presented as candidates; reject model additions outside supported correction policy.
- SPH-07: Candidate ranking MUST only reorder producer-owned candidates. Models producing only one hypothesis must not be described as exposing n-best. See [007](007-ime-ranking.md).
- SPH-08: Offline operation after provisioning MUST include capture, ASR, ranking and correction without server reachability. Unavailable optional correction leaves raw/rule-corrected text visibly identified.
- SPH-09: Silence MUST not create committed hallucinated text. Session audio is discarded when processing ends by default.
- SPH-10: Vendor recognition envelopes are transport metadata, never dictated text. Qwen3-ASR language headers and `<asr_text>` delimiters are parsed at the ASR adapter boundary, including leading whitespace. Plain text and literal markers within a transcription remain unchanged.

## Acceptance

| ID | Required evidence |
| --- | --- |
| SPH-A01 | Two recognizer revisions arrive out of order → older revision ignored |
| SPH-A02 | Ranker/correction finishes after cancel → no UI update, learning or insertion |
| SPH-A03 | Mandarin/English mixed names, numbers and negation → compare ASR vs corrected truth; regression reported |
| SPH-A04 | Bounded queue overflows → explicit audio discontinuity, no seamless-success claim |
| SPH-A05 | Stop mid-syllable → flush outcome shown; no duplicate segment commit |
| SPH-A06 | Network disabled after provisioning → local full pipeline still works |
| SPH-A07 | Silence/noise-only fixture → no candidate auto-committed |
| SPH-A08 | Missing correction model → explicit capability status; no network inference |

The runtime and checkpoint evaluation shortlist is in [research](../docs/research/on-device-speech.md). No weights are chosen in this foundation.
