package vocabulary

import (
	"arion.local/core/domain"
	"bytes"
	"reflect"
	"testing"
	"time"
)

func TestHotwordsEncryptedRestartAndLearningLifecycle(t *testing.T) {
	store, path := openTemp(t)
	now := time.Unix(1800000000, 0)
	learned, err := store.Learn([]domain.Replacement{{From: "clawed code", To: "Claude Code"}}, now)
	if err != nil {
		t.Fatal(err)
	}
	manual, err := store.AddHotword(" Claude Code ")
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.AddHotword("Claude Code")
	if err != nil || again != manual {
		t.Fatalf("duplicate changed identity %v %v", again, err)
	}
	if _, err := store.Add("ary on", "Arion"); err != nil {
		t.Fatal(err)
	}
	proposal := Entry{ID: "pending", To: "Pending", Hotword: true}
	entries := mustList(t, store)
	if err := store.write(append(entries, proposal)); err != nil {
		t.Fatal(err)
	}
	if raw := readBytes(t, path); bytes.Contains(raw, []byte("Claude Code")) {
		t.Fatal("plaintext stored")
	}
	restarted, err := Open(path, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	hotwords := func() []string { return domain.ASRHotwords(Memory(mustList(t, restarted))) }
	if got := hotwords(); !reflect.DeepEqual(got, []string{"Claude Code"}) {
		t.Fatalf("hotwords %v", got)
	}
	if err := restarted.Remove(learned[0].Entry.ID); err != nil {
		t.Fatal(err)
	}
	if got := hotwords(); !reflect.DeepEqual(got, []string{"Claude Code"}) {
		t.Fatalf("undo removed manual word %v", got)
	}
	if err := restarted.Remove(manual.ID); err != nil {
		t.Fatal(err)
	}
	if got := hotwords(); len(got) != 0 {
		t.Fatalf("removed hotword remains %v", got)
	}
	if _, err := restarted.Learn([]domain.Replacement{{From: "clawed code", To: "Claude Code"}}, now); err != nil {
		t.Fatal(err)
	}
	if got := hotwords(); !reflect.DeepEqual(got, []string{"Claude Code"}) {
		t.Fatalf("learning %v", got)
	}
	if _, err := restarted.Learn([]domain.Replacement{{From: "Claude Code", To: "clawed code"}}, now); err != nil {
		t.Fatal(err)
	}
	if got := hotwords(); len(got) != 0 {
		t.Fatalf("reversal %v", got)
	}
}

func TestLegacyLearnedHotwordSurvivesApproval(t *testing.T) {
	store, _ := openTemp(t)
	entries := []Entry{{ID: "legacy", From: "clawed code", To: "Claude Code", Approved: true, Origin: string(domain.OriginEdit)}}
	if err := store.write(entries); err != nil {
		t.Fatal(err)
	}
	if got := domain.ASRHotwords(Memory(mustList(t, store))); len(got) != 1 {
		t.Fatalf("legacy %v", got)
	}
	if _, err := store.Add("CLAWED CODE", "Claude Code"); err != nil {
		t.Fatal(err)
	}
	got := mustList(t, store)[0]
	if !got.Hotword || got.Origin != string(domain.OriginUser) || got.ID != "legacy" {
		t.Fatalf("promotion lost eligibility %v", got)
	}
}
