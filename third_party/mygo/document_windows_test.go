//go:build windows

package mygo

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestOpenSharedLetsAppsSave replaces and deletes a document's file
// while the app reads it. A file opened with os.Open refuses both ("Access
// is denied"); one openShared opened lets another app replace it with a
// POSIX-semantics rename (as os.Root.Rename and current save code do) or
// delete it, and the app goes on reading what it opened. A rename by
// MoveFileEx, as os.Rename does, is refused while any handle to the file
// is open, whatever it shares: an app saving so tries again once the app
// has read the file (TestDocumentOtherApps).
func TestOpenSharedLetsAppsSave(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	path := filepath.Join(dir, "doc.txt")
	reset := func() {
		t.Helper()
		if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	save := func() error {
		if err := os.WriteFile(path+".tmp", []byte("two"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root.Rename("doc.txt.tmp", "doc.txt")
	}

	// What os.Open refuses, here.
	reset()
	plain, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("with os.Open: save %v, delete %v", save(), os.Remove(path))
	plain.Close()

	reset()
	f, err := openShared(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := save(); err != nil {
		t.Fatalf("saving over a file the app reads: %v", err)
	}
	if b, err := io.ReadAll(f); err != nil || string(b) != "one" {
		t.Errorf("read %q, %v from the file opened before the save", b, err)
	}
	f.Close()
	if b, _ := os.ReadFile(path); string(b) != "two" {
		t.Errorf("the file has %q after the save", b)
	}

	reset()
	if f, err = openShared(path); err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := os.Remove(path); err != nil {
		t.Fatalf("deleting a file the app reads: %v", err)
	}
	if b, err := io.ReadAll(f); err != nil || string(b) != "one" {
		t.Errorf("read %q, %v from the file opened before it was deleted", b, err)
	}
}
