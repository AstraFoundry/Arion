package domain

import (
	"fmt"
	"math"
)

type Candidate struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Source string `json:"source"`
}

type RankedCandidate struct {
	ID    string   `json:"id"`
	Score *float64 `json:"score,omitempty"`
}

// ValidateRanking permits reordering only; the producer retains text ownership.
func ValidateRanking(candidates []Candidate, ranking []RankedCandidate) error {
	if len(candidates) == 0 || len(candidates) != len(ranking) {
		return fmt.Errorf("ranking must preserve a nonempty candidate set")
	}
	remaining := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if candidate.ID == "" || candidate.Text == "" || candidate.Source == "" || remaining[candidate.ID] {
			return fmt.Errorf("invalid or duplicate producer candidate: %q", candidate.ID)
		}
		remaining[candidate.ID] = true
	}
	for _, candidate := range ranking {
		if !remaining[candidate.ID] {
			return fmt.Errorf("unknown or duplicate ranked candidate: %q", candidate.ID)
		}
		if candidate.Score != nil && (math.IsNaN(*candidate.Score) || math.IsInf(*candidate.Score, 0)) {
			return fmt.Errorf("nonfinite ranking score: %q", candidate.ID)
		}
		delete(remaining, candidate.ID)
	}
	return nil
}
