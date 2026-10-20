# ADR 003: Shortcut-mode insertion routes

Status: Accepted; macOS implemented. Windows behavior unchanged.

## Context

DSK-06 requires target/selection revalidation and one commit attempt. AX selected-text writability does not prove that a web renderer implements insertion. Field ancestry, application names and framework names cannot establish a reliable universal editing transport. The [competitor investigation](../research/competitor-text-insertion.md) distinguishes an owned IMK client from external application delivery; its static findings are evidence, not external-app acceptance.

## Decision

macOS system-IME mode commits through its existing owned IMK client. macOS global-shortcut mode uses one guarded system-paste transaction for every external target. Route selection is automatic at session start from the current input source; Settings exposes neither mode nor transport choices. There is no AX write, Unicode-event route, application allowlist, ancestor-based route selection, probing write or automatic second transport. AX remains available for target capture, focus/selection validation and consent-gated reading.

A captured opaque target belongs to one session/generation and is consumed by one matching commit. A stale identity cannot consume another session's target. Commit carries the session's cancellation context. Before preparing, staging and posting, the adapter validates cancellation, the same frontmost process and focused element, secure input, and the captured selected range when AX exposes it. Without a readable range, only process/element focus can be checked. The final native post repeats target/selection/secure and clipboard checks. OS focus/clipboard APIs provide no atomic validation-and-write primitive; these checks reduce races without claiming to eliminate them.

Transactions serialize access to the system clipboard across target-provider instances. Each owns its complete snapshot (all items and data types), text, unique ownership marker and preallocated Command-V events. Snapshot/read or staging errors stop before posting. A changed clipboard before posting stops the attempt. Cleanup restores the snapshot only while its change count and marker still belong to Arion; another application's newer clipboard is preserved. Snapshot failures and restoration failures have explicit typed errors. Clipboard contents and target text are never diagnostics.

Command-V is posted once. A 400 ms hold gives the target time to consume the board; this interval is not an acknowledgement and cannot guarantee arbitrarily delayed consumption. Cancellation after posting cannot undo delivery or shorten clipboard retention; cleanup still runs once. No fallback or retry follows a posted/uncertain result. Text is preserved exactly, including Unicode and line breaks; Arion does not synthesize Return. Hosts can apply their own paste policies, so terminal and rich-editor compatibility still needs actual host testing.

## Commit outcome

A posted paste returns `unknown` and is labeled "Pasted", never inferred `applied` from a length increase or matching substring. `CLIPBOARD_RESTORE_FAILED` preserves that uncertain outcome and displays a warning to check the editor before retrying. A pre-post rejection returns `not_applied` with an explicit reason; text remains available for user-controlled copying. A cancelled session cannot publish a late completion or start edit learning. Windows retains its existing Unicode/default and explicit clipboard routes; old persisted delivery values remain compatible and do not change macOS shortcut transport.

## Root-cause evidence

In Chromium `147.0.7727.149` (matching the inspected Lark framework's version), [`setAccessibilitySelectedText:`](https://github.com/chromium/chromium/blob/147.0.7727.149/ui/accessibility/platform/ax_platform_node_cocoa.mm#L3463) dispatches `kReplaceSelectedText` and discards the action's boolean result. The web node's [`BrowserAccessibility::AccessibilityPerformAction`](https://github.com/chromium/chromium/blob/147.0.7727.149/ui/accessibility/platform/browser_accessibility.cc#L1284) implements `kSetValue` but not `kReplaceSelectedText`; its default returns false. The renderer also lists replacement as an [unreachable action](https://github.com/chromium/chromium/blob/147.0.7727.149/content/renderer/accessibility/render_accessibility_impl.cc#L400). This explains why replacing the whole AX value and inserting through selected text have different behavior. These are upstream sources, not proof that a vendor's binary is unmodified.

Lark's inspected chat editor is under `AXWebArea`. A temporary Computer Use AX value-write draft was cleared without sending; it does not exercise Arion's selected-text or Unicode commit. Historical diagnostics omit target identity and route, so an `unknown` outcome cannot be attributed to Lark or to a particular API. App/framework classification tests alone do not reproduce an insertion failure.

## Verification

Isolated adapter tests cover cancellation at transaction boundaries, serialization/queue cancellation, clipboard failures, unknown outcomes and identity/duplicate rejection. Native transaction tests use real named pasteboards and real CGEvent construction, intercepting delivery: multi-item/binary restoration, empty originals, ownership changes, secure/focus/selection refusal and one post. They do not establish external-editor delivery.

`TestGuardedSystemPaste` runs the production adapter against a separate owned AppKit/WebKit application with actual Command-V dispatch and editor input observations. It covers exact mixed Unicode/multiline selection replacement, duplicate commit, focus and range changes, cancellation, stale identity, secure fields and clipboard restoration. See [E2E](../quality/e2e.md) for the opt-in command and evidence limits. An installed Chrome is also tested with an isolated temporary profile and local textarea/contenteditable fixtures. Actual Lark, Codex, office and terminal acceptance remains separate.

The former input-mode selector and delivery controls are removed. The current Arion input source selects IMK for a new session; other sources select external-editor delivery. This choice is fixed for the session, with no automatic retry through another route after failure. Legacy stored input modes are retired; internal transport preferences remain readable for compatibility.
