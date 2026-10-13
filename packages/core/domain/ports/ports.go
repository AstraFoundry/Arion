// Package ports defines domain-owned cancellable boundaries for local adapters.
package ports

import (
	"context"

	"arion.local/core/contracts"
)

// FrameSink receives captured frames. Implementations only enqueue (SPH-02).
type FrameSink func(contracts.Frame)

// AudioCapture opens the selected input device for one session.
type AudioCapture interface {
	Start(ctx context.Context, id contracts.Identity, sink FrameSink) (AudioStream, error)
}

// AudioStream is one active capture. Stop delivers every remaining frame to
// the sink before returning and releases the device.
type AudioStream interface {
	Stop() error
}

// Recognizer instances belong to one session; Close releases native resources.
// Partial/Finish must return snapshots tagged with the original identity.
type Recognizer interface {
	Accept(context.Context, contracts.Frame) error
	Partial(context.Context) (contracts.Draft, error)
	Finish(context.Context) (contracts.Draft, error)
	Close() error
}

// RecognizerFactory creates per-session recognizers from a resident engine.
// It returns MODEL_MISSING while no verified model is loaded.
type RecognizerFactory interface {
	NewRecognizer(context.Context, contracts.Identity) (Recognizer, error)
}

// FinalizeRequest carries the bounded session audio retained in RAM for an
// optional acoustic second pass.
type FinalizeRequest struct {
	contracts.Identity
	SampleRate int
	Samples    []float32
	Draft      string
}

// Finalizer re-decodes retained audio after the user stops. It must not
// change identity or revision ownership.
type Finalizer interface {
	Finalize(context.Context, FinalizeRequest) (string, error)
}

type Corrector interface {
	Correct(context.Context, contracts.CorrectionRequest) (contracts.CorrectionResult, error)
}

type CandidateRanker interface {
	Rank(context.Context, contracts.RankRequest) (contracts.RankResult, error)
}

// TargetProvider captures the focused editor when dictation starts. It must
// return PERMISSION_DENIED without reading anything when untrusted.
type TargetProvider interface {
	Capture(context.Context, contracts.Identity) (contracts.TargetSnapshot, error)
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
