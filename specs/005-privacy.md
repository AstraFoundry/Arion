# 005 Privacy and trust boundaries

| Data | Default lifetime/location | Sync/log policy |
| --- | --- | --- |
| Raw audio/PCM | Bounded RAM for active session | Never sync/log; disk recording requires separate user action |
| ASR/correction drafts | Active session RAM | Never diagnostics; history separately opt-in |
| Target app/nearby/selected text | Bounded consent-gated session context | Never sync/log or persist automatically |
| Approved vocabulary/correction/preferences | Encrypted local values, including pairs learned from the user's corrections with their use counts | Client-encrypted sync only when enabled |
| Usage counts | Per-day numbers in `usage.json`: sessions, speaking time, words and words per rewrite scene; no text, app, host or time of day | Never sync/log; shown only in Settings › Overview |
| Group/device keys and auth tokens | OS credential store | Never logs/config/export without explicit protected flow |
| Models | Per-user verified model cache | Public artifact downloads only; no speech payload |
| App updates | Signed archive replaces the installed app | Release manifest and archive downloads from GitHub releases only; no content payload; checks can be turned off |
| Performance diagnostics | Local content-free metrics | No remote telemetry enabled by this foundation |

PRV-01: Dictation and ranking have no cloud inference route. Model downloads, encrypted sync (self-hosted or hosted) and account sign-in ([ADR 015](../docs/decisions/015-hosted-relay-account-authorization.md)) are separately enabled surfaces; test their destinations and payload classes.

PRV-07: Update checks request only the public release manifest and archive of the configured GitHub repository. They carry no audio, text, vocabulary, preference or device identifier, run only while the user's automatic-check preference is on or on an explicit request, and an archive whose Ed25519 signature does not match the built-in key is never installed ([ADR 010](../docs/decisions/010-in-app-updates.md)).

PRV-02: Context read is limited to the focused application/field and bounded text under each source's switch ([ADR 013](../docs/decisions/013-rewrite-context-on-device.md)): text around the cursor, OS-recognized text of the focused window only (the image is never stored) and recent dictations in memory. No full-screen or background capture, keystroke history or full document scraping. Secure/unknown fields disable reads/learning. User denylist overrides per-app preferences. While learning from corrections is on ([ADR 014](../docs/decisions/014-edit-learning.md)), the field Arion inserted into is re-read, as a bounded excerpt around its cursor, until the user stops editing it (at most 60 s); only the learned word pairs are kept, encrypted with the vocabulary.

PRV-03: Before using personal memory, disclose its type/scope and allow reset/export/delete. Session-only memory is never promoted automatically. Disabling sync does not imply deletion of already stored relay ciphertext; expose deletion separately.

PRV-04: A vetted authenticated encryption provider must fail closed on key/tag errors. OS key stores protect local keys; encryption protects persisted sensitive values including SQLite WAL/backups by encrypting before SQL writes. Full database encryption versus field encryption is an implementation decision that must account for metadata leakage.

PRV-05: Log only random correlation IDs, coarse capability/model IDs, numeric durations, queue depth and error codes. Exclude audio/transcripts, vocabulary, app window titles, selected text, clipboard, filesystem user paths, prompts and secrets. Failure bundles are opt-in and locally inspectable.

PRV-06: A relay operator, self-hosted or hosted, can observe timing, IP/device/group routing metadata and ciphertext sizes; the hosted relay also knows which account subject owns a tenant, never its keys. E2EE does not conceal all metadata or protect against a compromised endpoint. Document backup, key recovery, device revocation and deletion limits before enabling sync.

PRV-08: Usage counts record a session only after its commit completes. They hold calendar-day totals keyed by date and by scene name, never dictated text, bundle IDs, hosts or per-session timestamps, so they cannot reconstruct what was said or where.

## Acceptance

PRV-A01: offline capture/rank/correction generates no content network traffic; PRV-A02: artifact/log scan excludes synthetic sensitive canaries; PRV-A03: cancel/restart leaves no temporary PCM files; PRV-A04: relay backup contains ciphertext only; PRV-A05: denylist/password/unknown field blocks context; PRV-A06: lost keys cannot silently create plaintext data or replace old encrypted records; PRV-A07: with automatic checks off, or in the offline profile, the app makes no update request, and a tampered update archive is rejected. PRV-A08: a cancelled, silent or failed session adds no usage, and the usage file holds only dates, counts and scene names (`TestUsageMeterCountsCommittedSessions`, `TestUsageRoundTrip`).
