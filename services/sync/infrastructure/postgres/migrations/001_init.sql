CREATE TABLE tenant (
    id TEXT PRIMARY KEY,
    quota BIGINT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('active', 'frozen')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE account_link (
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    tenant_id TEXT NOT NULL REFERENCES tenant (id),
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (issuer, subject)
);

CREATE TABLE subject_control (
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    frozen BOOLEAN NOT NULL,
    purged BOOLEAN NOT NULL,
    session_revoked_at TIMESTAMPTZ,
    PRIMARY KEY (issuer, subject)
);

CREATE TABLE security_event (
    jti TEXT PRIMARY KEY,
    received_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE sync_group (
    id TEXT PRIMARY KEY,
    scope_id TEXT NOT NULL,
    key_epoch BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX sync_group_scope ON sync_group (scope_id);

CREATE TABLE device (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL REFERENCES sync_group (id),
    scope_id TEXT NOT NULL,
    public_key BYTEA NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'revoked')),
    created_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE INDEX device_group ON device (group_id);

CREATE TABLE envelope (
    server_revision BIGINT PRIMARY KEY,
    group_id TEXT NOT NULL,
    scope_id TEXT NOT NULL,
    record_id TEXT NOT NULL,
    device_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    key_epoch BIGINT NOT NULL,
    algorithm TEXT NOT NULL,
    nonce BYTEA NOT NULL,
    ciphertext BYTEA NOT NULL,
    integrity_tag BYTEA NOT NULL,
    aad BYTEA NOT NULL,
    logical_revision BIGINT NOT NULL,
    kind TEXT NOT NULL,
    expected_revision BIGINT NOT NULL,
    parents JSONB NOT NULL,
    suppressed BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX envelope_record ON envelope (group_id, record_id, server_revision);
CREATE INDEX envelope_pull ON envelope (group_id, server_revision);
CREATE INDEX envelope_scope ON envelope (scope_id);

CREATE TABLE revision_counter (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    next_revision BIGINT NOT NULL
);

INSERT INTO revision_counter (singleton, next_revision) VALUES (1, 1);

CREATE TABLE operation_receipt (
    device_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    fingerprint BYTEA NOT NULL,
    server_revision BIGINT NOT NULL,
    sibling BOOLEAN NOT NULL,
    suppressed BOOLEAN NOT NULL,
    key_epoch BIGINT NOT NULL,
    PRIMARY KEY (device_id, operation_id)
);

CREATE TABLE envelope_fingerprint (
    fingerprint BYTEA PRIMARY KEY,
    device_id TEXT NOT NULL,
    operation_id TEXT NOT NULL
);

CREATE TABLE tombstone_ack (
    server_revision BIGINT NOT NULL,
    device_id TEXT NOT NULL,
    PRIMARY KEY (server_revision, device_id)
);
