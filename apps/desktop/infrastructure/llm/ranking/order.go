package ranking

import (
	"context"
	"errors"

	"arion.local/core/contracts"
	"arion.local/core/domain"
	"arion.local/core/domain/ports"
)

// Order compares every eligible original and proposal using the same scorer.
func Order(ctx context.Context, scorer ports.KeyboardScorer, req contracts.KeyboardRankRequest, proposal string) ([]string, error) {
	if _, err := Prepare(req); err != nil {
		return nil, err
	}
	candidates := make([]domain.Candidate, 0, len(req.Candidates)+1)
	for _, c := range req.Candidates {
		if domain.KeyboardRankEligible(req.Candidates[0], c) {
			candidates = append(candidates, c.Candidate)
		}
	}
	if proposal != "" {
		candidates = append(candidates, domain.Candidate{ID: ProposalID(req), Text: proposal})
	}
	if len(candidates) == 1 {
		return []string{candidates[0].ID}, ctx.Err()
	}
	if scorer == nil {
		return nil, errors.New("keyboard comparison scorer unavailable")
	}
	texts := make([]string, len(candidates))
	for i, c := range candidates {
		texts[i] = c.Text
	}
	scores, err := scorer.ScoreKeyboard(ctx, req.Before, req.After, texts)
	if err != nil {
		return nil, err
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	order, err := domain.CompareKeyboardScores(scores, len(candidates))
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(order))
	for i, index := range order {
		ids[i] = candidates[index].ID
	}
	return ids, nil
}

// ProposalID cannot collide with a producer-owned identity.
func ProposalID(req contracts.KeyboardRankRequest) string {
	id := "proposal"
	for {
		collision := false
		for _, c := range req.Candidates {
			if c.ID == id {
				collision = true
				break
			}
		}
		if !collision {
			return id
		}
		id += "_"
	}
}
