package vocabulary

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"arion.local/core/domain"
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

func openTemp(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "vocabulary.bin")
	store, err := Open(path, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return store, path
}

func TestLearnedEntriesRoundtripEncrypted(t *testing.T) {
	store, path := openTemp(t)
	now := time.Unix(1_800_000_000, 0)
	if _, err := store.Add("arion", "Arion"); err != nil {
		t.Fatal(err)
	}
	results, err := store.Learn([]domain.Replacement{{From: "送火-canary", To: "纵火"}, {From: "jason", To: "Jayson"}}, now)
	if err != nil || len(results) != 2 || results[0].Outcome != domain.LearnAdded {
		t.Fatalf("learn %v %v", results, err)
	}
	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("canary")) || bytes.Contains(raw, []byte("Jayson")) {
		t.Fatal("plaintext found on disk")
	}
	if err := store.Recall([]domain.Replacement{{From: "jason", To: "Jayson"}}, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	memory := Memory(mustList(t, store))
	if len(memory) != 3 || memory[2].Recalls != 1 || memory[2].Origin != domain.OriginEdit || memory[0].Origin != domain.OriginUser {
		t.Fatalf("memory %+v", memory)
	}
	// The user corrects a learned replacement back: it is removed.
	results, _ = store.Learn([]domain.Replacement{{From: "Jayson", To: "jason"}}, now)
	if results[0].Outcome != domain.LearnReverted || len(mustList(t, store)) != 2 {
		t.Fatalf("revert %v", results)
	}
}

func mustList(t *testing.T, s *Store) []Entry {
	t.Helper()
	entries, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

// A crash between writing the temporary file and renaming it leaves the
// previous file in force; the next write replaces the leftover.
func TestCrashLeftoverIsIgnored(t *testing.T) {
	store, path := openTemp(t)
	if _, err := store.Add("a", "b"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".tmp", []byte("half-written"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, store); len(got) != 1 {
		t.Fatalf("entries %v", got)
	}
	if _, err := store.Learn([]domain.Replacement{{From: "c", To: "d"}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, store); len(got) != 2 {
		t.Fatalf("entries %v", got)
	}
}

// A file that fails authentication is never overwritten by learning.
func TestCorruptFileFailsClosed(t *testing.T) {
	store, path := openTemp(t)
	if _, err := store.Add("a", "b"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	raw[len(raw)/2] ^= 0xff
	_ = os.WriteFile(path, raw, 0o600)
	if _, err := store.Learn([]domain.Replacement{{From: "c", To: "d"}}, time.Now()); err == nil {
		t.Fatal("learned into a corrupt file")
	}
	if err := store.Recall([]domain.Replacement{{From: "a", To: "b"}}, time.Now()); err == nil {
		t.Fatal("recalled into a corrupt file")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(after, raw) {
		t.Fatal("corrupt file was overwritten")
	}
}

func TestConcurrentWritersKeepEveryEntry(t *testing.T) {
	store, _ := openTemp(t)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = store.Learn([]domain.Replacement{{From: fmt.Sprintf("from%d", i), To: "to"}}, time.Now())
			_, _ = store.List()
		}(i)
	}
	wg.Wait()
	if got := mustList(t, store); len(got) != 16 {
		t.Fatalf("entries %d", len(got))
	}
}
