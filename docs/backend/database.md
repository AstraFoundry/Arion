# Persistence

Local SQLite with encrypted sensitive values is a proposed adapter, not an implemented database. The concrete driver, migration runner and encryption provider require review before MEM-01. Encrypt content before it reaches SQL/WAL/backups; OS-kept keys and schema metadata have distinct trust boundaries.

Allowed local records: approved memory, user preferences, tombstones, encrypted outbox and noncontent model metadata. Audio and temporary context are RAM-only by default. Dictation history requires separate opt-in and is not part of initial sync.

IDs are opaque stable identifiers. Domain revision/parent references determine conflicts; wall-clock timestamps do not pick winners. Repository adapters implement transactions for local memory + outbox. Migrations are versioned, tested against prior snapshots and never modify applied revisions.

The future relay stores ciphertext envelopes, device authorization, operation deduplication and server cursors. Do not assume an ORM, PostgreSQL or broker; SQLite is an initial deployment candidate subject to concurrency/backup evaluation. See [memory sync](../../specs/003-memory-sync.md).
