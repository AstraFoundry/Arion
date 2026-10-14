# 003 Personal memory and self-hosted sync

## Memory

MEM-01: Allowed sync records are user-approved vocabulary entries, explicit correction pairs and per-app style preferences. A record has opaque ID, kind, content schema version, logical revision, origin device, approval provenance and tombstone. Temporary app/selection context is session-only and excluded from persistence/sync.

MEM-02: Learning adds a record only from the user's own correction of text Arion inserted ([ADR 014](../docs/decisions/014-edit-learning.md)). The record applies at once and is shown with an undo; undoing it, or correcting a learned replacement back, removes it. Model guesses, typing that continues or precedes the insertion, deletions, punctuation, changed numbers, sentence-start capitalization, lowered letter case, single Latin letters, grammar-dependent homophones (他/她/它, 的/地/得, 在/再) and rewrites of most of the insertion are never learned. The user can review, remove, export and disable learning/sync independently. Vocabulary may influence ranking only within its approved scope.

MEM-04: Memory is bounded. Learned entries unused for 90 days expire and, beyond 2,000 entries, the least used learned entries are evicted; entries the user typed never are. A learned pair whose From matches an entry the user typed leaves that entry unchanged. Recall applies approved entries as whole-word replacements in one pass, and only the entries relevant to the current text, at most eight, reach the rewrite prompt.

MEM-05: Learning never delays dictation. Following a field starts after the commit returns, each field read has its own time budget, a field that stops answering is abandoned, and the next dictation settles the previous edit without waiting. Memory writes are atomic and durable; a file that fails authentication or cannot be written is left unchanged, learning reports the failure, and dictation continues with whatever memory loaded.

MEM-03: Local writes and outbox creation occur in one transaction. Dictation never waits for network or sync. When sync is disabled, local memory remains available and no outbox uploads occur.

## Relay protocol design

SYNC-01: A self-hosted Go service stores only authenticated ciphertext envelopes and necessary routing metadata. The client encrypts before persistence into the outbox. Auth credentials and encryption keys are separate and neither is stored in repository config.

SYNC-02: Enrollment uses an existing authorized device or a recovery secret to establish group keys. A device has a revocable identity. Crypto library, pairing UX and recovery format are gated decisions; do not implement homemade key exchange. Device registration does not depend on an account: the hosted relay adds OIDC account authorization on top of device authentication, and self-hosted relays may leave it disabled ([ADR 015](../docs/decisions/015-hosted-relay-account-authorization.md)).

SYNC-03: Planned operations are authenticated `POST /v1/sync/push`, `GET /v1/sync/pull?cursor=…`, device enrollment and revoke. Push batches use bounded size and unique operation IDs; retries deduplicate by device and operation ID and return the original result. Pull cursors are opaque, server-ordered and distinct from logical memory revisions.

SYNC-04: Envelopes carry opaque group/record/device/operation IDs, schema version, key epoch, algorithm identifier, nonce, ciphertext, integrity tag and expected server revision. Associated metadata must be authenticated by the client encryption scheme. The relay validates device authentication, shape and revision policy; only authorized recipient clients can validate the E2EE tag and plaintext schema. Exact binary encoding and limits require a protocol fixture before adapter implementation.

SYNC-05: A concurrent edit produces explicit sibling revisions, never silent last-writer-wins based on wall clock. Delete/edit conflicts keep the record suppressed until reviewed. Tombstones are retained until all active devices acknowledge them; revoked/stale devices require full resync. Resolution creates a new descendant revision that is synchronized like a normal write.

SYNC-06: Unknown schema, invalid authentication, revoked device, stale key epoch, invalid tag, replay and cursor expiry are explicit outcomes. Cursor expiry triggers a validated full snapshot path, not clearing local memory. Uncertain push results reuse the same operation ID.

SYNC-07: Revocation rotates the key epoch for future writes and re-encrypts live records before new-device access. Previously downloaded plaintext cannot be remotely erased. Export/deletion/backup procedures disclose this boundary. Sync-disabled and server-deletion are separate visible actions.

## Account authorization

SYNC-08: An `AccountGate` application port resolves each request to a tenant before the relay core runs; the core never reads account data. `noop` (self-host default) uses one implicit tenant and device authentication only. `oidc` requires a Bearer JWT whose signature verifies against the configured issuer's JWKS and whose `iss`, `aud` and `exp` are valid and whose `ent` contains the configured entitlement (`arion.sync.hosted` for the hosted relay), and maps `(issuer, subject)` to a tenant (`account_link` unique on `(issuer, subject)`; `tenant(id, quota, state)`). Device authentication is still required inside the tenant. A missing entitlement, invalid token or frozen tenant is an explicit outcome, never a fallback to `noop`. An account never grants key access: losing every device without the recovery secret makes synced memory unrecoverable.

SYNC-09: With `oidc`, `POST /v1/security-events` accepts RFC 8417 Security Event Tokens pushed per RFC 8935, verified with the issuer's JWKS, `aud` set to the relay's server client and deduplicated by `jti`; it answers `202` or `400` with an RFC 8935 error. `account-purged` deletes the tenant's ciphertext, routing metadata, devices and account link; `account-disabled` freezes the tenant; `account-enabled` unfreezes it; `session-revoked` rejects that subject's tokens issued before the event until the client re-authorizes.

SYNC-10: Onboarding defaults to continuing locally without an account; signing in to set up sync is the second path. Desktop sign-in uses the system browser, a loopback `127.0.0.1` redirect on a random port and PKCE S256 (RFC 8252); the refresh token is kept only in the OS credential store. Signing out keeps local keys and memory. A signed-in user without the hosted entitlement keeps every local feature and may join the waitlist or use a self-hosted relay.

## Acceptance

| ID | Scenario |
| --- | --- |
| MEM-A01 | Offline edit → restart → outbox survives → reconnect → each operation applied once |
| MEM-A02 | Two devices concurrently edit → siblings visible and require resolution |
| MEM-A03 | Delete on A/edit on B → old content cannot silently reappear |
| MEM-A04 | Sync off → capture, ranking and local memory remain usable; no uploads |
| MEM-A05 | Inspect relay database/logs → no plaintext memory/audio/window context |
| MEM-A06 | Revoke B → B cannot pull/push; new key epoch works for active devices |
| MEM-A07 | Invalid tag/key epoch/unknown schema → explicit failure, no plaintext downgrade |
| MEM-A08 | Tampered routing metadata or replayed envelope → authentication/replay check rejects |
| MEM-A09 | Correct a dictated word in a field → the pair is learned, shown with an undo, and corrects the next dictation; undo or correcting it back removes it (`TestExtractCorrections`, `TestLearnRules`, `TestLearnedNoticeUndo`) |
| MEM-A10 | Continued typing, deletions, punctuation, changed numbers or a rewrite → nothing learned ([edit-learning-v1](../benchmarks/results/edit-learning-v1/README.md)) |
| MEM-A11 | Secure field, learning off, IME composition or Windows → the field is never followed or read (`TestCoordinatorNeverFollowsSensitiveOrWhenOff`) |
| MEM-A12 | Hung host, corrupt/truncated/wrong-key file, read-only directory or crash leftover → dictation unaffected, memory file unchanged or explicitly unavailable (`TestEditTrackerAbandonsHungHost`, `TestCorruptFileFailsClosed`, edit-learning-v1 faults) |
| MEM-A13 | `noop` relay → enroll, push and pull with device authentication only; no token, account table or events endpoint |
| MEM-A14 | `oidc` relay → wrong signature, issuer, audience, expired token or missing entitlement → explicit rejection; a valid token without a device signature, or with another tenant's device, cannot push or pull |
| MEM-A15 | Same `(issuer, subject)` → same tenant; another issuer with the same subject → a different tenant |
| MEM-A16 | Security events: purge → tenant ciphertext, routing metadata, devices and link gone; disable → push/pull rejected with data kept; enable → restored; session revoked → earlier tokens rejected, a newer token accepted; replayed `jti` → no second effect; bad signature, issuer or audience → `400` |
| MEM-A17 | Sign out → refresh token removed from the credential store; local keys, memory and dictation unaffected |

Deployment implementation is pending; see [handoff](../docs/project/agent-handoff.md).
