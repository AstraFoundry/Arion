package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"arion.local/sync/application"
	"arion.local/sync/domain"
)

// BeginAccount starts the transaction that claims a security event and
// applies its effect.
func (s *Store) BeginAccount(ctx context.Context) (application.AccountTx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &accountTx{tx: tx}, nil
}

// Lookup reads the link and subject control for one account.
func (s *Store) Lookup(ctx context.Context, issuer, subject string) (application.Access, error) {
	return lookup(ctx, s.pool, issuer, subject)
}

// EnsureTenant creates the tenant for a first authorized request. A concurrent
// insert returns the row that won the unique (issuer, subject) constraint.
func (s *Store) EnsureTenant(ctx context.Context, issuer, subject string, quota int64, now time.Time) (application.Tenant, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return application.Tenant{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO subject_control (issuer, subject, frozen, purged) VALUES ($1, $2, FALSE, FALSE)
		ON CONFLICT DO NOTHING`, issuer, subject); err != nil {
		return application.Tenant{}, err
	}
	var frozen, purged bool
	if err := tx.QueryRow(ctx, `SELECT frozen, purged FROM subject_control WHERE issuer = $1 AND subject = $2 FOR UPDATE`, issuer, subject).Scan(&frozen, &purged); err != nil {
		return application.Tenant{}, err
	}
	if purged {
		return application.Tenant{}, domain.Fail(domain.CodeUnauthorized)
	}
	if frozen {
		return application.Tenant{}, application.Fail(application.CodeTenantFrozen)
	}
	access, err := lookup(ctx, tx, issuer, subject)
	if err != nil {
		return application.Tenant{}, err
	}
	if access.Linked {
		if err := tx.Commit(ctx); err != nil {
			return application.Tenant{}, err
		}
		return access.Tenant, nil
	}
	var buf [16]byte
	if err := randomID(&buf); err != nil {
		return application.Tenant{}, err
	}
	tenant := application.Tenant{ID: encodeID(buf[:]), Quota: quota, State: application.TenantActive}
	if _, err := tx.Exec(ctx, `INSERT INTO tenant (id, quota, state, created_at) VALUES ($1, $2, $3, $4)`, tenant.ID, tenant.Quota, tenant.State, now); err != nil {
		return application.Tenant{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO account_link (issuer, subject, tenant_id, created_at) VALUES ($1, $2, $3, $4)`, issuer, subject, tenant.ID, now); err != nil {
		if isUnique(err) {
			_ = tx.Rollback(ctx)
			access, err = lookup(ctx, s.pool, issuer, subject)
			if err != nil {
				return application.Tenant{}, err
			}
			return access.Tenant, nil
		}
		return application.Tenant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return application.Tenant{}, err
	}
	return tenant, nil
}

type accountTx struct {
	tx   pgx.Tx
	done bool
}

func (t *accountTx) Commit(ctx context.Context) error {
	if t.done {
		return nil
	}
	if err := t.tx.Commit(ctx); err != nil {
		return err
	}
	t.done = true
	return nil
}

func (t *accountTx) Rollback(ctx context.Context) error {
	if t.done {
		return nil
	}
	t.done = true
	err := t.tx.Rollback(ctx)
	if errors.Is(err, pgx.ErrTxClosed) {
		return nil
	}
	return err
}

func (t *accountTx) ClaimJTI(ctx context.Context, jti string, now time.Time) (bool, error) {
	tag, err := t.tx.Exec(ctx, `INSERT INTO security_event (jti, received_at) VALUES ($1, $2) ON CONFLICT DO NOTHING`, jti, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (t *accountTx) SetFrozen(ctx context.Context, issuer, subject string, frozen bool, now time.Time) error {
	state := application.TenantActive
	if frozen {
		state = application.TenantFrozen
	}
	if _, err := t.tx.Exec(ctx, `INSERT INTO subject_control (issuer, subject, frozen, purged) VALUES ($1, $2, $3, FALSE)
		ON CONFLICT (issuer, subject) DO UPDATE SET frozen = CASE WHEN subject_control.purged THEN subject_control.frozen ELSE EXCLUDED.frozen END`,
		issuer, subject, frozen); err != nil {
		return err
	}
	_, err := t.tx.Exec(ctx, `UPDATE tenant SET state = $3
		FROM account_link
		WHERE account_link.tenant_id = tenant.id AND account_link.issuer = $1 AND account_link.subject = $2
		AND NOT EXISTS (SELECT 1 FROM subject_control WHERE issuer = $1 AND subject = $2 AND purged)`,
		issuer, subject, state)
	_ = now
	return err
}

func (t *accountTx) RevokeSessions(ctx context.Context, issuer, subject string, at time.Time) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO subject_control (issuer, subject, frozen, purged, session_revoked_at)
		VALUES ($1, $2, FALSE, FALSE, $3)
		ON CONFLICT (issuer, subject) DO UPDATE SET session_revoked_at = CASE
			WHEN subject_control.session_revoked_at IS NULL THEN EXCLUDED.session_revoked_at
			ELSE GREATEST(subject_control.session_revoked_at, EXCLUDED.session_revoked_at)
		END`, issuer, subject, at)
	return err
}

func (t *accountTx) Purge(ctx context.Context, issuer, subject string) error {
	var scope string
	err := t.tx.QueryRow(ctx, `SELECT tenant_id FROM account_link WHERE issuer = $1 AND subject = $2`, issuer, subject).Scan(&scope)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil {
		if _, err := t.tx.Exec(ctx, `WITH doomed AS (
				SELECT id AS device_id FROM device WHERE scope_id = $1
			), gone_env AS (
				DELETE FROM envelope WHERE scope_id = $1 RETURNING server_revision
			), gone_ack AS (
				DELETE FROM tombstone_ack WHERE server_revision IN (SELECT server_revision FROM gone_env)
			), gone_fp AS (
				DELETE FROM envelope_fingerprint WHERE device_id IN (SELECT device_id FROM doomed)
			), gone_rc AS (
				DELETE FROM operation_receipt WHERE device_id IN (SELECT device_id FROM doomed)
			), gone_dev AS (
				DELETE FROM device WHERE scope_id = $1
			)
			DELETE FROM sync_group WHERE scope_id = $1`, scope); err != nil {
			return err
		}
		if _, err := t.tx.Exec(ctx, `DELETE FROM account_link WHERE issuer = $1 AND subject = $2`, issuer, subject); err != nil {
			return err
		}
		if _, err := t.tx.Exec(ctx, `DELETE FROM tenant WHERE id = $1`, scope); err != nil {
			return err
		}
	}
	_, err = t.tx.Exec(ctx, `INSERT INTO subject_control (issuer, subject, frozen, purged) VALUES ($1, $2, TRUE, TRUE)
		ON CONFLICT (issuer, subject) DO UPDATE SET frozen = TRUE, purged = TRUE`, issuer, subject)
	return err
}

type queryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func lookup(ctx context.Context, q queryer, issuer, subject string) (application.Access, error) {
	access := application.Access{}
	var frozen, purged bool
	var session *time.Time
	err := q.QueryRow(ctx, `SELECT frozen, purged, session_revoked_at FROM subject_control WHERE issuer = $1 AND subject = $2`, issuer, subject).
		Scan(&frozen, &purged, &session)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return application.Access{}, err
	}
	if err == nil {
		access.Frozen = frozen
		access.Purged = purged
		if session != nil {
			access.HasSessionRevocation = true
			access.SessionRevokedAt = *session
		}
	}
	var tenant application.Tenant
	err = q.QueryRow(ctx, `SELECT tenant.id, tenant.quota, tenant.state
		FROM account_link JOIN tenant ON tenant.id = account_link.tenant_id
		WHERE account_link.issuer = $1 AND account_link.subject = $2`, issuer, subject).Scan(&tenant.ID, &tenant.Quota, &tenant.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return access, nil
	}
	if err != nil {
		return application.Access{}, err
	}
	access.Linked = true
	access.Tenant = tenant
	if tenant.State == application.TenantFrozen {
		access.Frozen = true
	}
	return access, nil
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
