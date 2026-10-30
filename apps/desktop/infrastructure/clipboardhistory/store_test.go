package clipboardhistory

import (
	"bytes"
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
	if _, err = service.Capture(domain.ClipPayload{Kind: domain.ClipFiles, Files: []string{file}}, l, now.Add(time.Second)); err != nil {
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
	if _, err = application.NewClipboard(restart).Refresh(l, now.Add(100*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(file); string(b) != "original bytes" {
		t.Fatal("original file mutated")
	}
	if err = os.WriteFile(path, sealed, 0600); err != nil {
		t.Fatal(err)
	}
	wrong, _ := Open(path, bytes.Repeat([]byte{8}, 32))
	if _, err = application.NewClipboard(wrong).Refresh(l, now.Add(100*24*time.Hour)); err == nil {
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
