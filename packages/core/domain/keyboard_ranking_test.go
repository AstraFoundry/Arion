package domain

import "testing"

func TestKeyboardPromotionRequiresActualInputRange(t *testing.T) {
	first := KeyboardCandidate{Candidate: Candidate{ID: "first", Text: "现", Source: "rime"}, End: 4}
	full := KeyboardCandidate{Candidate: Candidate{ID: "full", Text: "西安", Source: "rime"}, End: 4}
	prefix := KeyboardCandidate{Candidate: Candidate{ID: "prefix", Text: "西", Source: "rime"}, End: 2}
	for _, test := range []struct {
		name      string
		candidate KeyboardCandidate
		eligible  bool
	}{
		{"different-length-full-conversion", full, true},
		{"equal-length-prefix", prefix, false},
		{"different-start", KeyboardCandidate{Candidate: full.Candidate, Start: 2, End: 4}, false},
		{"unknown-range", KeyboardCandidate{Candidate: full.Candidate}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := KeyboardRankEligible(first, test.candidate); got != test.eligible {
				t.Fatalf("eligibility = %v", got)
			}
		})
	}
	first.End, full.End = 0, 0
	if KeyboardRankEligible(first, full) {
		t.Fatal("unknown ranges permit promotion")
	}
	if _, err := PromoteKeyboardCandidate([]KeyboardCandidate{first, full}, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := PromoteKeyboardCandidate([]KeyboardCandidate{first, full}, full.ID); err == nil {
		t.Fatal("unknown-range promotion accepted")
	}
}
