# Logging and diagnostics

The desktop records structured English JSONL events locally in `<data directory>/logs/events.0.jsonl` (newest), with older files numbered 1–4. No remote telemetry or automatic log upload is configured. The directory is private (0700) and files/exports use 0600 on POSIX; Windows uses the user's profile access controls.

## Events and ownership

- `packages/core/domain/ports.Diagnostics` carries only pipeline stage, random session identity/generation, elapsed duration, stable failure code, queue metrics and validation verdict. Application code observes target capture, recognizer creation, audio start, context reads, acoustic finalization, live/final rewrite, validation and target commit. Observers must return immediately.
- `apps/desktop/infrastructure/diagnostics` implements a bounded 256-entry nonblocking queue and one disk worker. It projects a strict allowlist from product events; product event objects are never serialized. Session logs preserve state, sequence, elapsed time, the first draft only, correction verdict and commit outcome. Audio levels, subsequent drafts and generated tokens are ignored. Queue pressure increments an explicit `events_dropped` count instead of delaying capture or inference.
- `bootstrap` records app lifecycle and trigger/cancel/suspend/lock events, model and provisioning phase changes, permission categories, shortcut/hold registration, device enumeration, edit-learning counts/verdicts and storage/update failures. Model IDs are checked against the embedded catalog. Raw errors and details are replaced with stable codes, including `IO_FAILED` for uncoded infrastructure errors.
- Each process has a fresh random `run` ID; session IDs are random and never derived from user/device identity. `at` is UTC wall time; `durationMs` is monotonic stage duration for pipeline events and monotonic time since session start for product session events. `count` holds repair/pair/drop counts; queue depth has its own field. Permission events name only the capability and consent state.

Do not log audio, transcripts, model prompts, candidate text, approved vocabulary, app identities/window titles/hosts, scene choices, selection/context, clipboard, user filesystem paths, auth tokens or keys. Masking transcripts is insufficient; omit the fields. The schema validates event/enum names, numeric ranges and opaque correlation IDs before writing and during export. New model IDs must belong to the embedded catalog; export preserves syntactically valid historical model IDs after a catalog update retires a model. Native SDK/ASR verbose logging and raw worker stderr are not collected into the diagnostic store or ZIP.

## Retention and failure behavior

Files rotate at 2 MiB or a UTC day boundary, retaining at most five files (10 MiB total). Files older than seven days are removed on startup and subsequent writes; exports also filter entries older than seven days. Logs may cover less time under heavy activity. This is a rolling support buffer, not a complete history.

Storage starts only after the app obtains its single-instance lock, when native startup becomes ready. An unavailable log directory leaves dictation usable and surfaces export failure; stderr reports only a content-free code. Write/read/sync failures remain explicit and sticky for that recorder: export fails rather than claiming to contain complete logs. A restart can reopen a repaired directory. Quit drains accepted records; an abrupt termination may lose queued entries and does not produce an automatic crash dump.

## Export

Settings › About › **Export logs…** opens a native save dialog with a ZIP filename. Cancel creates no archive. The action disables while choosing/saving, reports success or failure in the current interface language, and reveals a successful file in Finder/Explorer. The file stays local until the user shares it.

The worker flushes accepted records and snapshots only the five named diagnostic files. Export rejects symlinks, oversized/corrupt records, unknown fields and invalid values, then writes a private temporary ZIP and atomically replaces the selected destination. A failed export leaves an existing destination intact. No recursive profile scan is used. `manifest.json` contains schema/app version (omitted for unversioned development builds), OS/architecture, export time, limits and excluded data classes; JSONL entries are locally inspectable after unzipping. Preferences, usage, vocabulary, model files, raw stderr and native crash dumps are excluded. The final export-completed event is recorded locally after the exported snapshot.

## Verification

`TestExportExcludesSensitiveCanaries` exercises sensitive product fields, raw error/path canaries and unrelated profile files. Recorder tests cover size/age limits, unknown/corrupt source records, symlinks, destination preservation, concurrent producers/export/close and error reporting. Core tests verify observer stage/identity/error behavior without changing insertion. About view tests cover both languages, progress/result/error states and repeated-click suppression; controller tests cover native chooser cancellation and disk errors. Run `pnpm core:check`, `pnpm desktop:check` and `pnpm i18n:check`; use `go test -race` for the diagnostic store, controller and core application.

The macOS native chooser was exercised in an isolated test app: cancel created no archive, save produced the expected ZIP, and Finder selected it. Windows native chooser/Explorer behavior still requires a Windows host; headless rendering does not establish it. [Competitor evidence](../research/competitor-diagnostics.md) motivates the stage boundaries; [privacy requirements](../../specs/005-privacy.md) remain authoritative.
