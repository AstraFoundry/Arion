package domain

import (
	"context"
	"time"
)

// Beginner opens a transaction. Implementations persist ciphertext and
// routing metadata only.
type Beginner interface {
	Begin(ctx context.Context) (Tx, error)
}

// Tx is the unit of work for one device mutation. Rollback is a no-op after
// a successful commit.
type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error

	Group(ctx context.Context, id string) (Group, error)
	InsertGroup(ctx context.Context, group Group) error
	SetEpoch(ctx context.Context, groupID string, epoch uint64) error

	Device(ctx context.Context, id string) (Device, error)
	InsertDevice(ctx context.Context, device Device) error
	Devices(ctx context.Context, groupID string) ([]Device, error)
	RevokeDevice(ctx context.Context, deviceID string, at time.Time) error

	Envelopes(ctx context.Context, groupID, recordID string) ([]Envelope, error)
	Envelope(ctx context.Context, groupID string, revision int64) (Envelope, error)
	InsertEnvelope(ctx context.Context, env Envelope) error
	SetSuppressed(ctx context.Context, revision int64, suppressed bool) error
	DeleteEnvelope(ctx context.Context, revision int64) error
	ListAfter(ctx context.Context, groupID string, after int64, limit int) ([]Envelope, error)
	CreatedAt(ctx context.Context, groupID string, revision int64) (time.Time, bool, error)
	MaxRevision(ctx context.Context, groupID string) (int64, error)
	NextRevision(ctx context.Context) (int64, error)

	Receipt(ctx context.Context, deviceID, operationID string) (Receipt, bool, error)
	InsertReceipt(ctx context.Context, receipt Receipt) error
	FingerprintExists(ctx context.Context, fingerprint []byte) (bool, error)
	InsertFingerprint(ctx context.Context, fingerprint []byte, deviceID, operationID string) error

	Ack(ctx context.Context, revision int64, deviceID string) error
	Acks(ctx context.Context, revision int64) ([]string, error)
	ScopeBytes(ctx context.Context, scopeID string) (int64, error)
}
