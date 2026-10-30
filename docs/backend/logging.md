# Logging and diagnostics

The desktop records structured English JSONL events locally in `<data directory>/logs/events.0.jsonl` (newest), with older files numbered 1–4. No remote telemetry or automatic log upload is configured. The directory is private (0700) and files/exports use 0600 on POSIX; Windows uses the user's profile access controls.

## Events and ownership

- `packages/core/domain/ports.Diagnostics` carries only pipeline stage, random session identity/generation, elapsed duration, stable failure code, queue metrics and validation verdict. Application code observes target capture, recognizer creation, audio start, context reads, acoustic finalization, live/final rewrite, validation and target commit. Observers must return immediately.
- `apps/desktop/infrastructure/diagnostics` implements a bounded 256-entry nonblocking queue and one disk worker. It projects a strict allowlist from product events; product event objects are never serialized. Session logs preserve state, sequence, elapsed time, the first draft only, correction verdict and commit outcome. A successful copied commit may include the boolean `noInput` fact, which distinguishes confirmed absence of editable input from an uncertain copy-only result; it contains no target identity or text. Audio levels, subsequent drafts and generated tokens are ignored. Queue pressure increments an explicit `events_dropped` count instead of delaying capture or inference.
- `bootstrap` records app lifecycle and trigger/cancel/suspend/lock events, model and provisioning phase changes, permission categories, hold key registration, device enumeration, edit-learning counts/verdicts and storage/update failures. Input-method setup records bounded `ime_install_started`, `ime_install_pending`, `ime_install_completed`, `ime_install_failed`, `ime_settings_open`, `ime_remove` and `ime_source_state` events; the state event uses only absent/pending/installed/selected/unsupported outcomes; pending activation is distinct from an install failure, and raw OS error text/paths remain excluded. Model IDs are checked against the embedded catalog. Raw errors and details are replaced with stable codes, including `IO_FAILED` for uncoded infrastructure errors.
- Each process has a fresh random `run` ID; session IDs are random and never derived from user/device identity. `at` is UTC wall time; `durationMs` is monotonic stage duration for pipeline events and monotonic time since session start for product session events. `count` holds repair/pair/drop counts; queue depth has its own field. Permission events name only the capability and consent state.

Do not log audio, transcripts, model prompts, candidate text, approved vocabulary, app identities/window titles/hosts, scene choices, selection/context, clipboard, user filesystem paths, auth tokens or keys. Masking transcripts is insufficient; omit the fields. The schema validates event/enum names, numeric ranges and opaque correlation IDs before writing and during export. New model IDs must belong to the embedded catalog; export preserves syntactically valid historical model IDs after a catalog update retires a model. Native SDK/ASR verbose logging and raw worker stderr are not collected into the diagnostic store or ZIP.

### macOS input-source diagnostics

`ime_native` records Arion's own fixed parent/mode IDs, requested identity, process ID, installed-directory presence, native lookup/identity-match/enable/select/selectability flags, operation step and signed numeric OSStatus (including zero). Register, enable, select and switch-away operations record before/after snapshots; installation/removal and host stop/launch/expected-build verification also report their result as a stable error code. Non-query snapshots include the installed executable SHA-256, or an explicit build-read failure code. Registration status covers the existing LaunchServices/TIS registration operation; enable status covers the existing parent-then-child operation, not separate per-call statuses.

Periodic queries record the first snapshot and changes only. The current OS source is classified as `arion_mode`, `arion_parent`, `other` or `unavailable`; foreign input-source IDs and target application identities are never retained. Native flags are process-local observations at capture time, independent of host connection readiness and editor acceptance. These logs cannot observe the failed ID inside WeChat's system HUD. No private lookup API, preference write, source selection or cache refresh is added by diagnostics. The same bounded recorder/export path validates the nested closed schema, rejects unknown fields and owns a copy of the payload before asynchronous writing.

For an independent process comparison, `arion-ime-probe -op query -diagnostics` includes the same observations in its JSON output without modifying input-source state or opening a transport listener. Omit the flag to retain the existing probe output. The regular app writes these records to its local rotating log and includes them in About's diagnostic export.

## Retention and failure behavior

Files rotate at 2 MiB or a UTC day boundary, retaining at most five files (10 MiB total). Files older than seven days are removed on startup and subsequent writes; exports also filter entries older than seven days. Logs may cover less time under heavy activity. This is a rolling support buffer, not a complete history.

Storage starts only after the app obtains its single-instance lock, when native startup becomes ready. An unavailable log directory leaves dictation usable and surfaces export failure; stderr reports only a content-free code. Write/read/sync failures remain explicit and sticky for that recorder: export fails rather than claiming to contain complete logs. A restart can reopen a repaired directory. Quit drains accepted records; an abrupt termination may lose queued entries and does not produce an automatic crash dump.

## Export

Settings › About › **Export logs…** opens a native save dialog with a ZIP filename. Cancel creates no archive. The action disables while choosing/saving, reports success or failure in the current interface language, and reveals a successful file in Finder/Explorer. The file stays local until the user shares it.

The worker flushes accepted records and snapshots only the five named diagnostic files. Export rejects symlinks, oversized/corrupt records, unknown fields and invalid values, then writes a private temporary ZIP and atomically replaces the selected destination. A failed export leaves an existing destination intact. No recursive profile scan is used. `manifest.json` contains schema/app version (omitted for unversioned development builds), OS/architecture, export time, limits and excluded data classes; JSONL entries are locally inspectable after unzipping. Preferences, usage, vocabulary, model files, raw stderr and native crash dumps are excluded. The final export-completed event is recorded locally after the exported snapshot.

## Verification

`TestExportExcludesSensitiveCanaries` exercises sensitive product fields, raw error/path canaries and unrelated profile files. Recorder tests cover size/age limits, unknown/corrupt source records, symlinks, destination preservation, concurrent producers/export/close and error reporting. Core tests verify observer stage/identity/error behavior without changing insertion. About view tests cover both languages, progress/result/error states and repeated-click suppression; controller tests cover native chooser cancellation and disk errors. Run `pnpm core:check`, `pnpm desktop:check` and `pnpm i18n:check`; use `go test -race` for the diagnostic store, controller and core application.

The macOS native chooser was exercised in an isolated test app: cancel created no archive, save produced the expected ZIP, and Finder selected it. Windows native chooser/Explorer behavior still requires a Windows host; headless rendering does not establish it. [Competitor evidence](../research/competitor-diagnostics.md) motivates the stage boundaries; [privacy requirements](../../specs/005-privacy.md) remain authoritative.

Input-method maintenance diagnostics include bounded helper PID and availability state (`unknown`, `available`, `selected`, `unavailable`) alongside own parent/child IDs and native operation statuses. `disable` and `verify_removal` are separate operations; a disable return code is not completion. Existing `ime_install_started`, `ime_install_completed` and `ime_install_failed` events also cover automatic replacements. Input text, foreign source identifiers and helper stderr are not added to exports.

## Keyboard model ranking

Detailed ranking events are recorded only in session-only developer mode: in About, click the version ten times within five seconds. The revealed switch disables it immediately; restarting also disables it. Input, context, candidates, IDs and model output remain excluded. Use the regular Export logs action after reproducing a problem. See [ADR 037](../decisions/037-keyboard-context-and-developer-diagnostics.md).

`keyboard_ranking_configuration` remains available outside developer mode. `keyboard_ranking` identifies the catalog model, candidate count, elapsed time, stable error code and outcome: `accepted` (a different first candidate proposed), `unchanged`, `no_context`, `no_comparable_candidates`, `timeout`, `cancelled`, `unavailable` or `rejected`. Model events start only after debounce reaches the ranker.

`keyboard_ranking_trace` records closed reasons without text:

| Reason | Meaning |
| --- | --- |
| `no_comparable_candidates` | Native filtering left fewer than two Han candidates covering the leader's input range; count is 0 or 1. Multiword input often has one full-range sentence and several shorter prefix words. No model ran. |
| `navigation` / `caret_not_at_end` | The menu was navigated, expanded or paged, or the caret moved inside the composition. |
| `input_bounds` / `non_chinese` | Input/context exceeded admission bounds, or the first candidate was not Chinese. |
| `freeze` / `superseded` / `stale_revision` | Composition ownership was revoked, a result lost to newer work, or an old native message arrived. Superseded work includes cancellation during debounce. |
| `ranking_disabled` / `invalid_request` | No configured ranking lane or an invalid snapshot identity. |
| `context_target` / `context_capture` / `context_read` | Target admission, capture or Accessibility read failed; the stable code distinguishes permission, unsupported and changed targets. |
| `context_marked_mismatch` / `context_bounds` | Captured caret text did not end in the owned preedit, or window context exceeded its bound. |
| `proposal_sent` / `delivery_failed` | The bridge sent the proposal or transport failed. Delivery does not prove native application. |

Native trace messages are admitted only for the verified live activation and a fresh revision. Reasons/counts are validated before recording; exports accept only the closed schema. A model can keep the first candidate, and no existing candidate can be replaced by newly generated text. Neither waiting nor enabling context guarantees a visible reorder.
