# ADR 041: Comparative keyboard candidates

## Status

Accepted for implementation; model quality and installed-editor acceptance require separate measurements. Updates [ADR 038](038-contextual-keyboard-completion.md).

## Decision

A phonetic match authorizes a candidate to compete, not to take first place. The settled adapter proposes at most one whole-input phrase, independently validates pronunciation, then compares all eligible engine candidates and the distinct proposal with the same approved context. A shared local completion interface serves both tasks; no product cloud route is introduced. The generator sees existing alternatives. The selector receives preceding, following and bounded window text, prioritizes the caret sentence, and returns only a complete eligible-ID permutation. It must retain input order when evidence is inconclusive. These are model instructions, not a calibrated confidence guarantee. No model-written confidence is used as a score.

Local selection decoding constrains ID choices, shape and count. Independent validation rejects missing, invented or repeated IDs, extra fields, truncation and cancellation. The comparison shares the existing settled deadline with generation. An invalid comparison is an explicit failure, never permission to publish an unreviewed generated phrase. A generated proposal is inserted only when selected first; otherwise it is discarded and the selected original order applies. Duplicate generated text competes as its existing engine identity.

The `keyboard-ordering-v1` host capability authorizes arbitrary comparable-ID permutations and a completion accompanied by an original-candidate permutation. The bridge owns negotiation; an incoming request cannot enable it. Older hosts receive stable single-ID promotion or a completion without a permutation. The native host maps IDs back to the original snapshot, preserves noncomparable slots, and inserts an accepted proposal ahead of the selected original order. Navigation, activation and revision checks remain mandatory. A provisional proposal still never commits itself or enters ordinary dictionary learning.

The per-keystroke lightweight model remains a separate lane. Its existing trusted-leader policy is not evidence that a candidate was scored or that its quality is optimal. This change does not invent Rime probabilities from source names or mix incomparable model scores. Future numeric fusion requires actual producer evidence and measured calibration.

## Validation

RNK-A01, RNK-A03, RNK-A06 and RNK-A09 cover permutation validation, capability negotiation, consumed-input ranges, cancellation and native engine identity. Native fixtures test full original ordering followed by generated insertion and user-selected commit. Authored private studies compare preservation, rescue, harmful promotion, abstention, malformed results and latency through actual local and explicitly authorized research providers. Reference labels never enter requests. Component inference does not establish installed-editor Accessibility behavior or population accuracy.
