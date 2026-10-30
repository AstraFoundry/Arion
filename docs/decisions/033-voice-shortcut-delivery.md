# ADR 033: Current-caret delivery for global voice shortcuts

Status: Accepted. Amends shortcut entry-point routing in [ADR 003](003-insertion-routes.md).

## Context

A globally observed speech shortcut can be pressed while no input field is focused. Selecting Arion as the OS input source does not establish an editable IMK or TSF client. An IMK commit acknowledgement confirms that the host received the command; calling the client's void insertion interface cannot prove that text appeared in a field. Routing a global shortcut through that client can therefore display Inserted without leaving the result on the clipboard.

## Decision

Both global voice keys call `Coordinator.StartShortcut`. An empty translation target starts ordinary dictation with the user's polishing policy; a nonempty target forces translation as before. This entry point selects external-editor delivery before preparing the session and does not consult the source-selected IME route.

The existing external adapter inserts at the current editable caret, including an empty field, moved caret or newly focused field. No editable target leaves the exact final processed result on the clipboard with `copied`. Sensitive fields stay blocked. A source change or unrelated IME deactivation cannot cancel this external session. Cancellation, generation checks, single commit and no retry after a posted or uncertain result remain unchanged.

The input menu, tray and in-app toggle retain source-selected delivery and owned composition. Keyboard conversion and its native ownership rules are unchanged. This decision adds no setting and does not retry a failed IME commit through another route.

## Verification

The bootstrap regression drives both shortcut handlers while the configured route selector reports system IME. It requires an external session for both keys. Core tests require the processed polishing or translation result to reach the clipboard target once, and keep the external session alive through IME deactivation. Native adapter tests and manual installed-app acceptance remain separate from these routing checks.
