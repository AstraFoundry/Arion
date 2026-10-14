package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	_, err := Load(nil)
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestLoadRejectsBothSecretSources(t *testing.T) {
	_, err := Load([]string{
		"ARION_SYNC_DATABASE_URL=postgres://db/sync",
		"ARION_SYNC_DATABASE_URL_FILE=/tmp/db",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestLoadReadsSecretFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db")
	if err := os.WriteFile(path, []byte("postgres://db/from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"ARION_SYNC_DATABASE_URL_FILE=" + path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://db/from-file" || cfg.AccountGate != "noop" {
		t.Fatalf("%+v", cfg)
	}
}

func TestLoadOIDCRequiresIssuerURL(t *testing.T) {
	_, err := Load([]string{
		"ARION_SYNC_DATABASE_URL=postgres://db/sync",
		"ARION_SYNC_ACCOUNT_GATE=oidc",
		"ARION_SYNC_OIDC_ISSUER=not a url",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestLoadRejectsHMACAlgorithm(t *testing.T) {
	_, err := Load([]string{
		"ARION_SYNC_DATABASE_URL=postgres://db/sync",
		"ARION_SYNC_OIDC_ALGORITHMS=HS256,EdDSA",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestLoadRejectsNoneAlgorithm(t *testing.T) {
	_, err := Load([]string{
		"ARION_SYNC_DATABASE_URL=postgres://db/sync",
		"ARION_SYNC_OIDC_ALGORITHMS=none,EdDSA",
	})
	if err == nil {
		t.Fatal("expected an error")
	}
}
