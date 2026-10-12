# Logging and diagnostics

Use structured, English, content-free events with random correlation IDs, numeric durations, coarse model/capability IDs, queue depth and stable error codes. Logs stay local by default; no remote telemetry is configured.

Do not log audio, transcripts, model prompts, candidate text, approved vocabulary, app/window titles, selection/context, clipboard, user filesystem paths, auth tokens or keys. Masking transcripts is insufficient; omit the fields. Native SDK/ASR verbose logging must be audited before embedding.

Use monotonic durations for local latency and opaque operation IDs for sync diagnostics. Invalid key/tag/target/version errors remain explicit rather than silently caught. Diagnostic exports are opt-in and inspectable locally. See [privacy spec](../../specs/005-privacy.md).
