package clipboardhistory

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
)

func TestEncryptedClipboardRoundTripTamperAndFileReferences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clipboard.bin")
	key := bytes.Repeat([]byte{7}, 32)
	repo, err := Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewClipboard(repo)
	now := time.Now().UTC()
	l := domain.DefaultClipLimits()
	if _, err = service.Capture(domain.ClipTextPayload("sensitive-canary-你好\n🙂"), l, now); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "original-canary.txt")
	if err = os.WriteFile(file, []byte("original bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	grant := []byte("native-access-canary")
	if _, err = service.Capture(domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{file}, FileAccess: [][]byte{grant}}, l, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	sealed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("canary")) || bytes.Contains(sealed, []byte(file)) || bytes.Contains(sealed, []byte("original bytes")) {
		t.Fatal("plaintext on disk")
	}
	restart, _ := Open(path, key)
	items, err := restart.Load()
	if err != nil || len(items) != 2 {
		t.Fatalf("restart %v %v", items, err)
	}
	if len(items[0].Payload.FileAccess) != 1 || !bytes.Equal(items[0].Payload.FileAccess[0], grant) {
		t.Fatal("native file access did not survive encrypted restart")
	}
	if _, err = application.NewClipboard(restart).Refresh(l); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(file); string(b) != "original bytes" {
		t.Fatal("original file mutated")
	}
	if err = os.WriteFile(path, sealed, 0600); err != nil {
		t.Fatal(err)
	}
	wrong, _ := Open(path, bytes.Repeat([]byte{8}, 32))
	if _, err = application.NewClipboard(wrong).Refresh(l); err == nil {
		t.Fatal("wrong key accepted")
	}
	if b, _ := os.ReadFile(path); !bytes.Equal(b, sealed) {
		t.Fatal("wrong key replaced snapshot")
	}
	sealed[len(sealed)-1] ^= 1
	if err = os.WriteFile(path, sealed, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Capture(domain.ClipTextPayload("new"), l, now); err == nil {
		t.Fatal("tamper overwritten")
	}
	if b, _ := os.ReadFile(path); !bytes.Equal(b, sealed) {
		t.Fatal("corrupt snapshot lost")
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if entry.Name() != "clipboard.bin" && entry.Name() != "original-canary.txt" {
			t.Fatalf("left temporary file %s", entry.Name())
		}
	}
}

func TestClipboardLegacySnapshotRefreshesFileAccess(t *testing.T) {
	repo, err := Open(filepath.Join(t.TempDir(), "clipboard.bin"), bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	p := domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{filepath.Join(t.TempDir(), "legacy.txt")}}
	// Compute the original schema's identity independently of the current
	// payload type, so a future field change cannot silently rewrite old IDs.
	legacy, _ := json.Marshal(struct {
		Kind  domain.ClipKind `json:"kind"`
		Files []string        `json:"files"`
	}{p.Kind, p.Files})
	digest := sha256.Sum256(legacy)
	old := domain.Clip{ID: hex.EncodeToString(digest[:]), Payload: p, Copied: time.Now(), Pinned: true}
	b, _ := json.Marshal(snapshot{Version: 1, Items: []domain.Clip{old}})
	if err := repo.file.Write(b); err != nil {
		t.Fatal(err)
	}
	p.FileAccess = [][]byte{[]byte("refreshed native reference")}
	items, err := application.NewClipboard(repo).Capture(p, domain.DefaultClipLimits(), old.Copied.Add(time.Second))
	if err != nil || len(items) != 1 || items[0].ID != old.ID || !items[0].Pinned {
		t.Fatal("legacy file reference lost its identity or pin")
	}
	loaded, err := repo.Load()
	if err != nil || len(loaded) != 1 || string(loaded[0].Payload.FileAccess[0]) != "refreshed native reference" {
		t.Fatal("upgraded snapshot lost native access")
	}
}
