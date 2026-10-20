package domain

import (
	"errors"
	"testing"
)

func TestParseValues(t *testing.T) {
	if c, err := ParseChannel(""); err != nil || c != Stable {
		t.Fatalf("empty channel = %q, %v", c, err)
	}
	if c, err := ParseChannel("beta"); err != nil || c != Beta {
		t.Fatalf("beta = %q, %v", c, err)
	}
	if _, err := ParseChannel("nightly"); !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("nightly: %v", err)
	}
	for _, bad := range []string{"darwin", "darwin-x86", "../darwin-arm64", "DARWIN-ARM64"} {
		if _, err := ParseTarget(bad); err == nil {
			t.Errorf("target %q accepted", bad)
		}
	}
	if target, ok := ManifestTarget("update-windows-amd64.json"); !ok || target != "windows-amd64" {
		t.Fatalf("manifest target = %q, %v", target, ok)
	}
	for _, good := range []string{"arion-0.1.0-darwin-arm64.tar.gz", "Arion.0.1.0-alpha.6.arm64.dmg", "arion-0.1.0-to-0.1.1-darwin-arm64.delta"} {
		if _, err := ParseFileName(good); err != nil {
			t.Errorf("file %q rejected", good)
		}
	}
	for _, bad := range []string{"", "a/b", "..", "a..b", ".hidden", "update-darwin-arm64.json", "a b"} {
		if _, err := ParseFileName(bad); err == nil {
			t.Errorf("file %q accepted", bad)
		}
	}
}

func TestParseIndex(t *testing.T) {
	valid := `{"schema":"astrafoundry.release-channel.v1","product":"arion","channel":"beta","updatedAt":"2026-10-20T00:00:00Z","targets":{"darwin-arm64":{"version":"0.1.0-alpha.6","prerelease":true,"publishedAt":"2026-10-20T00:00:00Z","notes":"","installer":{"name":"Arion.0.1.0-alpha.6.arm64.dmg","size":10,"sha256":"` + hex64 + `"},"update":{"name":"arion-0.1.0-alpha.6-darwin-arm64.tar.gz","size":9,"sha256":"` + hex64 + `"}}}}`
	index, err := ParseIndex([]byte(valid), Beta)
	if err != nil || index.Targets["darwin-arm64"].Version != "0.1.0-alpha.6" {
		t.Fatalf("index = %+v, %v", index, err)
	}
	if _, err := ParseIndex([]byte(valid), Stable); !errors.Is(err, ErrInvalidIndex) {
		t.Fatalf("a beta index read as stable: %v", err)
	}
	if _, err := ParseIndex([]byte(`{"schema":"other"}`), Beta); !errors.Is(err, ErrInvalidIndex) {
		t.Fatalf("wrong schema: %v", err)
	}
}

func TestCheckManifest(t *testing.T) {
	if err := CheckManifest([]byte(`{"version":"1.0.0","url":"https://x/a.tar.gz","size":3,"signature":"c2ln"}`)); err != nil {
		t.Fatal(err)
	}
	if err := CheckManifest([]byte(`{"version":"1.0.0"}`)); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("incomplete manifest: %v", err)
	}
}

const hex64 = "0000000000000000000000000000000000000000000000000000000000000000"
