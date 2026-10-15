# Competitor diagnostic boundaries

This comparison reuses the repository's recorded package evidence. It does not inspect anyone's runtime logs, recordings, account data or credentials, and it does not establish current vendor defaults, retention, export behavior or runtime correctness.

| Product and recorded version | Evidence | Useful diagnostic boundaries for Arion |
| --- | --- | --- |
| Typeless 2.8.1.130 | [Installed-artifact report](typeless.md), [TL ledger](evidence/typeless-static.json): separate audio-session/chunk, processing/refinement lifecycle, native insertion and history surfaces | Correlate trigger, capture, recognition/refinement and delivery as distinct stages. Packaged lifecycle names establish instrumentation surfaces, not a verified event sequence or logging policy. |
| Doubao IME 1.0.2 | [DS-commit](evidence/doubao-static.json): `finish_vad_requested`, `finish_audio_send` and final-commit diagnostics; [DD-paste-gate, DD-paste-handoff, DD-commit-strategy](evidence/doubao-decompiled.json) | Record session/generation, stage completion, ownership/focus failures, insertion outcome and cancellation. Preserve an uncertain acknowledgement as unknown; never report it as a confirmed write or retry it automatically. |
| Chatterfly 1.0.6.13398 | [CF-asm-processor, CF-asm-sse-open, CF-asm-sse-dispatch, CF-asm-do-post](evidence/chatterfly-rewrite.json), [report](chatterfly.md): logger calls, request start timing, typed performance/error/result packets, stable/enhanced text separation | Measure request boundaries and distinguish preview, final rewrite, validation and handoff. These are client-side static observations; server timing, actual output and telemetry settings remain unverified. |

Arion adopts the stage boundaries and diagnostic categories, with its own local inference and stricter content exclusions. Logs contain random process/session IDs, generations/sequences, state transitions, first-draft timing, stage durations, model/provisioning state, bounded segment queue depth/overflow, rewrite verdicts, delivery outcomes, permission categories and persistence/update failures. Continuous draft, token and microphone-level events are not written. App identities, hosts, scene choices and all user content are omitted.

The bounded local store and explicit Settings › About ZIP export are Arion product choices, not claims that the competitors use the same retention or export mechanism. See [logging implementation](../backend/logging.md).
