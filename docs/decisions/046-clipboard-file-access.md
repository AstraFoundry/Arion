# ADR 046: Clipboard file access

Status: Accepted. Extends the file-reference contract in [010](../../specs/010-clipboard.md) and [ADR 034](034-clipboard-history.md).

## Context

A valid filesystem path is not necessarily authority to read a file. macOS can attach a security-scoped URL to a Finder clipboard copy while permitting metadata inspection of an otherwise protected path. Keeping only the path discards the authorization needed to publish that file later. The system can then omit `public.file-url` while retaining generic URL representations. Local pasteboard caches and generic URL readers do not establish successful external file publication.

## Decision

Decode incoming file URLs through AppKit. Preserve a read-only security-scoped bookmark per file while the incoming scope is active, with one matching stop for each successful start. Store these opaque grants beside their ordered paths inside the existing encrypted history snapshot; include their bytes in retention and per-item limits. They do not participate in resource identity. A fresh copy updates available grants without changing the record's ID or pin; a path-only recopy preserves an existing grant.

On delivery, resolve the bookmark without UI, mounting or implicit access. Reject invalid/stale grants, failed scope activation and a resolved path that differs from the original path after symlink normalization. Do not follow a moved file. Hold scopes only through native publication/flush, then release them. A legacy path-only reference must have sufficient current access to create a scope; failure is visible and never automatically selects a broader permission prompt or an unscoped retry. [ADR 047](047-clipboard-file-access-guidance.md) adds a separate file-history opt-in and enable confirmation.

MyGo carries bookmarks as private typed-transfer metadata. It never advertises or writes those bytes to a native clipboard, and rejects them for drag publication and non-macOS clipboard writes. File URLs themselves are ready before native publication so the server can establish receiving-process access. All history items retain the internal origin marker. Native scope preparation fails before replacing the clipboard; source-file contents are never copied into history or changed.

Dictation clipboard snapshots separately retain incoming file URL scopes through their bounded staging/restoration transaction, releasing them on completion, failure or ownership loss. They do not require persistent bookmarks.

The existing encrypted snapshot is upgraded in place without discarding old records; [clipboard persistence](../project/clipboard.md#ownership-and-persistence) owns its schema version. Native delivery requests neither full-directory nor full-disk grants. Collection may reach a system folder-access prompt only after the explicit file-history opt-in in ADR 047. Bookmarks remain local credentials, excluded from diagnostics, UI previews, search and synchronization.

## Verification

Portable tests cover identity/pin preservation, access refresh, byte limits and aligned filtering. Persistence tests cover old records, encrypted grants and restart. Native tests capture a Finder-style file reference, resolve its serialized grant in a fresh subprocess, publish only file URLs, and reject moved, mismatched and corrupt references. Named-pasteboard tests cannot establish actual privacy-service grants or WeChat consumption; those require receiving-app acceptance on the signed development build.

Apple documents [security-scoped URLs and balanced access](https://developer.apple.com/documentation/foundation/nsurl) and [persistent file access](https://developer.apple.com/documentation/security/accessing-files-from-the-macos-app-sandbox).
