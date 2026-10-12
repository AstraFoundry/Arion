// Package ports defines domain-owned cancellable boundaries for local adapters.
package ports

import (
	"arion.local/core/contracts"
	"context"
)

// Recognizer instances belong to one session; Close releases native resources.
// Partial/Finish must return snapshots tagged with the original identity.
type Recognizer interface {
	Accept(context.Context, contracts.Frame) error
	Partial(context.Context) (contracts.Draft, error)
	Finish(context.Context) (contracts.Draft, error)
	Close() error
}

type Corrector interface {
	Correct(context.Context, contracts.CorrectionRequest) (contracts.CorrectionResult, error)
}

type CandidateRanker interface {
	Rank(context.Context, contracts.RankRequest) (contracts.RankResult, error)
}

type InputTarget interface {
	Capabilities(context.Context, contracts.TargetToken) (contracts.Capabilities, error)
	ReadContext(context.Context, contracts.Identity, contracts.TargetToken) (contracts.ContextSnapshot, error)
	Commit(context.Context, contracts.CommitRequest) (contracts.CommitResult, error)
}

// CompositionTarget owns only the host range associated with this session.
// Implementations require the OS host's edit/thread protocol.
type CompositionTarget interface {
	Update(context.Context, contracts.TargetToken, contracts.Draft) error
	Cancel(context.Context, contracts.Identity, contracts.TargetToken) error
}
