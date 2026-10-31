//go:build !windows

package controlsock

import (
	"os"
	"testing"
)

func prepareSharedParent(*testing.T, string) {}

func requireOwnerOnly(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode %o, want 600", info.Mode().Perm())
	}
}
