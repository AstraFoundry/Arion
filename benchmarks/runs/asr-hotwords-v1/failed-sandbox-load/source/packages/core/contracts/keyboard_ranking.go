package contracts

import "arion.local/core/domain"

// KeyboardRankRequest is a reference-free snapshot, independent of dictation.
// Before contains only approved context; it never authorizes a host-text read.
type KeyboardRankRequest struct {
	Identity
	Revision   uint64                     `json:"revision"`
	Input      string                     `json:"input"`
	Before     string                     `json:"before"`
	Candidates []domain.KeyboardCandidate `json:"candidates"`
}

type KeyboardRankResult struct {
	Identity
	Revision   uint64                   `json:"revision"`
	Candidates []domain.RankedCandidate `json:"candidates"`
	Template   string                   `json:"template"`
	Decision   string                   `json:"decision"`
}
