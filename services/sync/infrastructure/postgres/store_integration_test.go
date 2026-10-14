//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"arion.local/sync/domain"
)

func TestPostgresPushPull(t *testing.T) {
	url := os.Getenv("ARION_SYNC_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("ARION_SYNC_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	store := New(pool)
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	now := time.Unix(1_700_000_000, 0).UTC()
	group, err := domain.NewGroup("grp-int", "local", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.InsertGroup(ctx, group); err != nil {
		t.Fatal(err)
	}
	key := bytesOf(32)
	if err := tx.InsertDevice(ctx, domain.Device{ID: "dev-int", GroupID: group.ID, ScopeID: "local", PublicKey: key, Status: domain.DeviceActive, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	rev, err := tx.NextRevision(ctx)
	if err != nil {
		t.Fatal(err)
	}
	env := domain.Envelope{
		ServerRevision: rev, GroupID: group.ID, ScopeID: "local", RecordID: "rec-int", DeviceID: "dev-int",
		OperationID: "op-int", SchemaVersion: 1, KeyEpoch: 1, Algorithm: "aes-256-gcm",
		Nonce: bytesOf(12), Ciphertext: []byte("ciphertext-bytes"), Tag: bytesOf(16), AAD: []byte("meta"),
		LogicalRevision: 1, Kind: domain.KindWrite, CreatedAt: now,
	}
	if err := tx.InsertEnvelope(ctx, env); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	read, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback(ctx)
	page, err := read.ListAfter(ctx, group.ID, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 1 || string(page[0].Ciphertext) != "ciphertext-bytes" {
		t.Fatalf("page %+v", page)
	}
	if err := read.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM envelope WHERE group_id = 'grp-int'`)
		_, _ = pool.Exec(context.Background(), `DELETE FROM device WHERE id = 'dev-int'`)
		_, _ = pool.Exec(context.Background(), `DELETE FROM sync_group WHERE id = 'grp-int'`)
	})
}

func bytesOf(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i + 1)
	}
	return out
}
