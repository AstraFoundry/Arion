package vocabulary

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"arion.local/core/domain"
)

func TestAddVariantsEncryptedRestartAndIndependentRemoval(t *testing.T) {
	store, path := openTemp(t)
	added, err := store.AddVariants([]string{" ary on ", "ARY ON", "阿里昂"}, " Arion ")
	if err != nil || len(added) != 2 || added[0].From != "ary on" || added[1].From != "阿里昂" {
		t.Fatalf("added %v, %v", added, err)
	}
	if added[0].ID == added[1].ID || added[0].To != "Arion" || added[1].To != "Arion" {
		t.Fatalf("independent aliases %v", added)
	}
	raw := readBytes(t, path)
	if bytes.Contains(raw, []byte("Arion")) || bytes.Contains(raw, []byte("ary on")) {
		t.Fatal("plaintext on disk")
	}
	restarted, err := Open(path, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, restarted); !reflect.DeepEqual(got, added) {
		t.Fatalf("restart %v", got)
	}
	if err := restarted.Remove(added[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, restarted); len(got) != 1 || got[0] != added[1] {
		t.Fatalf("remove affected sibling: %v", got)
	}
}

func TestAddVariantsIdempotentAndConflictsAtomic(t *testing.T) {
	store, path := openTemp(t)
	entry, err := store.Add("ary on", "Arion")
	if err != nil {
		t.Fatal(err)
	}
	raw := readBytes(t, path)
	got, err := store.Add("ARY ON", "Arion")
	if err != nil || got != entry || !bytes.Equal(raw, readBytes(t, path)) {
		t.Fatalf("idempotent add %v, %v", got, err)
	}
	if _, err := store.AddVariants([]string{"new alias", "ARY ON"}, "Other"); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict %v", err)
	}
	if !bytes.Equal(raw, readBytes(t, path)) || !reflect.DeepEqual(mustList(t, store), []Entry{entry}) {
		t.Fatal("conflict changed encrypted file")
	}
	for _, froms := range [][]string{{"new alias", " "}, {"new alias", "Arion"}, {"new alias", "bad\nentry"}} {
		if _, err := store.AddVariants(froms, "Arion"); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid batch %v: %v", froms, err)
		}
		if !bytes.Equal(raw, readBytes(t, path)) {
			t.Fatal("invalid batch changed encrypted file")
		}
	}
}

func TestAddVariantsPromotesLearnedWithoutChangingHistory(t *testing.T) {
	store, path := openTemp(t)
	now := time.Now().Add(-time.Hour).Truncate(time.Second)
	_, err := store.Learn([]domain.Replacement{{From: "ary on", To: "Arion"}, {From: "阿里昂", To: "Arion"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Recall([]domain.Replacement{{From: "ary on", To: "Arion"}}, now); err != nil {
		t.Fatal(err)
	}
	before := mustList(t, store)
	raw := readBytes(t, path)
	if _, err := store.AddVariants([]string{"ary on", "阿里昂"}, "Other"); !errors.Is(err, ErrConflict) {
		t.Fatalf("learned conflict %v", err)
	}
	if !bytes.Equal(raw, readBytes(t, path)) {
		t.Fatal("failed batch promoted learned entry")
	}
	got, err := store.AddVariants([]string{"ARY ON"}, "Arion")
	want := before[0]
	want.Origin = string(domain.OriginUser)
	if err != nil || len(got) != 1 || got[0] != want {
		t.Fatalf("promotion %v, %v; want %v", got, err, want)
	}
	// Each alias retains its own provenance and TTL. Explicit approval
	// protects only the selected alias from expiry and learned changes.
	_, err = store.Learn([]domain.Replacement{{From: "ary on", To: "Other"}}, now.Add(domain.LearnedTTL+time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if entries := mustList(t, store); len(entries) != 1 || entries[0] != want {
		t.Fatalf("promotion did not protect entry independently: %v", entries)
	}
}

func TestAddVariantsUnapprovedDoesNotBlock(t *testing.T) {
	store, _ := openTemp(t)
	proposal := Entry{ID: "proposal", From: "ary on", To: "Other", Approved: false, Origin: "proposal"}
	if err := store.write([]Entry{proposal}); err != nil {
		t.Fatal(err)
	}
	added, err := store.AddVariants([]string{"ary on"}, "Arion")
	if err != nil || len(added) != 1 || !added[0].Approved {
		t.Fatalf("add %v, %v", added, err)
	}
	if got := mustList(t, store); len(got) != 2 || got[0] != proposal {
		t.Fatalf("proposal changed %v", got)
	}
}

func TestAddVariantsPreservesDistinctUnicodeSources(t *testing.T) {
	store, _ := openTemp(t)
	first, err := store.Add("σ", "Sigma")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Add("ς", "FinalSigma")
	if err != nil || second.ID == first.ID {
		t.Fatalf("distinct source rejected or reused: %v, %v", second, err)
	}
	if got, err := store.Add("Σ", "Sigma"); err != nil || got != first {
		t.Fatalf("equivalent source not idempotent: %v, %v", got, err)
	}
	entries := mustList(t, store)
	if len(entries) != 2 {
		t.Fatalf("stored %v", entries)
	}
	v := domain.NewVocabulary(domain.Replacements(Memory(entries)))
	if got, fired := v.Apply("σ ς Σ"); got != "Sigma FinalSigma Sigma" || len(fired) != 2 {
		t.Fatalf("stored sources applied %q, %v", got, fired)
	}
}

func TestAddVariantsReadsLegacyEncryptedPairs(t *testing.T) {
	store, path := openTemp(t)
	// This is the original pair-only JSON schema, without learning fields.
	plain := []byte(`[{"id":"legacy","from":"ary on","to":"Arion","approved":true,"origin":"user_approved"}]`)
	nonce := bytes.Repeat([]byte{3}, store.aead.NonceSize())
	sealed := store.aead.Seal(nonce, nonce, plain, []byte("arion-vocabulary-v1"))
	if err := os.WriteFile(path, sealed, 0o600); err != nil {
		t.Fatal(err)
	}
	added, err := store.AddVariants([]string{"ARY ON", "阿里昂"}, "Arion")
	if err != nil || len(added) != 2 || added[0].ID != "legacy" || added[0].Created != 0 {
		t.Fatalf("legacy add %v, %v", added, err)
	}
	restarted, _ := Open(path, bytes.Repeat([]byte{7}, 32))
	if got := mustList(t, restarted); !reflect.DeepEqual(got, added) {
		t.Fatalf("legacy restart %v", got)
	}
}

func TestAddVariantsStorageFailureLeavesFileUnchanged(t *testing.T) {
	store, path := openTemp(t)
	if _, err := store.Add("ary on", "Arion"); err != nil {
		t.Fatal(err)
	}
	raw := readBytes(t, path)
	if err := os.Mkdir(path+".tmp", 0o700); err != nil {
		t.Fatal(err)
	}
	if got, err := store.AddVariants([]string{"阿里昂", "阿瑞昂"}, "Arion"); err == nil || got != nil {
		t.Fatalf("write failure returned %v, %v", got, err)
	}
	if !bytes.Equal(raw, readBytes(t, path)) || len(mustList(t, store)) != 1 {
		t.Fatal("failed write changed existing memory")
	}
	if err := os.Remove(path + ".tmp"); err != nil {
		t.Fatal(err)
	}
	raw[len(raw)-1] ^= 1
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := store.AddVariants([]string{"阿里昂", "阿瑞昂"}, "Arion"); err == nil || got != nil {
		t.Fatalf("authentication failure returned %v, %v", got, err)
	}
	if !bytes.Equal(raw, readBytes(t, path)) {
		t.Fatal("corrupt file overwritten")
	}
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
