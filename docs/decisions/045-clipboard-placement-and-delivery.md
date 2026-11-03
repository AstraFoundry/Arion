# ADR 045: Independent clipboard placement and delivery

Status: Accepted by the user. Amends the geometry-based delivery rule in [ADR 034](034-clipboard-history.md) and [010](../../specs/010-clipboard.md).

## Context

The clipboard picker used a successful input-position query to enable paste. A destination can receive system paste while its provider exposes no standard text role, caret bounds or field rectangle. Geometry failure therefore selected Copy even when delivery was possible. A trusted-process WeChat reproduction with the caret in its composer returned `kAXErrorNoValue` from both application and system-wide focus queries while the original AX window was available and secure input was off. Missing AX focus therefore does not establish that the application cannot receive paste. Voice shortcuts already use system paste without an editable-focus gate ([ADR 003](003-insertion-routes.md)); their current-caret destination policy is distinct from the picker's captured destination.

## Decision

Use `Anchor()` only for optional picker placement. Prefer input geometry and otherwise use the pointer. Independently call the captured target's identity/security check to choose Paste or Copy; row clicks, Enter and the return icon share that result. Missing geometry cannot grant or revoke paste eligibility.

Capture the destination before the picker receives keyboard focus. After closing the picker, activate its captured application and revalidate the destination before writing the clipboard and again before posting one paste. Do not adopt a different foreground application, window or captured field. Cancellation and stale selection checks remain in force, with no retry after a posted or uncertain result.

On macOS retain the focused AX element and, when available, its AX window. Validate both captured identities when present. The focus handle need not expose a standard text role or geometry to remain a verifiable destination. When capture explicitly reports `kAXErrorNoValue` or `kAXErrorAttributeUnsupported` and supplies the original AX window, that exact window can authorize one system paste without a focused-element handle. Revalidate the same application/window and all available secure-input signals before writing and before posting. A newly exposed field in that window remains subject to secure-input checks. A provider timeout, failed/denied query, missing window or captured secure-input observation never grants this route, even if later queries improve. Existing captured-field identity must still match when it was available. There is no WeChat-specific exception. This changes history-picker delivery only; dictation-history capture eligibility and internal-write exclusion remain governed by CLP-03.

On Windows retain the existing native window, focused control and UI Automation identity/password/read-only checks. A verified editable destination can paste even without a usable caret rectangle, field bounds or monitor-DPI result.

If initial destination validation fails, keep browsing and Copy available. A destination change or safety failure after choosing Paste reports failure without copying as a second transport or redirecting to a new target. Geometry, focus and security queries do not read editor contents.

## Consequences and verification

The picker can appear beside the mouse while still offering Paste. A posted system paste is an attempt, not proof of host consumption or a reliable observation of typing state. With window-only identity, an application without an active editor may still receive a paste command; the selected resource remains on the system clipboard. This policy cannot simultaneously guarantee automatic paste into opaque composers and copy-only behavior whenever no real input is focused.

Bootstrap regressions cover independent geometry/eligibility combinations and revalidation before clipboard writes. Owned native macOS metadata fixtures cover exact field/window identity, window-only delivery for explicit missing/unsupported focus, absent destination identity, permissions, provider errors and protected input without reading a real app or dispatching keys. Existing UI action-mode tests cover row/Enter/search-Enter/return-icon routing. Windows execution and actual WeChat host consumption require separate actual-OS acceptance.
