package ports

import (
	"time"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

// Diagnostic reports a pipeline boundary without carrying content. Observers
// must return immediately; inference and capture must never wait for disk I/O.
type Diagnostic struct {
	QueueOverflow bool
	QueueDepth    int
	Verdict       domain.RewriteVerdict
	Identity      contracts.Identity
	Stage         DiagnosticStage
	Duration      time.Duration
	Failure       domain.ErrorCode
}

type DiagnosticStage string

const (
	DiagnosticSegmentQueue DiagnosticStage = "segment_queue"
	DiagnosticValidation   DiagnosticStage = "rewrite_validation"
	DiagnosticTarget       DiagnosticStage = "target_capture"
	DiagnosticRecognizer   DiagnosticStage = "recognizer_create"
	DiagnosticCapture      DiagnosticStage = "audio_start"
	DiagnosticFinalize     DiagnosticStage = "acoustic_finalize"
	DiagnosticContext      DiagnosticStage = "context_read"
	DiagnosticPreview      DiagnosticStage = "rewrite_preview"
	DiagnosticRewrite      DiagnosticStage = "rewrite_final"
	DiagnosticCommit       DiagnosticStage = "target_commit"
)

// Diagnostics is a content-free observer, separate from product events that
// carry drafts and target text.
type Diagnostics interface{ Observe(Diagnostic) }
