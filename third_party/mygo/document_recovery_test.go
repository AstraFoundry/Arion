//go:build !js

package mygo

import (
	"encoding/json/v2"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// recoveryDocs keeps the automatic restore from racing the explicit restores below.
func recoveryDocs(t *testing.T) *Documents[string] {
	t.Helper()
	docRegistry.Lock()
	launched := docRegistry.launched
	docRegistry.launched = false
	docRegistry.Unlock()
	defer func() {
		docRegistry.Lock()
		docRegistry.launched = launched
		docRegistry.Unlock()
	}()
	return testDocs(t, DocumentsOptions[string]{Autosave: AutosaveOff})
}

func writeRecoveryOrphan(t *testing.T, ds *Documents[string], name string, entries []recoveryEntry, copies map[string]string) string {
	t.Helper()
	root, err := ds.kind.recovery.root()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	index, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), index, 0o600); err != nil {
		t.Fatal(err)
	}
	for id, content := range copies {
		if err := os.WriteFile(filepath.Join(dir, id), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDocumentRecoveryKeepsSkippedCopy(t *testing.T) {
	ds := recoveryDocs(t)
	path := writeTemp(t, "already-open.txt", "on disk")
	if _, err := ds.Open(path); err != nil {
		t.Fatal(err)
	}
	stamp, _, err := statStamp(path)
	if err != nil {
		t.Fatal(err)
	}
	dir := writeRecoveryOrphan(t, ds, "already-open", []recoveryEntry{{ID: "unsaved", Path: path, Stamp: stamp}}, map[string]string{"unsaved": "unsaved changes"})

	if n := ds.kind.restore(); n != 0 {
		t.Fatalf("restored %d documents, want none", n)
	}
	if got := readString(t, filepath.Join(dir, "unsaved")); got != "unsaved changes" {
		t.Fatalf("recovery copy = %q, want unsaved changes", got)
	}
}

func TestDocumentRecoveryKeepsCopyWhenBackupFails(t *testing.T) {
	ds := recoveryDocs(t)
	dir := writeRecoveryOrphan(t, ds, "backup-fails", []recoveryEntry{{ID: "unsaved", Name: "Untitled"}}, map[string]string{"unsaved": "unsaved changes"})
	create := ds.kind.create
	blocked := writeTemp(t, "user-data-is-a-file", "not a directory")
	ds.kind.create = func(content any, name string) (*docBase, error) {
		App.SetPath(PathUserData, blocked)
		return create(content, name)
	}
	t.Cleanup(func() { ds.kind.create = create })

	if n := ds.kind.restore(); n != 0 {
		t.Fatalf("restored %d documents without a recovery copy, want none", n)
	}
	if got := readString(t, filepath.Join(dir, "unsaved")); got != "unsaved changes" {
		t.Fatalf("recovery copy = %q, want unsaved changes", got)
	}
}

func TestDocumentRecoveryCleansOnlyRestoredEntries(t *testing.T) {
	ds := recoveryDocs(t)
	entries := []recoveryEntry{{ID: "good", Name: "Good"}, {ID: "bad", Name: "Bad"}}
	dir := writeRecoveryOrphan(t, ds, "partial", entries, map[string]string{"good": "good content", "bad": "bad content"})
	read := ds.kind.read
	ds.kind.read = func(r io.Reader) (any, error) {
		v, err := read(r)
		if err == nil && v == "bad content" {
			return nil, errors.New("cannot read bad content")
		}
		return v, err
	}

	if n := ds.kind.restore(); n != 1 {
		t.Fatalf("restored %d documents, want one", n)
	}
	if _, err := os.Stat(filepath.Join(dir, "good")); !os.IsNotExist(err) {
		t.Fatalf("the restored copy was not removed: %v", err)
	}
	if got := readString(t, filepath.Join(dir, "bad")); got != "bad content" {
		t.Fatalf("failed recovery copy = %q, want bad content", got)
	}
	var remaining []recoveryEntry
	if err := json.Unmarshal([]byte(readString(t, filepath.Join(dir, "index.json"))), &remaining); err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].ID != "bad" {
		t.Fatalf("remaining index = %+v, want only bad", remaining)
	}

	ds.kind.read = read
	if n := ds.kind.restore(); n != 1 {
		t.Fatalf("restored %d documents on the second attempt, want one", n)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("the fully restored session remains: %v", err)
	}
}

func TestDocumentRecoveryRemovesSessionsWithoutEntries(t *testing.T) {
	for _, missingIndex := range []bool{false, true} {
		name := "empty-index"
		if missingIndex {
			name = "missing-index"
		}
		t.Run(name, func(t *testing.T) {
			ds := recoveryDocs(t)
			var copies map[string]string
			if missingIndex {
				copies = map[string]string{"unindexed": "unsaved changes"}
			}
			dir := writeRecoveryOrphan(t, ds, name, []recoveryEntry{}, copies)
			if missingIndex {
				if err := os.Remove(filepath.Join(dir, "index.json")); err != nil {
					t.Fatal(err)
				}
			}

			if n := ds.kind.restore(); n != 0 {
				t.Fatalf("restored %d documents, want none", n)
			}
			if _, err := os.Stat(dir); !os.IsNotExist(err) {
				t.Fatalf("the session without entries remains: %v", err)
			}
		})
	}
}
