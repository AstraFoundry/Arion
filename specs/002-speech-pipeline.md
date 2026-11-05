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
- SPH-09: Silence MUST not create committed hallucinated text. A recognition window is quiet when no 20 ms span, or the whole buffer when it is shorter, reaches -42 dBFS RMS. Quiet windows are not submitted to the recognizer, so system-turn hotwords cannot be copied into the draft before speech, during a quiet tail, or on a fully silent recording. A window that contains speech still receives the session hotword snapshot, including its leading and trailing quiet samples. An ordered copy of that snapshot is replaced before it is shown (SPH-13). Session audio is discarded when processing ends by default.
- SPH-10: Vendor recognition envelopes are transport metadata, never dictated text. Qwen3-ASR language headers and `<asr_text>` delimiters are parsed at the ASR adapter boundary, including leading whitespace. Plain text and literal markers within a transcription remain unchanged.

- SPH-11: Each recording snapshots bounded, approved canonical ASR hotwords before recognizer creation. Preview, segment acoustic refinement and full-audio finalization use that same snapshot; memory changes affect the next recording. Wrong aliases and generated rewrite text never enter ASR context. Selection follows MEM-07. Context is a recognition hint, not a forced substitution or an accuracy guarantee.
- SPH-12: Native ASR receives a per-request space-separated `prompt`, including an explicit empty string to clear context. Nonempty context requires an advertised prompt capability; unsupported workers fail explicitly. Product term/rune bounds and the native model's token context bound are independently enforced. No cloud inference is introduced.
- SPH-13: A speech window is still submitted with its snapshot. Before that transcript is published, an ordered copy of two or more contiguous snapshot terms is discarded: the whole transcript, a leading run, or a trailing run. Punctuation and a leading `气声` label do not make the copy publishable. The same samples are decoded again with an empty prompt, and only that result is published. One matching term is kept. The energy floor is not raised, so a breathy window that crosses it is still submitted.

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
| SPH-A09 | Mutate memory during capture → preview and all acoustic finalizers retain the initial snapshot; the next session receives updated terms |
| SPH-A10 | Prompt-capable speech.cpp protocol fixtures receive exact canonical context; empty requests clear it; unsupported workers and invalid terms fail explicitly |
| SPH-A11 | Real local Qwen3 paired audio evaluation retains raw outputs, fixture/model/runtime hashes, unrelated-word and silence controls, accuracy and latency; report regressions and platform limitations |
| SPH-A12 | Quiet windows with hotwords stay empty and are not submitted; energy at the floor, a short loud span, negative polarity, a quiet tail, a hold longer than the segment limit, and speech after leading silence keep the snapshot (`TestQuietAudioDoesNotEmitHotwords`, `TestSpeechFloorWindows`) |
| SPH-A13 | An ordered hotword copy, including a `气声` label, a leading run and a trailing run, is replaced by an empty-prompt decode of the same samples and is not published; one term, a reversed pair, a non-contiguous pair and a run inside a sentence stay (`TestHotwordEchoShapes`, `TestHotwordEchoRedecode`) |

The runtime and checkpoint evaluation shortlist is in [research](../docs/research/on-device-speech.md). No weights are chosen in this foundation.
