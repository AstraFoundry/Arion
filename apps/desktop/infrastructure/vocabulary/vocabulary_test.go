package vocabulary

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// PRV-A02 / PRV-A06: values are ciphertext on disk and a wrong key or a
// tampered file fails closed.
func TestEncryptedAtRest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vocabulary.bin")
	key := bytes.Repeat([]byte{7}, 32)
	store, err := Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add("canary-阿里昂", "Arion"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("canary")) || bytes.Contains(raw, []byte("Arion")) {
		t.Fatal("plaintext found on disk")
	}
	entries, err := store.List()
	if err != nil || len(entries) != 1 || entries[0].To != "Arion" || !entries[0].Approved {
		t.Fatalf("roundtrip failed: %v %v", entries, err)
	}
	other, _ := Open(path, bytes.Repeat([]byte{8}, 32))
	if _, err := other.List(); err == nil {
		t.Fatal("wrong key decrypted")
	}
	raw[len(raw)-1] ^= 1
	_ = os.WriteFile(path, raw, 0o600)
	if _, err := store.List(); err == nil {
		t.Fatal("tampered file accepted")
	}
	if _, err := store.Add(" ", "x"); err != ErrInvalid {
		t.Fatal("blank entry accepted")
	}
}
