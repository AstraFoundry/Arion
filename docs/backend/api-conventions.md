# API conventions

The current desktop uses in-process typed Go ports, not HTTP. [004 Contracts](../../specs/004-contracts.md) owns versioned local semantics. Native worker/IME IPC requires explicit framing/version/capability checks when introduced.

The relay serves authenticated versioned HTTP `/v1/sync/…`. Success responses return operation outcomes, revisions/cursors and ciphertext records. Errors use stable machine codes and a fixed content-free message.

Status mapping: invalid envelope, unknown schema, invalid tag or unsupported recovery `400`; unauthorized or revoked session `401`; revoked device, scope mismatch, missing entitlement, frozen tenant or quota `403`; missing route or record `404`; conflict, stale key epoch, replay or operation mismatch `409`; expired cursor `410` with `resync` `/v1/sync/snapshot`; payload too large `413`; rate limited `429`; internal failure `500` without private details. Security events use `202` or RFC 8935 `400` JSON, not this envelope. Idempotent retries retain operation identity. Never retry a desktop `unknown` insertion through this network policy.

Contracts must be round-trip tested before shipping. Additive compatible fields retain v1; changing identity/conflict/crypto semantics requires a version migration and ADR. See [003 Memory sync](../../specs/003-memory-sync.md).
