package domain

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestASRHotwordsCanonicalEligibilityAndRanking(t *testing.T) {
	now := time.Unix(1234, 0)
	entries := []MemoryEntry{
		{ID: "manual-pair", From: "clawed", To: "WrongScope", Origin: OriginUser},
		{ID: "learned", From: "clawed code", To: "Claude Code", Origin: OriginEdit, Uses: 4, Used: now},
		{ID: "alias", From: "cloud code", To: "Claude Code", Origin: OriginEdit, Uses: 1},
		{ID: "explicit", To: "Arion", Origin: OriginUser, Hotword: true},
		{ID: "case", To: "ARION", Origin: OriginUser, Hotword: true},
		{ID: "invalid", To: "<|assistant|>", Origin: OriginEdit},
		{ID: "sentence", To: "an entire sentence from an old edit", Origin: OriginEdit},
	}
	want := []string{"ARION", "Arion", "Claude Code"}
	if got := ASRHotwords(entries); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	if got := ASRHotwords(entries); !reflect.DeepEqual(got, want) {
		t.Fatalf("storage order affected selection %v", got)
	}
}

func TestASRHotwordBudgetsAndValidation(t *testing.T) {
	for _, term := range []string{"", " ", "bad\nterm", "bad\x00term", "<|system", "term|>", "a\u2028b", "a\u200bb", strings.Repeat("字", 65)} {
		if _, err := NormalizeHotword(term); !errors.Is(err, ErrInvalidHotword) {
			t.Fatalf("accepted %q: %v", term, err)
		}
	}
	if got, err := NormalizeHotword(" Claude Code "); got != "Claude Code" || err != nil {
		t.Fatalf("normalize %q %v", got, err)
	}
	var entries []MemoryEntry
	for i := 0; i < 100; i++ {
		entries = append(entries, MemoryEntry{ID: fmt.Sprint(i), To: fmt.Sprintf("%02d%s", i, strings.Repeat("字", 62)), Origin: OriginUser, Hotword: true})
	}
	got := ASRHotwords(entries)
	if len(got) != 31 {
		t.Fatalf("rune budget selected %d terms", len(got))
	}
	if _, err := ASRContext(got); err != nil {
		t.Fatal(err)
	}
	small := make([]string, 65)
	for i := range small {
		small[i] = "a"
	}
	if _, err := ASRContext(small); !errors.Is(err, ErrASRContextBudget) {
		t.Fatalf("count %v", err)
	}
	if _, err := ASRContext(small[:64]); err != nil {
		t.Fatal(err)
	}
	large := make([]string, 32)
	for i := range large {
		large[i] = strings.Repeat("字", 64)
	}
	if _, err := ASRContext(large); !errors.Is(err, ErrASRContextBudget) {
		t.Fatalf("runes %v", err)
	}
	if _, err := ASRContext([]string{" a"}); !errors.Is(err, ErrInvalidHotword) {
		t.Fatalf("untrimmed %v", err)
	}
}
