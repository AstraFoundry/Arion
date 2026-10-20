package domain

import (
	"math"
	"testing"
)

func TestSessionCannotReviveOrSkipReview(t *testing.T) {
	for _, state := range []SessionState{Completed, Cancelled, Failed} {
		for _, next := range []SessionState{Preparing, Capturing, Committing, Cancelled, Failed} {
			if CanTransition(state, next) {
				t.Errorf("terminal %s revived as %s", state, next)
			}
		}
	}
	if _, err := Transition(Finalizing, Committing); err == nil {
		t.Fatal("commit skipped review")
	}
	for _, state := range []SessionState{Preparing, Capturing, Finalizing, Reviewing, Committing} {
		if !CanTransition(state, Cancelled) {
			t.Errorf("cannot cancel %s", state)
		}
	}
	state := Idle
	for _, next := range []SessionState{Preparing, Capturing, Finalizing, Reviewing, Committing, Completed} {
		var err error
		state, err = Transition(state, next)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestRankingPreservesProducerCandidates(t *testing.T) {
	candidates := []Candidate{{ID: "a", Text: "Arion", Source: "dictionary"}, {ID: "b", Text: "Orion", Source: "asr"}}
	if err := ValidateRanking(candidates, []RankedCandidate{{ID: "b"}, {ID: "a"}}); err != nil {
		t.Fatal(err)
	}
	nan := math.NaN()
	cases := [][]RankedCandidate{
		{{ID: "a"}},
		{{ID: "a"}, {ID: "a"}},
		{{ID: "a"}, {ID: "new"}},
		{{ID: "a", Score: &nan}, {ID: "b"}},
	}
	for _, ranked := range cases {
		if err := ValidateRanking(candidates, ranked); err == nil {
			t.Errorf("accepted invalid ranking: %#v", ranked)
		}
	}
	if err := ValidateRanking([]Candidate{{ID: "a", Text: "A", Source: "asr"}, {ID: "a", Text: "B", Source: "asr"}}, []RankedCandidate{{ID: "a"}, {ID: "b"}}); err == nil {
		t.Fatal("accepted duplicate producer IDs")
	}
}
