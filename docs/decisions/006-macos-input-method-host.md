# ADR 006: macOS input-method host and local IPC

Status: Accepted; implemented. Install/select/remove on a real input-source list requires a person or a dedicated test profile (IME-A01).

## Context

DSK-01/IME-01 require a user-selectable system IME mode. A MyGo window is not an InputMethodKit server (ADR 001), and recognition must stay in Arion's process with its resident models and session coordinator.

## Decision

- `ArionInput.app` is a small Objective-C InputMethodKit server shipped inside `Arion.app/Contents/Resources`. It contains no recognition. It declares one input mode, `local.arion.inputmethod.voice`, and passes every key event through, so ordinary typing is unchanged (IME-04).
- **Install and removal are explicit settings actions.** Install copies the bundle to `~/Library/Input Methods`, registers and enables it with Text Input Sources; the user then selects "Arion Voice" in the input menu. Remove disables the source and deletes the bundle, and switches Arion back to shortcut mode.
- **IPC** is a per-user Unix domain socket at `~/Library/Application Support/Arion/ime.sock` (mode 0600; only the same user can connect), one JSON object per line. The IME sends `activate` (with the client bundle ID), `deactivate`, `escape` and `committed`; Arion sends `mark`, `commit` and `cancel`.
- **Ownership:** in `system_ime` mode the coordinator captures an IME target only while the source is active, mirrors every accepted draft revision into the owned marked text, commits the reviewed text once through `insertText` and waits for the IME's acknowledgement (`applied`, otherwise `unknown`). Cancellation, failure, Escape, deactivation and the host's `commitComposition` clear the owned marked text and invalidate the session; an unconfirmed draft is never committed implicitly (IME-03).

## Consequences

The IME mode works only while Arion is running. Secure fields disable input methods system-wide, so no context can be read there. Candidate windows (RNK-01) are not shown because the recognizer exposes no alternatives. Windows TSF remains a separate task; on Windows the mode is reported unsupported rather than simulated.

## Alternatives considered

Running recognition inside the IME process (duplicate model residency, two recognizer lifecycles) and XPC services (needs signed, sandbox-aware bundles not available to ad hoc development builds).
