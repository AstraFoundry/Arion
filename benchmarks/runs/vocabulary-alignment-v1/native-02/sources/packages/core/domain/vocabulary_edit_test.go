package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNormalizeVocabularyEdit(t *testing.T) {
	got, err := NormalizeVocabularyEdit([]string{"  ary on ", "ARY ON", "阿里昂", "σ", "ς"}, " Arion ")
	want := []Replacement{{From: "ary on", To: "Arion"}, {From: "阿里昂", To: "Arion"}, {From: "σ", To: "Arion"}, {From: "ς", To: "Arion"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized %v, %v", got, err)
	}
	if _, err := NormalizeVocabularyEdit([]string{"arion"}, "Arion"); err != nil {
		t.Fatalf("case correction rejected: %v", err)
	}
}

func TestNormalizeVocabularyEditPreservesDistinctUnicodeMatches(t *testing.T) {
	pairs, err := NormalizeVocabularyEdit([]string{"σ", "Σ", "ς", "s", "S", "ſ"}, "Canonical")
	if err != nil || len(pairs) != 4 {
		t.Fatalf("normalized %v, %v", pairs, err)
	}
	v := NewVocabulary(pairs)
	for _, source := range []string{"σ", "Σ", "ς", "s", "S", "ſ"} {
		got, fired := v.Apply(source)
		if got != "Canonical" || len(fired) != 1 {
			t.Fatalf("source %q: applied %q, %v", source, got, fired)
		}
		if got := v.Terms(source, 8); !reflect.DeepEqual(got, []string{"Canonical"}) {
			t.Fatalf("source %q: recalled %q", source, got)
		}
	}
}

func TestLearningUsesVocabularySourceEquivalence(t *testing.T) {
	now := time.Unix(1000, 0)
	entries, _ := Learn(nil, Replacement{From: "σ", To: "Sigma"}, "sigma", now)
	entries, learned := Learn(entries, Replacement{From: "ς", To: "FinalSigma"}, "final", now)
	if learned.Outcome != LearnAdded || len(entries) != 2 {
		t.Fatalf("distinct source overwritten: %v, %v", entries, learned)
	}
	entries, learned = Learn(entries, Replacement{From: "Σ", To: "Sigma"}, "unused", now)
	if learned.Outcome != LearnReinforced || learned.Entry.ID != "sigma" || len(entries) != 2 {
		t.Fatalf("equivalent source not reinforced: %v, %v", entries, learned)
	}
	entries = Recall(entries, []Replacement{{From: "ς", To: "FinalSigma"}}, now)
	if entries[0].Recalls != 0 || entries[1].Recalls != 1 {
		t.Fatalf("recall conflated distinct sources: %v", entries)
	}
}

func TestNormalizeVocabularyEditRejectsInvalidBatch(t *testing.T) {
	tests := []struct {
		name  string
		froms []string
		to    string
	}{
		{"no aliases", nil, "Arion"},
		{"empty alias", []string{"ary on", " "}, "Arion"},
		{"empty target", []string{"ary on"}, " "},
		{"identity", []string{"ary on", " Arion "}, " Arion "},
		{"alias newline", []string{"ary on\n"}, "Arion"},
		{"target newline", []string{"ary on"}, "Arion\n"},
		{"alias control", []string{"ary\x00on"}, "Arion"},
		{"target control", []string{"ary on"}, "A\u0085rion"},
		{"invalid encoding", []string{"\xff"}, "Arion"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeVocabularyEdit(test.froms, test.to)
			if !errors.Is(err, ErrInvalidVocabularyEdit) || got != nil {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}
