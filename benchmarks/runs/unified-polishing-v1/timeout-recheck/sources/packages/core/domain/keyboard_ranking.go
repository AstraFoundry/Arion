package domain

import (
	"errors"
	"unicode/utf8"
)

// KeyboardCandidate retains producer identity and optional consumed-input end.
// End is a UTF-8 byte offset in the raw Pinyin input; zero means unavailable.
type KeyboardCandidate struct {
	Candidate
	End int `json:"end,omitempty"`
}

// KeyboardRankEligible preserves conversion coverage before semantic scoring.
// Without producer spans, equal scalar length is a conservative compatibility
// proxy, not proof that two candidates consume the same input.
func KeyboardRankEligible(first, candidate KeyboardCandidate) bool {
	if first.End > 0 || candidate.End > 0 {
		return first.End > 0 && candidate.End == first.End
	}
	return utf8.RuneCountInString(first.Text) == utf8.RuneCountInString(candidate.Text)
}

// PromoteKeyboardCandidate returns a full stable permutation, never new text.
func PromoteKeyboardCandidate(candidates []KeyboardCandidate, pick string) ([]RankedCandidate, error) {
	if len(candidates) == 0 {
		return nil, errors.New("empty keyboard candidate set")
	}
	plain := make([]Candidate, len(candidates))
	order := make([]RankedCandidate, len(candidates))
	chosen := -1
	for i, c := range candidates {
		plain[i], order[i] = c.Candidate, RankedCandidate{ID: c.ID}
		if c.ID == pick {
			chosen = i
		}
	}
	if err := ValidateRanking(plain, order); err != nil {
		return nil, err
	}
	if chosen < 0 || !KeyboardRankEligible(candidates[0], candidates[chosen]) {
		return nil, errors.New("unknown or incompatible keyboard candidate")
	}
	selected := order[chosen]
	copy(order[1:chosen+1], order[:chosen])
	order[0] = selected
	return order, nil
}
