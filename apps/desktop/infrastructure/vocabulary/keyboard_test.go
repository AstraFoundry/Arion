package vocabulary

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"arion.local/core/domain"
)

func TestKeyboardReadingEncryptedLifecycle(t *testing.T) {
	store, path := openTemp(t)
	term, err := store.AddHotword("星槎工坊")
	if err != nil {
		t.Fatal(err)
	}
	bound, err := store.SetKeyboardPinyin(term.ID, term.To, "xing cha gong fang")
	if err != nil {
		t.Fatal(err)
	}
	if raw := readBytes(t, path); bytes.Contains(raw, []byte(bound.KeyboardPinyin)) || bytes.Contains(raw, []byte(term.To)) {
		t.Fatal("plaintext reading or word stored")
	}
	restarted, err := Open(path, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, restarted)[0]; got != bound {
		t.Fatalf("lost reading after restart: %+v", got)
	}
	projection, err := restarted.KeyboardTerms()
	if err != nil || len(projection) != 1 || projection[0].ID != bound.ID || projection[0].Reading != bound.KeyboardPinyin {
		t.Fatalf("encrypted projection: %v %v", projection, err)
	}
	before := readBytes(t, path)
	if _, err := restarted.SetKeyboardPinyin(term.ID, term.To, bound.KeyboardPinyin); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, readBytes(t, path)) {
		t.Fatal("idempotent reading update rewrote ciphertext")
	}
	if _, err := restarted.SetKeyboardPinyin(term.ID, "different", "xing cha"); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale spelling accepted: %v", err)
	}
	if !bytes.Equal(before, readBytes(t, path)) {
		t.Fatal("stale update rewrote ciphertext")
	}
	if err := restarted.Recall([]domain.Replacement{{From: term.From, To: term.To}}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, restarted)[0].KeyboardPinyin; got != bound.KeyboardPinyin {
		t.Fatal("memory roundtrip discarded reading")
	}
	if _, err := restarted.SetKeyboardPinyin(term.ID, term.To, ""); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, restarted)[0]; got.KeyboardPinyin != "" || !got.Hotword {
		t.Fatal("clearing reading damaged hotword")
	}
	if err := restarted.Remove(term.ID); err != nil {
		t.Fatal(err)
	}
	projection, err = restarted.KeyboardTerms()
	if err != nil || len(projection) != 0 {
		t.Fatalf("removed term survives projection: %v %v", projection, err)
	}
	if _, err := restarted.SetKeyboardPinyin(term.ID, term.To, "xing cha"); !errors.Is(err, ErrConflict) {
		t.Fatal("deleted entry recreated")
	}
}

func TestKeyboardReadingRejectsUnapprovedAndInvalidWithoutWriting(t *testing.T) {
	store, path := openTemp(t)
	if err := store.write([]Entry{{ID: "proposal", To: "星槎", Approved: false}}); err != nil {
		t.Fatal(err)
	}
	before := readBytes(t, path)
	if _, err := store.SetKeyboardPinyin("proposal", "星槎", "xing cha"); !errors.Is(err, ErrConflict) {
		t.Fatal("unapproved entry accepted")
	}
	if _, err := store.SetKeyboardPinyin("proposal", "星槎", "xing\ncha"); !errors.Is(err, domain.ErrInvalidKeyboardPinyin) {
		t.Fatal("invalid reading accepted")
	}
	if !bytes.Equal(before, readBytes(t, path)) {
		t.Fatal("invalid request changed original ciphertext")
	}
}

func TestKeyboardReadingClearsWhenLearnedSpellingChanges(t *testing.T) {
	store, _ := openTemp(t)
	now := time.Now()
	learned, err := store.Learn([]domain.Replacement{{From: "星茶", To: "星槎"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	entry := learned[0].Entry
	if _, err := store.SetKeyboardPinyin(entry.ID, entry.To, "xing cha"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Learn([]domain.Replacement{{From: "星茶", To: "星槎"}}, now); err != nil {
		t.Fatal(err)
	}
	if mustList(t, store)[0].KeyboardPinyin != "xing cha" {
		t.Fatal("reinforcement lost reading")
	}
	if _, err := store.Learn([]domain.Replacement{{From: "星茶", To: "星辰"}}, now); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, store)[0]; got.ID != entry.ID || got.KeyboardPinyin != "" {
		t.Fatal("re-pointing retained stale pronunciation")
	}
	if _, err := store.SetKeyboardPinyin(entry.ID, entry.To, "xing cha"); !errors.Is(err, ErrConflict) {
		t.Fatal("stale pronunciation accepted")
	}
}
