package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"arion.local/sync/domain"
)

// Store is the PostgreSQL repository.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps a pool. The caller owns the pool's lifecycle.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Ping checks that the database accepts a connection.
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Begin starts a sync transaction.
func (s *Store) Begin(ctx context.Context) (domain.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgTx{tx: tx}, nil
}

type pgTx struct {
	tx   pgx.Tx
	done bool
}

func (t *pgTx) Commit(ctx context.Context) error {
	if t.done {
		return nil
	}
	if err := t.tx.Commit(ctx); err != nil {
		return err
	}
	t.done = true
	return nil
}

func (t *pgTx) Rollback(ctx context.Context) error {
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

func (t *pgTx) Group(ctx context.Context, id string) (domain.Group, error) {
	var group domain.Group
	var epoch int64
	err := t.tx.QueryRow(ctx, `SELECT id, scope_id, key_epoch, created_at FROM sync_group WHERE id = $1`, id).
		Scan(&group.ID, &group.ScopeID, &epoch, &group.CreatedAt)
	if err != nil {
		return domain.Group{}, mapErr(err)
	}
	group.KeyEpoch = uint64(epoch)
	return group, nil
}

func (t *pgTx) InsertGroup(ctx context.Context, group domain.Group) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO sync_group (id, scope_id, key_epoch, created_at) VALUES ($1, $2, $3, $4)`,
		group.ID, group.ScopeID, int64(group.KeyEpoch), group.CreatedAt)
	return mapErr(err)
}

func (t *pgTx) SetEpoch(ctx context.Context, groupID string, epoch uint64) error {
	tag, err := t.tx.Exec(ctx, `UPDATE sync_group SET key_epoch = $2 WHERE id = $1`, groupID, int64(epoch))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.Fail(domain.CodeNotFound)
	}
	return nil
}

func (t *pgTx) Device(ctx context.Context, id string) (domain.Device, error) {
	var device domain.Device
	var status string
	err := t.tx.QueryRow(ctx, `SELECT id, group_id, scope_id, public_key, status, created_at, revoked_at FROM device WHERE id = $1`, id).
		Scan(&device.ID, &device.GroupID, &device.ScopeID, &device.PublicKey, &status, &device.CreatedAt, &device.RevokedAt)
	if err != nil {
		return domain.Device{}, mapErr(err)
	}
	device.Status = domain.DeviceStatus(status)
	return device, nil
}

func (t *pgTx) InsertDevice(ctx context.Context, device domain.Device) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO device (id, group_id, scope_id, public_key, status, created_at, revoked_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		device.ID, device.GroupID, device.ScopeID, device.PublicKey, string(device.Status), device.CreatedAt, device.RevokedAt)
	return mapErr(err)
}

func (t *pgTx) Devices(ctx context.Context, groupID string) ([]domain.Device, error) {
	rows, err := t.tx.Query(ctx, `SELECT id, group_id, scope_id, public_key, status, created_at, revoked_at
		FROM device WHERE group_id = $1 ORDER BY id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Device
	for rows.Next() {
		var device domain.Device
		var status string
		if err := rows.Scan(&device.ID, &device.GroupID, &device.ScopeID, &device.PublicKey, &status, &device.CreatedAt, &device.RevokedAt); err != nil {
			return nil, err
		}
		device.Status = domain.DeviceStatus(status)
		out = append(out, device)
	}
	return out, rows.Err()
}

func (t *pgTx) RevokeDevice(ctx context.Context, deviceID string, at time.Time) error {
	tag, err := t.tx.Exec(ctx, `UPDATE device SET status = 'revoked', revoked_at = $2 WHERE id = $1`, deviceID, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.Fail(domain.CodeNotFound)
	}
	return nil
}

const envelopeSelect = `server_revision, group_id, scope_id, record_id, device_id, operation_id, schema_version, key_epoch, algorithm, nonce, ciphertext, integrity_tag, aad, logical_revision, kind, expected_revision, parents, suppressed, created_at`

func (t *pgTx) Envelopes(ctx context.Context, groupID, recordID string) ([]domain.Envelope, error) {
	rows, err := t.tx.Query(ctx, `SELECT `+envelopeSelect+` FROM envelope WHERE group_id = $1 AND record_id = $2 ORDER BY server_revision`, groupID, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEnvelopes(rows)
}

func (t *pgTx) Envelope(ctx context.Context, groupID string, revision int64) (domain.Envelope, error) {
	row := t.tx.QueryRow(ctx, `SELECT `+envelopeSelect+` FROM envelope WHERE group_id = $1 AND server_revision = $2`, groupID, revision)
	env, err := scanEnvelope(row)
	if err != nil {
		return domain.Envelope{}, mapErr(err)
	}
	return env, nil
}

func (t *pgTx) InsertEnvelope(ctx context.Context, env domain.Envelope) error {
	parents, err := json.Marshal(env.Parents)
	if err != nil {
		return err
	}
	if env.Parents == nil {
		parents = []byte("[]")
	}
	_, err = t.tx.Exec(ctx, `INSERT INTO envelope (
		server_revision, group_id, scope_id, record_id, device_id, operation_id, schema_version, key_epoch,
		algorithm, nonce, ciphertext, integrity_tag, aad, logical_revision, kind, expected_revision, parents, suppressed, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		env.ServerRevision, env.GroupID, env.ScopeID, env.RecordID, env.DeviceID, env.OperationID, env.SchemaVersion, int64(env.KeyEpoch),
		env.Algorithm, env.Nonce, env.Ciphertext, env.Tag, env.AAD, int64(env.LogicalRevision), string(env.Kind), env.ExpectedRevision, parents, env.Suppressed, env.CreatedAt)
	return mapErr(err)
}

func (t *pgTx) SetSuppressed(ctx context.Context, revision int64, suppressed bool) error {
	tag, err := t.tx.Exec(ctx, `UPDATE envelope SET suppressed = $2 WHERE server_revision = $1`, revision, suppressed)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.Fail(domain.CodeNotFound)
	}
	return nil
}

func (t *pgTx) DeleteEnvelope(ctx context.Context, revision int64) error {
	_, err := t.tx.Exec(ctx, `DELETE FROM envelope WHERE server_revision = $1`, revision)
	return err
}

func (t *pgTx) ListAfter(ctx context.Context, groupID string, after int64, limit int) ([]domain.Envelope, error) {
	rows, err := t.tx.Query(ctx, `SELECT `+envelopeSelect+` FROM envelope
		WHERE group_id = $1 AND server_revision > $2 ORDER BY server_revision LIMIT $3`, groupID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectEnvelopes(rows)
}

func (t *pgTx) CreatedAt(ctx context.Context, groupID string, revision int64) (time.Time, bool, error) {
	var created time.Time
	err := t.tx.QueryRow(ctx, `SELECT created_at FROM envelope WHERE group_id = $1 AND server_revision = $2`, groupID, revision).Scan(&created)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return created, true, nil
}

func (t *pgTx) MaxRevision(ctx context.Context, groupID string) (int64, error) {
	var max int64
	err := t.tx.QueryRow(ctx, `SELECT COALESCE(MAX(server_revision), 0) FROM envelope WHERE group_id = $1`, groupID).Scan(&max)
	return max, err
}

func (t *pgTx) NextRevision(ctx context.Context) (int64, error) {
	var rev int64
	err := t.tx.QueryRow(ctx, `UPDATE revision_counter SET next_revision = next_revision + 1 WHERE singleton = 1 RETURNING next_revision - 1`).Scan(&rev)
	return rev, err
}

func (t *pgTx) Receipt(ctx context.Context, deviceID, operationID string) (domain.Receipt, bool, error) {
	var receipt domain.Receipt
	var epoch int64
	err := t.tx.QueryRow(ctx, `SELECT device_id, operation_id, fingerprint, server_revision, sibling, suppressed, key_epoch
		FROM operation_receipt WHERE device_id = $1 AND operation_id = $2`, deviceID, operationID).
		Scan(&receipt.DeviceID, &receipt.OperationID, &receipt.Fingerprint, &receipt.ServerRevision, &receipt.Sibling, &receipt.Suppressed, &epoch)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Receipt{}, false, nil
	}
	if err != nil {
		return domain.Receipt{}, false, err
	}
	receipt.KeyEpoch = uint64(epoch)
	return receipt, true, nil
}

func (t *pgTx) InsertReceipt(ctx context.Context, receipt domain.Receipt) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO operation_receipt (device_id, operation_id, fingerprint, server_revision, sibling, suppressed, key_epoch)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		receipt.DeviceID, receipt.OperationID, receipt.Fingerprint, receipt.ServerRevision, receipt.Sibling, receipt.Suppressed, int64(receipt.KeyEpoch))
	return mapErr(err)
}

func (t *pgTx) FingerprintExists(ctx context.Context, fingerprint []byte) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM envelope_fingerprint WHERE fingerprint = $1)`, fingerprint).Scan(&exists)
	return exists, err
}

func (t *pgTx) InsertFingerprint(ctx context.Context, fingerprint []byte, deviceID, operationID string) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO envelope_fingerprint (fingerprint, device_id, operation_id) VALUES ($1, $2, $3)`, fingerprint, deviceID, operationID)
	return mapErr(err)
}

func (t *pgTx) Ack(ctx context.Context, revision int64, deviceID string) error {
	_, err := t.tx.Exec(ctx, `INSERT INTO tombstone_ack (server_revision, device_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, revision, deviceID)
	return err
}

func (t *pgTx) Acks(ctx context.Context, revision int64) ([]string, error) {
	rows, err := t.tx.Query(ctx, `SELECT device_id FROM tombstone_ack WHERE server_revision = $1 ORDER BY device_id`, revision)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (t *pgTx) ScopeBytes(ctx context.Context, scopeID string) (int64, error) {
	var total int64
	err := t.tx.QueryRow(ctx, `SELECT COALESCE(SUM(octet_length(ciphertext)), 0) FROM envelope WHERE scope_id = $1`, scopeID).Scan(&total)
	return total, err
}

func collectEnvelopes(rows pgx.Rows) ([]domain.Envelope, error) {
	var out []domain.Envelope
	for rows.Next() {
		env, err := scanEnvelope(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, env)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEnvelope(row scanner) (domain.Envelope, error) {
	var env domain.Envelope
	var epoch, logical int64
	var kind string
	var parents []byte
	err := row.Scan(
		&env.ServerRevision, &env.GroupID, &env.ScopeID, &env.RecordID, &env.DeviceID, &env.OperationID,
		&env.SchemaVersion, &epoch, &env.Algorithm, &env.Nonce, &env.Ciphertext, &env.Tag, &env.AAD,
		&logical, &kind, &env.ExpectedRevision, &parents, &env.Suppressed, &env.CreatedAt,
	)
	if err != nil {
		return domain.Envelope{}, err
	}
	env.KeyEpoch = uint64(epoch)
	env.LogicalRevision = uint64(logical)
	env.Kind = domain.Kind(kind)
	if len(parents) > 0 {
		if err := json.Unmarshal(parents, &env.Parents); err != nil {
			return domain.Envelope{}, err
		}
	}
	return env, nil
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Fail(domain.CodeNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return domain.Fail(domain.CodeConflict)
	}
	return err
}
