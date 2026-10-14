package application

import (
	"context"
	"time"
)

// Scope is the partition a request may touch. ID is opaque to the domain.
type Scope struct {
	ID    string
	Quota int64
}

// Gate resolves a request to a scope before device authentication. noop and
// oidc are the two implementations; a failing oidc check must not fall
// through to noop.
type Gate interface {
	Mode() string
	Authorize(ctx context.Context, authorizationHeader string) (Scope, error)
}

const (
	// EventAccountPurged deletes the tenant and its ciphertext.
	EventAccountPurged = "https://schemas.openid.net/secevent/risc/event-type/account-purged"
	// EventAccountDisabled freezes the tenant and revokes its sessions.
	EventAccountDisabled = "https://schemas.openid.net/secevent/risc/event-type/account-disabled"
	// EventAccountEnabled unfreezes the tenant.
	EventAccountEnabled = "https://schemas.openid.net/secevent/risc/event-type/account-enabled"
	// EventSessionRevoked rejects tokens issued at or before the event.
	EventSessionRevoked = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	// EventEntitlementsChanged is accepted and recorded. Entitlements are
	// read from each access token.
	EventEntitlementsChanged = "https://astrafoundry.si/secevent/entitlements-changed"
)

// SecurityEvent is a verified SET reduced to the fields the relay acts on.
type SecurityEvent struct {
	JTI     string
	Issued  time.Time
	Issuer  string
	Subject string
	Types   []string
}

// EventVerifier checks a SET with the issuer JWKS.
type EventVerifier interface {
	Verify(ctx context.Context, token string) (SecurityEvent, error)
}

// Tenant is the hosted relay's unit of quota and freeze state.
type Tenant struct {
	ID    string
	Quota int64
	State string
}

const (
	TenantActive = "active"
	TenantFrozen = "frozen"
)

// Access is the account-link row the gate reads. The domain never sees it.
type Access struct {
	Tenant               Tenant
	Linked               bool
	Frozen               bool
	Purged               bool
	SessionRevokedAt     time.Time
	HasSessionRevocation bool
}

// Directory maps (issuer, subject) to a tenant.
type Directory interface {
	Lookup(ctx context.Context, issuer, subject string) (Access, error)
	EnsureTenant(ctx context.Context, issuer, subject string, quota int64, now time.Time) (Tenant, error)
}

// Accounts applies security-event effects in one transaction.
type Accounts interface {
	BeginAccount(ctx context.Context) (AccountTx, error)
}

// AccountTx claims a jti and mutates the linked tenant. Rollback is a no-op
// after commit.
type AccountTx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	ClaimJTI(ctx context.Context, jti string, now time.Time) (bool, error)
	SetFrozen(ctx context.Context, issuer, subject string, frozen bool, now time.Time) error
	RevokeSessions(ctx context.Context, issuer, subject string, at time.Time) error
	Purge(ctx context.Context, issuer, subject string) error
}
