# ADR 047: Clipboard file access guidance

Status: Accepted. User-approved extension of [ADR 046](046-clipboard-file-access.md) and [CLP-09](../../specs/010-clipboard.md).

## Context

Background file collection can trigger folder-access requests that a user does not expect. Per-location explanatory dialogs add interruptions and duplicate a choice that can be made once in Clipboard Settings. A system refusal is persistent; Arion cannot promise to show the system request again.

## Decision

Use a separate **Record copied files** preference and one explicit enable confirmation. The [workflow](../project/clipboard.md#file-access-guidance) owns defaults, persistence and visibility. Explain that file references support later selection/pasting and that access to their containing folders may be required. Confirmation enables collection, not OS access. Cancel and failed persistence leave it off. Do not open per-folder Arion dialogs or retain location acknowledgement, cooldown or mute state.

When off, reject native file-list providers and inspect URI offers without bookmarks before any text/image fallback, hide stored files and block stale delivery without deleting retained ciphertext. Direct image bytes and text remain independent. When on, retain scoped native publication, original-path validation and target/cancellation guarantees from ADR 046. Native permission denial remains a visible settings notice with explicit system-navigation help. Never reset TCC, automatically broaden access or retry a paste. Keep missing/provider errors and encrypted-history storage failures separate from source-file permission evidence.

Derive localized Documents, Desktop and Downloads system purpose strings from the existing English/Simplified Chinese catalogs. File contents are not uploaded or backed up by this workflow.

## Verification

CLP-A08 covers preference migration, confirmation/cancellation/persistence failure, pre-read file exclusion, hidden history and stale delivery. Native permission prompts and receiving-app consumption require signed target-OS acceptance; headless UI and injected provider tests do not establish them.
