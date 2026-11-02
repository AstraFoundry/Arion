# ADR 042: Comparative keyboard candidates

## Status

Accepted for implementation; model quality and installed-editor acceptance require separate measurements. Updates [ADR 038](038-contextual-keyboard-completion.md).

## Decision

A phonetic match authorizes a candidate to compete, not to take first place. The settled adapter proposes at most one whole-input phrase and independently validates pronunciation, then compares all eligible originals and the distinct proposal through the bundled Chinese IME desktop-preset language model. Generation uses the shared local completion interface and approved bounded caret/window context. Comparison reuses the existing keyboard worker and speech-priority lifecycle, with separate resident fast and desktop scoring weights in the same process, without introducing a cloud route.

Both keyboard weight files are hash-verified from the model catalog and staged during packaging; typing never downloads them. The keyboard preset remains on the fast lane. The desktop preset handles settled comparison because the keyboard preset rejected a valid technical correction in the exposed regressions.

The scorer returns actual mean log probabilities for each alternative followed by up to sixteen Unicode characters of the approved following text, conditioned on approved preceding text. The model bounds its preceding context internally. Window text informs generation, not this small scorer. The initial development experiment with an LLM-written ranking and candidate-aware generation regressed; the final generator decodes independently to avoid anchoring to an incorrect original. Numeric scoring is a fluency and nearby-context signal, not calibrated semantic confidence or a replica of a competitor's fusion model.

Candidates are stably sorted by comparable scores. A challenger must improve on the original leader by more than 0.2 mean log-probability units to replace it; otherwise the original remains first. This development-calibrated margin is provisional, not a population guarantee. Producer probabilities are not available in the current contract and are not invented from source names. A generated proposal is inserted only when it wins this comparison; otherwise it is discarded and the original-candidate ordering applies. Matching text retains its existing identity. `NONE` declines new text; existing comparable candidates can still be scored.

The comparison shares the existing settled deadline with generation. Missing, nonfinite or incomplete scores, cancellation and worker failure remain explicit errors. They never authorize an unreviewed generated phrase. The worker's serialized bounded exchange drains canceled replies and retains its watchdog. A declined proposal has a content-free developer diagnostic.

The `keyboard-ordering-v1` host capability authorizes arbitrary comparable-ID permutations and a completion accompanied by an original-candidate permutation. The bridge owns negotiation; an incoming request cannot enable it. Older hosts receive stable single-ID promotion or a completion without a permutation. The native host maps IDs back to the original snapshot, preserves noncomparable slots, and inserts an accepted proposal ahead of the selected original order. Navigation, activation and revision checks remain mandatory. A provisional proposal still never commits itself or enters ordinary dictionary learning.

The per-keystroke lightweight model remains a separate lane. Its existing trusted-leader policy is not evidence that a candidate was scored or that its quality is optimal. This change does not invent Rime probabilities from source names or mix incomparable model scores. Future numeric fusion requires actual producer evidence and measured calibration.

## Validation

RNK-A01, RNK-A03, RNK-A06 and RNK-A09 cover permutation validation, capability negotiation, consumed-input ranges, cancellation and native engine identity. Native fixtures test full original ordering followed by generated insertion and user-selected commit. Authored private studies compare preservation, rescue, harmful promotion, abstention, malformed results and latency through actual local and explicitly authorized research providers. Reference labels never enter requests. Component inference does not establish installed-editor Accessibility behavior or population accuracy.
