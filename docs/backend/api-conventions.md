# API conventions

The current desktop uses in-process typed Go ports, not HTTP. [004 Contracts](../../specs/004-contracts.md) owns versioned local semantics. Native worker/IME IPC requires explicit framing/version/capability checks when introduced.

The planned self-hosted relay uses authenticated versioned HTTP `/v1/sync/…`; exact wire schemas and enrollment/key flow are an implementation gate. Success responses return operation IDs, revisions/cursors and ciphertext records as appropriate. Errors use stable machine codes and a content-free message; UI maps codes to centralized copy.

Planned status mapping: invalid envelope 400; unauthorized/revoked 401/403; conflicting revision 409; expired cursor 410 with explicit resync procedure; bounded payload exceeded 413; rate limited 429; internal failure 500 without private details. Idempotent retries retain operation identity. Never retry a desktop `unknown` insertion through this network policy.

Contracts must be round-trip tested before shipping. Additive compatible fields retain v1; changing identity/conflict/crypto semantics requires a version migration and ADR. See [003 Memory sync](../../specs/003-memory-sync.md).
