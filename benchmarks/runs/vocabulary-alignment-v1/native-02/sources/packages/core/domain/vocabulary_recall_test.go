package domain

import (
	"reflect"
	"testing"
)

func TestVocabularyAliasesShareCanonicalRecallBudget(t *testing.T) {
	v := NewVocabulary([]Replacement{
		{From: "cloud code", To: "Claude Code"},
		{From: "clod code", To: "Claude Code"},
		{From: "克劳德", To: "Claude Code"},
		{From: "阿里昂", To: "Arion"},
	})
	got := v.Terms("cloud code clod code 克劳德 Claude Code 阿里昂", 2)
	if !reflect.DeepEqual(got, []string{"Claude Code", "Arion"}) {
		t.Fatalf("canonical recall: %q", got)
	}
	corrected, fired := v.Apply("cloud code 和 clod code、克劳德")
	if corrected != "Claude Code 和 Claude Code、Claude Code" || len(fired) != 3 {
		t.Fatalf("per-alias correction evidence lost: %q %v", corrected, fired)
	}
	if got := v.Terms("cloud coder and clod codes", 8); len(got) != 0 {
		t.Fatalf("partial word recalled: %q", got)
	}
}
