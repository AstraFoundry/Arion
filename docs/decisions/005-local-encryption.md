# ADR 005: Local encryption of approved vocabulary

Status: Accepted for local storage; sync key exchange still pending (SYNC-01).

## Context

[005 Privacy](../../specs/005-privacy.md) requires approved vocabulary to be stored as encrypted local values with OS-kept keys, failing closed on key or tag errors, and forbids homemade cryptography.

## Decision

Vocabulary entries are serialized and sealed with AES-256-GCM from the Go standard library (`crypto/aes`, `crypto/cipher`) with a random 96-bit nonce per write and a fixed associated-data label. The 256-bit key is generated with `crypto/rand` and kept in the login Keychain (`SecItemAdd`/`SecItemCopyMatching`, accessible after first unlock, this device only); Windows uses DPAPI. Any Keychain or authentication failure is an error: there is no plaintext or on-disk key fallback. Files are written via a temporary file and rename.

## Consequences

Losing the Keychain item makes the vocabulary unreadable, by design (PRV-A06). Ad hoc development builds may trigger a Keychain access prompt after re-signing. Group keys, device enrollment and key epochs for sync are not covered and remain gated.

## Alternatives considered

SQLite with full-database encryption (no vetted pure-Go option without new dependencies) and storing the key next to the data (defeats the purpose).
