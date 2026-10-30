# ADR 034: Local clipboard history

Status: Accepted by the user for P0, excluding application ignore lists. Amends the menu-bar recovery surface in [ADR 011](011-live-local-rewriting.md); recognition and delivered text remain distinct in the active session.

## Decision

Implement [010](../../specs/010-clipboard.md) in portable Go domain/application code, native OS clipboard adapters, and MyGo FSD views. Ordinary file resources remain references; folders are excluded. Image bytes have no required source file. Clipboard collection is separately opt-in and never grants polishing/context/sync access.

Persist one bounded authenticated encrypted snapshot using AES-256-GCM and a distinct OS-kept `clipboard-key`. Reuse the encrypted-file primitive with vocabulary rather than duplicate encryption/atomic-write logic. No new database dependency or plaintext image cache is introduced. Clipboard authentication/read failure stops writes until an explicit retry succeeds; no key replacement or automatic reset.

Internal writes carry native revision identity and a clipboard marker. Polling never classifies origin by content. Transactions serialize clipboard access and mark only the revisions they actually write, including restoration; a newer external write remains eligible. History resources use explicit system copy/paste, separately from dictation target ownership. The panel captures an external destination before taking focus, validates restoration and posts at most once. No second delivery transport follows failure.

Settings expose one collection switch with subordinate shortcut/count controls. The native shortcut recorder is shared with dictation and translation. The input-anchored compact picker provides optional search, hover-only pins, context-sensitive Enter/return-icon/row activation (paste into a verified input, otherwise copy), compact file icons and extension/count tags, with inline image thumbnails and passive hover previews for hidden text, file names and images. The 270-DIP-wide picker shows at most four rows; preview flyouts do not activate or intercept input. Missing input geometry opens the picker beside the pointer in copy-only mode; it does not block browsing history. The macOS picker uses a nonactivating fullscreen-capable panel so activation does not move the user to Arion’s desktop Space. Delete/clear, separate full-preview and history-management pages are excluded; count/storage limits perform cleanup, with the oldest unpinned records evicted first. There is no age policy or periodic expiry task. Storage-byte limits remain application-managed. See the [settings comparison](../research/clipboard-settings.md).

Input-target dictation remains excluded and retains system clipboard restoration. A positive native `NoInput` observation plus successful final copying may explicitly record that result while history is enabled; permission uncertainty and delivery fallbacks cannot authorize recording. The menu-bar last-dictation block is removed.

## Consequences

Whole-snapshot encryption trades write cost for a small dependency-free bounded store; disk work is serialized off the main thread. All sensitive metadata stays inside the ciphertext. Search/preview operate on local in-memory values. Pins cannot bypass the total space limit. Capture baselines prevent importing the pre-enable clipboard. Native compatibility and Windows execution require actual OS checks.
