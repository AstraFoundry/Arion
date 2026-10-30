# ADR 037: Keyboard context defaults and developer diagnostics

## Status

Accepted. Updates the context defaults and diagnostic policy in [ADR 036](036-two-stage-keyboard-ranking.md).

## Decision

Keyboard caret and current-window context default on for new preferences and absent fields. Neural ranking remains opt-in; no context is read while it is off. Existing explicit false values remain false. Both switches remain independently controllable, use the existing macOS Accessibility permission, and retain safe-target, activation, marked-text and cancellation checks. No OCR or screen-recording permission is added. Windows still uses recent confirmed text until its adapter supports verified target context.

Detailed keyboard diagnostics require session-only developer mode. In About, ten clicks on the version within five seconds enable it and reveal a switch to disable it. Restart always disables it; serialized preferences cannot enable it. Ordinary users retain configuration and other support diagnostics without per-keystroke ranking traces. Existing logs are retained according to the normal rotation policy.

Native eligibility skips carry a closed reason, a bounded eligible-candidate count and the existing activation/revision envelope over the authenticated bridge. The recorder discards the envelope and never stores input, candidate text or IDs, window titles, context, model output or target identities. Bridge traces distinguish stale revisions, superseded results, freeze, malformed requests, disabled ranking and transport delivery. Model observations distinguish abstention without context from an unchanged model result and retain stable error codes. Context failures identify capture, safe-target validation, read, marked-text mismatch or bounds. Delivery of a proposal is not proof the native menu applied it.

Continuous multiword input has no special exclusion. Ranking requires at least two candidates with the same consumed-input range as the first candidate, among the bounded native snapshot. A composed full-input candidate alongside shorter prefix words can leave only one eligible candidate. This records `no_comparable_candidates` and performs no inference. The ranking models select existing engine IDs; they do not synthesize a replacement sentence. This change observes that constraint rather than altering commit coverage.

## Validation

Native Rime/AppKit fixtures cover multiword eligibility; bridge tests reject stale, foreign and malformed skip events. Recorder/export tests enforce closed metadata. Application and UI tests cover default preferences, preservation of saved false values, session-only developer mode, the hidden gesture and disabling diagnostics. Windows native execution and installed-editor context acceptance require their actual hosts.
