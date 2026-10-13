# ADR 003: Shortcut-mode insertion routes

Status: Accepted; macOS implemented, Windows adapter compiled.

## Context

DSK-06 requires a commit that revalidates the target and inserts once; an unsupported target or uncertain result must not trigger a second automatic attempt, and clipboard use must be visible (architecture: input ownership). Chromium and Electron editors report a settable `AXSelectedText` attribute yet ignore the write, so "try Accessibility, then something else" would be a hidden second attempt.

## Decision

The macOS adapter chooses **one route when dictation starts**, from declared capabilities, and the commit never switches routes:

1. **Accessibility** when the focused element exposes a settable selected-text attribute and a readable value, and the host is not a Chromium/Electron editor (known bundle IDs or an embedded `Electron Framework.framework`).
2. **Keystrokes** otherwise: the text is posted as Unicode keyboard events in short composed-character chunks. Line breaks become spaces: Return is never synthesized, so Arion cannot send a message or run a terminal command.
3. **Paste** only when the user selects it in settings: Arion saves every clipboard item, writes the text, posts Command-V, and restores the saved items only if the clipboard change count is still Arion's own (DSK-A05).

Before the single attempt the adapter requires the same focused element (or the same frontmost process when no element is exposed) and rejects secure input. When the field value is readable, a length increase after the attempt reports `applied`; otherwise the outcome is `unknown`, which the UI labels "Typed" or "Pasted" rather than "Inserted", and which is never retried (CTR-A06). A focus change yields `not_applied/TARGET_CHANGED` and the text stays available to copy from the menu bar.

Known secure fields (`AXSecureTextField` or secure event input) block the session before the microphone opens. Unknown field safety disables context reading but not insertion (DSK-08).

## Consequences

Electron and browser editors receive keystrokes, which IMEs or key remappers in the target app can still observe. Verification is best-effort where the value is unreadable. Windows uses `SendInput` Unicode events with foreground-window revalidation; UI Automation value-pattern verification is a follow-up.

## Alternatives considered

- Clipboard paste everywhere (Typeless-style): simplest, but touches user data on every dictation.
- Accessibility first with keystroke fallback: violates the single-attempt rule when the first write silently fails.
