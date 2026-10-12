// Package contracts defines portable boundary data, not a ready IPC encoding.
package contracts

import "arion.local/core/domain"

const Version = 1

type Identity struct {
	SessionID  string `json:"session_id"`
	Generation uint64 `json:"generation"`
}

type EventKind string

const (
	SessionStateChanged EventKind = "session_state"
	AudioDiscontinuity  EventKind = "audio_discontinuity"
	DraftReplaced       EventKind = "draft_replaced"
	CandidatesRanked    EventKind = "candidates_ranked"
	CorrectionReady     EventKind = "correction_ready"
	CommitResultReady   EventKind = "commit_result"
	ErrorOccurred       EventKind = "error"
)

type Envelope struct {
	Version int `json:"version"`
	Identity
	Sequence uint64    `json:"sequence"`
	Kind     EventKind `json:"kind"`
	Revision uint64    `json:"revision,omitempty"`
}

// Frame samples are interleaved float32 PCM in the adapter-declared format.
type Frame struct {
	Identity
	Sequence      uint64
	SampleOffset  uint64
	SampleRate    uint32
	Channels      uint16
	Samples       []float32
	Discontinuity bool
}

type Draft struct {
	Identity
	Revision   uint64             `json:"revision"`
	Text       string             `json:"text"`
	Final      bool               `json:"final"`
	Candidates []domain.Candidate `json:"candidates"`
}

type TargetToken string

type Capabilities struct {
	ContextRead        bool
	SingleInsertion    bool
	OwnedComposition   bool
	CandidateSelection bool
}

// ContextSnapshot offsets count Unicode scalar values in Text.
type ContextSnapshot struct {
	Identity
	Target         TargetToken
	Text           string
	SelectionStart uint32
	SelectionEnd   uint32
	SafeToRead     bool
}

type RankRequest struct {
	Draft        Draft
	Context      ContextSnapshot
	LanguageHint string
}

type RankResult struct {
	Identity
	Revision   uint64
	Candidates []domain.RankedCandidate
}

type CorrectionRequest struct {
	Draft   Draft
	Context ContextSnapshot
}

type CorrectionResult struct {
	Identity
	SourceRevision uint64
	Text           string
	RequiresReview bool
}

type CommitRequest struct {
	Identity
	Target         TargetToken
	SourceRevision uint64
	Text           string
}

type CommitOutcome string

const (
	Applied    CommitOutcome = "applied"
	NotApplied CommitOutcome = "not_applied"
	Unknown    CommitOutcome = "unknown"
)

type CommitResult struct {
	Identity
	Outcome CommitOutcome
}
