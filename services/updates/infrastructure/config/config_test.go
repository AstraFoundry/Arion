package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func base() []string {
	return []string{
		"ARION_UPDATES_PUBLIC_URL=https://api.arion.example/",
		"ARION_UPDATES_S3_ENDPOINT=https://account.r2.cloudflarestorage.com",
		"ARION_UPDATES_S3_REGION=auto",
		"ARION_UPDATES_S3_BUCKET=releases",
		"ARION_UPDATES_S3_PREFIX=arion",
		"ARION_UPDATES_S3_ACCESS_KEY_ID=id",
		"ARION_UPDATES_S3_SECRET_ACCESS_KEY=secret",
	}
}

func TestLoad(t *testing.T) {
	cfg, err := Load(base())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicURL != "https://api.arion.example" || !cfg.S3PathStyle || cfg.PresignTTL != 10*time.Minute || cfg.CacheTTL != 30*time.Second || cfg.HTTPAddr != ":8081" {
		t.Fatalf("config = %+v", cfg)
	}
}

func TestLoadSecretFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(file, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := append(base()[:6], "ARION_UPDATES_S3_SECRET_ACCESS_KEY_FILE="+file)
	cfg, err := Load(env)
	if err != nil || cfg.S3SecretKey != "from-file" {
		t.Fatalf("secret file = %q, %v", cfg.S3SecretKey, err)
	}
	if _, err := Load(append(base(), "ARION_UPDATES_S3_SECRET_ACCESS_KEY_FILE="+file)); err == nil {
		t.Fatal("value and file both set were accepted")
	}
}

func TestLoadRejects(t *testing.T) {
	for _, extra := range []string{
		"ARION_UPDATES_PUBLIC_URL=http://api.arion.example",
		"ARION_UPDATES_PUBLIC_URL=https://user:pw@api.arion.example",
		"ARION_UPDATES_S3_BUCKET=",
		"ARION_UPDATES_S3_PATH_STYLE=yes",
		"ARION_UPDATES_PRESIGN_TTL=200h",
		"ARION_UPDATES_CACHE_TTL=soon",
		"ARION_UPDATES_LOG_LEVEL=loud",
	} {
		if _, err := Load(append(base(), extra)); err == nil {
			t.Errorf("accepted %s", extra)
		}
	}
	if _, err := Load(append(base(), "ARION_UPDATES_PUBLIC_URL=http://127.0.0.1:8081")); err != nil {
		t.Errorf("loopback http rejected: %v", err)
	}
}
