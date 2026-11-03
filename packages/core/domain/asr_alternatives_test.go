package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestPrepareASRAnchorsCanonicalTerms(t *testing.T) {
	lexicon := &SpellingLexicon{Readings: map[string][]string{"非": {"fei1"}, "飞": {"fei1"}, "书": {"shu1"}}}
	aliases := NewVocabulary([]Replacement{{From: "寇代克斯", To: "Codex"}, {From: "noise", To: "Unselected"}})
	cases := []struct {
		name, source string
		terms, want  []string
	}{
		{"empty", "", []string{"Codex"}, nil},
		{"nonempty unrelated noise", "ordinary words", []string{"Codex", "飞书"}, nil},
		{"Chinese complete canonical", "把非书打开。", []string{"飞书", "书飞"}, []string{"飞书"}},
		{"already canonical", "把飞书打开。", []string{"飞书"}, nil},
		{"approved cross script alias", "用寇代克斯检查。", []string{"Codex", "Unrelated"}, []string{"Codex"}},
		{"English typo", "inspect novaboardd now", []string{"NovaBoard", "Unrelated"}, []string{"NovaBoard"}},
		{"English compound", "inspect nova board now", []string{"NovaBoard"}, []string{"NovaBoard"}},
		{"short fuzzy excluded", "inspect cot now", []string{"Cat"}, nil},
		{"whole Latin boundary", "xnovaboardd2", []string{"NovaBoard"}, nil},
		{"unselected alias", "noise", []string{"Codex"}, nil},
		{"duplicate snapshot term", "nova board", []string{"NovaBoard", "NovaBoard"}, []string{"NovaBoard"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := lexicon.PrepareASR(tc.source, tc.terms, aliases)
			if got := plan.Terms(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Terms() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPrepareASRCanonicalOnlyCrossScriptAbstains(t *testing.T) {
	plan := (&SpellingLexicon{}).PrepareASR("用寇代克斯检查。", []string{"Codex"}, nil)
	if len(plan.Terms()) != 0 {
		t.Fatalf("must not invent cross-script pronunciation: %q", plan.Terms())
	}
}

func TestASRResolveRequiresCompleteSourceBackbone(t *testing.T) {
	source := "Please inspect novaboardd, then stop."
	plan := (&SpellingLexicon{}).PrepareASR(source, []string{"NovaBoard", "OtherTerm"}, nil)
	cases := []struct {
		name, proposal, want string
		accepted             bool
	}{
		{"selected", "Please inspect NovaBoard, then stop.", "Please inspect NovaBoard, then stop.", true},
		{"keep", source, source, true},
		{"punctuation and whitespace", "Please  inspect NovaBoard; then stop!", "Please inspect NovaBoard, then stop.", true},
		{"table echo", "NovaBoard OtherTerm", source, false},
		{"addition after valid recognition", "Please inspect NovaBoard, then stop. OtherTerm", source, false},
		{"prefix", "OtherTerm Please inspect NovaBoard, then stop.", source, false},
		{"missing backbone", "Please inspect NovaBoard.", source, false},
		{"empty", "", source, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, accepted := plan.Resolve(tc.proposal)
			if got != tc.want || accepted != tc.accepted {
				t.Fatalf("Resolve() = (%q, %v), want (%q, %v)", got, accepted, tc.want, tc.accepted)
			}
		})
	}
}

func TestASRResolveRepeatedSourceSpans(t *testing.T) {
	aliases := NewVocabulary([]Replacement{{From: "寇代克斯", To: "Codex"}})
	plan := (&SpellingLexicon{}).PrepareASR("寇代克斯，寇代克斯。", []string{"Codex"}, aliases)
	got, accepted := plan.Resolve("Codex; 寇代克斯!")
	if !accepted || got != "Codex，寇代克斯。" {
		t.Fatalf("Resolve() = (%q, %v)", got, accepted)
	}
}

func TestASRResolveSingleRelevantWordRejectsVocabularyEcho(t *testing.T) {
	plan := (&SpellingLexicon{}).PrepareASR("novaboardd", []string{"NovaBoard", "OtherTerm"}, nil)
	if got, accepted := plan.Resolve("NovaBoard OtherTerm"); accepted || got != "novaboardd" {
		t.Fatalf("single-word vocabulary echo Resolve() = (%q, %v)", got, accepted)
	}
}

func TestASRResolveAmbiguousAssignmentsPreserveSource(t *testing.T) {
	plan := (&SpellingLexicon{}).PrepareASR("alphabeta.", []string{"AlphaBeta", "Alpha Beta"}, nil)
	if got, accepted := plan.Resolve("AlphaBeta!"); accepted || got != "alphabeta." {
		t.Fatalf("ambiguous Resolve() = (%q, %v)", got, accepted)
	}
}

func TestASRResolveWhitespaceCannotChooseAReplacement(t *testing.T) {
	plan := (&SpellingLexicon{}).PrepareASR("alpha beta.", []string{"alphabeta"}, nil)
	if got, accepted := plan.Resolve("alphabeta."); accepted || got != "alpha beta." {
		t.Fatalf("whitespace-ambiguous Resolve() = (%q, %v)", got, accepted)
	}
}

func TestASRPlanShortlistAndReturnedTermsAreBounded(t *testing.T) {
	var entries []Replacement
	var terms, words []string
	for _, suffix := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"} {
		canonical, alias := "Target"+suffix, "source"+suffix
		entries = append(entries, Replacement{From: alias, To: canonical})
		terms = append(terms, canonical)
		words = append(words, alias)
	}
	plan := (&SpellingLexicon{}).PrepareASR(strings.Join(words, ", "), terms, NewVocabulary(entries))
	if got := plan.Terms(); !reflect.DeepEqual(got, terms[:8]) {
		t.Fatalf("Terms() = %q, want %q", got, terms[:8])
	}
	copy := plan.Terms()
	copy[0] = "mutated"
	if plan.Terms()[0] != terms[0] {
		t.Fatal("Terms exposed mutable plan storage")
	}
}

func TestASRResolveManyAlternativesUsesBoundedParsing(t *testing.T) {
	aliases := NewVocabulary([]Replacement{{From: "oldname", To: "NewName"}})
	plan := (&SpellingLexicon{}).PrepareASR(strings.Repeat("oldname, ", 100), []string{"NewName"}, aliases)
	got, accepted := plan.Resolve(strings.Repeat("NewName; ", 100))
	if !accepted || got != strings.Repeat("NewName, ", 100) {
		t.Fatalf("repeated alternatives Resolve() accepted=%v", accepted)
	}
}
