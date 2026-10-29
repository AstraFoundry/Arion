package domain

import "errors"

// KeyboardCandidate retains producer identity and consumed-input byte range.
// End zero means unavailable; unavailable ranges never authorize promotion.
type KeyboardCandidate struct {
	Candidate
	Start int `json:"start,omitempty"`
	End   int `json:"end,omitempty"`
}

// KeyboardRankEligible preserves conversion coverage before semantic scoring.
// Keeping the exact producer first choice is safe even without range metadata.
// Different text lengths may consume the same input (for example xian).
func KeyboardRankEligible(first, candidate KeyboardCandidate) bool {
	if first == candidate {
		return true
	}
	return first.Start >= 0 && first.End > first.Start &&
		candidate.Start == first.Start && candidate.End == first.End
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
