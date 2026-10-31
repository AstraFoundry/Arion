package contracts

import "arion.local/core/domain"

// KeyboardRankRequest is a reference-free snapshot, independent of dictation.
// Before contains only approved context; it never authorizes a host-text read.
type KeyboardRankRequest struct {
	Identity
	AllowCompletion bool                       `json:"allowCompletion,omitempty"`
	Revision        uint64                     `json:"revision"`
	Application     string                     `json:"-"`
	Preedit         string                     `json:"preedit,omitempty"`
	Input           string                     `json:"input"`
	Before          string                     `json:"before"`
	After           string                     `json:"-"`
	Screen          string                     `json:"-"`
	Candidates      []domain.KeyboardCandidate `json:"candidates"`
}

type KeyboardRankResult struct {
	Completion *domain.KeyboardCompletion `json:"completion,omitempty"`
	Identity
	Revision   uint64                   `json:"revision"`
	Candidates []domain.RankedCandidate `json:"candidates"`
	Template   string                   `json:"template"`
	Decision   string                   `json:"decision"`
}
