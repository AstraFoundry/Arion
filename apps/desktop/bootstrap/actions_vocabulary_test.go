package bootstrap

import (
	"errors"
	"path/filepath"
	"testing"

	"arion.local/desktop/infrastructure/vocabulary"
	arion "arion.local/desktop/ui/shared/runtime"
)

func TestKeyboardReadingActionPreservesEntryAndRejectsStaleSpelling(t *testing.T) {
	store, err := vocabulary.Open(filepath.Join(t.TempDir(), "vocabulary.enc"), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{vocab: store}
	x := actions{a}
	if err := x.AddHotword("星槎工坊"); err != nil {
		t.Fatal(err)
	}
	entry := a.state.Vocabulary[0]
	if entry.DefaultKeyboardPinyin != "xing cha gong fang" || entry.KeyboardPinyin != "" {
		t.Fatalf("expected an unapproved dictionary suggestion: %+v", entry)
	}
	persisted, err := store.List()
	if err != nil || len(persisted) != 1 || persisted[0].KeyboardPinyin != "" {
		t.Fatal("displaying a default must not persist or publish it", err)
	}
	if err := x.SetKeyboardPinyin(entry.ID, entry.To, "xing cha gong fang"); err != nil {
		t.Fatal(err)
	}
	if a.state.Vocabulary[0].KeyboardPinyin != "xing cha gong fang" || !a.state.Vocabulary[0].Hotword {
		t.Fatal("reading was not projected without changing hotword ownership")
	}
	if err := x.SetKeyboardPinyin(entry.ID, "星茶工坊", "xing cha gong fang"); !errors.Is(err, arion.ErrVocabularyConflict) {
		t.Fatal(err)
	}
	if err := x.SetKeyboardPinyin(entry.ID, entry.To, "Xing"); !errors.Is(err, arion.ErrVocabularyInvalid) {
		t.Fatal(err)
	}
	if err := x.SetKeyboardPinyin(entry.ID, entry.To, ""); err != nil {
		t.Fatal(err)
	}
	if len(a.state.Vocabulary) != 1 || a.state.Vocabulary[0].KeyboardPinyin != "" {
		t.Fatal("clearing reading removed or retained the wrong data")
	}
}

func TestVocabularyVariantsActionIsAtomic(t *testing.T) {
	store, err := vocabulary.Open(filepath.Join(t.TempDir(), "vocabulary.enc"), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{vocab: store}
	x := actions{a}
	if err := x.AddReplacementVariants([]string{"cloud code", "clawed code"}, "Claude Code"); err != nil {
		t.Fatal(err)
	}
	if len(a.state.Vocabulary) != 2 || a.vocabulary.Len() != 2 {
		t.Fatalf("missing variants: %v", a.state.Vocabulary)
	}
	if err := x.AddReplacementVariants([]string{"new form", "CLOUD CODE"}, "Other"); !errors.Is(err, arion.ErrVocabularyConflict) {
		t.Fatalf("conflict: %v", err)
	}
	if len(a.state.Vocabulary) != 2 {
		t.Fatalf("failed edit changed active memory: %v", a.state.Vocabulary)
	}
	entries, err := store.List()
	if err != nil || len(entries) != 2 {
		t.Fatalf("failed edit changed stored memory: %v %v", entries, err)
	}
	if err := x.AddReplacement("identical", "identical"); !errors.Is(err, arion.ErrVocabularyInvalid) {
		t.Fatalf("invalid: %v", err)
	}
	if err := x.AddReplacement("a rion", "Arion"); err != nil {
		t.Fatal(err)
	}
	if len(a.state.Vocabulary) != 3 {
		t.Fatalf("legacy add failed: %v", a.state.Vocabulary)
	}
}

func TestInvalidVocabularyEditDoesNotOpenStore(t *testing.T) {
	a := &App{}
	if err := (actions{a}).AddReplacementVariants([]string{"same"}, "same"); !errors.Is(err, arion.ErrVocabularyInvalid) {
		t.Fatalf("invalid: %v", err)
	}
	if a.vocab != nil {
		t.Fatal("invalid input opened vocabulary store")
	}
}

func TestHotwordActionsAndIndependentCorrectionRemoval(t *testing.T) {
	store, err := vocabulary.Open(filepath.Join(t.TempDir(), "vocabulary.enc"), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	a := &App{vocab: store}
	x := actions{a}
	if err := x.AddHotword("Arion"); err != nil {
		t.Fatal(err)
	}
	if err := x.AddReplacement("ary on", "Arion"); err != nil {
		t.Fatal(err)
	}
	if len(a.asrHotwords) != 1 || a.asrHotwords[0] != "Arion" || a.vocabulary.Len() != 1 || len(a.state.Vocabulary) != 2 {
		t.Fatalf("active memory %v %v", a.asrHotwords, a.state.Vocabulary)
	}
	if err := x.AddHotword("<|system|>"); !errors.Is(err, arion.ErrVocabularyInvalid) {
		t.Fatalf("invalid %v", err)
	}
	x.RemoveReplacement(a.state.Vocabulary[1].ID)
	if len(a.asrHotwords) != 1 || len(a.state.Vocabulary) != 1 || a.vocabulary.Len() != 0 {
		t.Fatal("correction removal removed independent hotword")
	}
	x.RemoveReplacement(a.state.Vocabulary[0].ID)
	if len(a.asrHotwords) != 0 {
		t.Fatal("removed hotword remains active")
	}
}
