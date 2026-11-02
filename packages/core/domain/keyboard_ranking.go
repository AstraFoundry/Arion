package domain

import (
	"errors"
	"math"
	"sort"
)

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

// OrderKeyboardCandidates replaces only comparable slots and retains every engine ID.
func OrderKeyboardCandidates(candidates []KeyboardCandidate, ids []string) ([]RankedCandidate, error) {
	if len(candidates) == 0 {
		return nil, errors.New("empty keyboard candidate set")
	}
	eligible := make(map[string]bool)
	for _, c := range candidates {
		if KeyboardRankEligible(candidates[0], c) {
			eligible[c.ID] = true
		}
	}
	if len(ids) != len(eligible) {
		return nil, errors.New("incomplete keyboard order")
	}
	for _, id := range ids {
		if !eligible[id] {
			return nil, errors.New("invalid keyboard order")
		}
		eligible[id] = false
	}
	out := make([]RankedCandidate, len(candidates))
	plain := make([]Candidate, len(candidates))
	next := 0
	for i, c := range candidates {
		plain[i] = c.Candidate
		out[i] = RankedCandidate{ID: c.ID}
		if KeyboardRankEligible(candidates[0], c) {
			out[i].ID = ids[next]
			next++
		}
	}
	return out, ValidateRanking(plain, out)
}

// KeyboardPromotionMargin is a development-calibrated mean log probability gap.
const KeyboardPromotionMargin = 0.2

// CompareKeyboardScores preserves the original leader when evidence is close.
func CompareKeyboardScores(scores []float64, count int) ([]int, error) {
	if count == 0 || len(scores) != count {
		return nil, errors.New("incomplete keyboard comparison scores")
	}
	order := make([]int, count)
	for i, score := range scores {
		if math.IsNaN(score) || math.IsInf(score, 0) {
			return nil, errors.New("nonfinite keyboard score")
		}
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool { return scores[order[i]] > scores[order[j]] })
	if scores[order[0]]-scores[0] <= KeyboardPromotionMargin {
		for i, index := range order {
			if index == 0 {
				copy(order[1:i+1], order[:i])
				order[0] = 0
				break
			}
		}
	}
	return order, nil
}
