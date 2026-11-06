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
	DiagnosticAudioWait    DiagnosticStage = "audio_device_wait"
	DiagnosticAudioContext DiagnosticStage = "audio_context_init"
	DiagnosticAudioSelect  DiagnosticStage = "audio_device_select"
	DiagnosticAudioDevice  DiagnosticStage = "audio_device_init"
	DiagnosticAudioReuse   DiagnosticStage = "audio_device_reuse"
	DiagnosticAudioStart   DiagnosticStage = "audio_device_start"
	DiagnosticAudioStop    DiagnosticStage = "audio_stop"
	DiagnosticFinish       DiagnosticStage = "recognizer_finish"
	DiagnosticFinalize     DiagnosticStage = "acoustic_finalize"
	DiagnosticContext      DiagnosticStage = "context_read"
	DiagnosticRewrite      DiagnosticStage = "rewrite_final"
	DiagnosticCommit       DiagnosticStage = "target_commit"
)

// Diagnostics is a content-free observer, separate from product events that
// carry drafts and target text.
type Diagnostics interface{ Observe(Diagnostic) }
