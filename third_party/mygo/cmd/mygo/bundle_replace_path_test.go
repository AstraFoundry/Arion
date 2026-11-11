package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplacePathCommitsPackageStage(t *testing.T) {
	base := t.TempDir()
	src, dst := filepath.Join(base, "stage"), filepath.Join(base, "final")
	for path, value := range map[string]string{src: "new package", dst: "old package"} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "payload"), []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := replacePath(src, dst); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(dst, "payload")); err != nil || string(got) != "new package" {
		t.Fatalf("destination = %q, %v; want new package", got, err)
	}
	if _, err := os.Lstat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists or unexpected stat error: %v", err)
	}
	if old, err := filepath.Glob(filepath.Join(base, ".old-*")); err != nil || len(old) != 0 {
		t.Fatalf("old destinations = %v, %v", old, err)
	}
}

func TestReplacePathMissingStageRestoresDestination(t *testing.T) {
	base := t.TempDir()
	src, dst := filepath.Join(base, "missing-stage"), filepath.Join(base, "final")
	if err := os.Mkdir(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "payload"), []byte("old package"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := replacePath(src, dst); !os.IsNotExist(err) {
		t.Fatalf("rename error = %v, want missing source", err)
	}
	if got, err := os.ReadFile(filepath.Join(dst, "payload")); err != nil || string(got) != "old package" {
		t.Fatalf("destination = %q, %v; want preserved old package", got, err)
	}
	if old, err := filepath.Glob(filepath.Join(base, ".old-*")); err != nil || len(old) != 0 {
		t.Fatalf("rollback left old destinations = %v, %v", old, err)
	}
}
