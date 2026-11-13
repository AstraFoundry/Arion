package domain

import (
	"errors"
	"fmt"
)

// ErrorCode is a stable, content-free failure category from the 004 contract.
type ErrorCode string

// DetailInputMethodClientOutdated identifies an editor that still has the
// previous native input-method build loaded after an application upgrade.
const DetailInputMethodClientOutdated = "input method client build outdated"

const (
	DetailInputMethodArtifactUnverified  = "input method artifact unverified"
	DetailInputMethodProtocolUnsupported = "input method protocol unsupported"
)

// Context read details are closed diagnostic categories, never field content.
const (
	DetailContextWindowNodes   = "context_window_nodes"
	DetailContextWindowDepth   = "context_window_depth"
	DetailContextWindowText    = "context_window_text"
	DetailContextWindowRange   = "context_window_range"
	DetailContextWindowFrame   = "context_window_frame"
	DetailContextWindowEmpty   = "context_window_empty"
	DetailContextWindowTimeout = "context_window_timeout"
)

// Polish failure details are content-free categories of a rewrite that fell
// back to local cleanup. The UI explains each with its next step.
const (
	PolishReasoningRequired  = "reasoning required"
	PolishRateLimited        = "rate limited"
	PolishQuotaExhausted     = "quota exhausted"
	PolishCredentialRejected = "credential rejected"
	PolishModelUnavailable   = "model unavailable"
	PolishModelNotFound      = "model not found"
	PolishTimeout            = "timeout"
)

// Detailer is implemented by errors that carry a content-free failure detail.
type Detailer interface{ FailureDetail() string }

// DetailOf returns the failure detail err carries, or "".
func DetailOf(err error) string {
	var d Detailer
	if errors.As(err, &d) {
		return d.FailureDetail()
	}
	return ""
}

// DiagnosticPhaser is implemented by errors that identify the closed
// pipeline phase where a failure occurred (PRV-05).
type DiagnosticPhaser interface{ DiagnosticPhase() string }

// PhaseOf returns the closed diagnostic phase err carries, or "".
func PhaseOf(err error) string {
	var p DiagnosticPhaser
	if errors.As(err, &p) {
		return p.DiagnosticPhase()
	}
	return ""
}

// DiagnosticReasoner is implemented by errors that identify a closed,
// content-free failure step or category for local diagnostics (PRV-05).
type DiagnosticReasoner interface{ DiagnosticReason() string }

// ReasonOf returns the closed diagnostic reason err carries, or "".
func ReasonOf(err error) string {
	var r DiagnosticReasoner
	if errors.As(err, &r) {
		return r.DiagnosticReason()
	}
	return ""
}

// DiagnosticCoder is implemented by infrastructure errors that carry a
// domain ErrorCode for diagnostics without wrapping *domain.Error directly.
type DiagnosticCoder interface{ DiagnosticCode() ErrorCode }

// DiagnosticCodeOf returns CodeOf(err), falling back to DiagnosticCoder.
func DiagnosticCodeOf(err error) ErrorCode {
	if code := CodeOf(err); code != "" {
		return code
	}
	var c DiagnosticCoder
	if errors.As(err, &c) {
		return c.DiagnosticCode()
	}
	return ""
}

// Closed diagnostic phases for pipeline stages (PRV-05).
const (
	PhaseSetup       = "setup"
	PhasePreparation = "preparation"
	PhaseSelection   = "selection"
	PhaseFormatting  = "formatting"
	PhaseWorker      = "worker"
)

// Closed rewrite diagnostic reasons identify the exact failing step or
// transport category without logging content, URLs, keys or raw errors (PRV-05).
const (
	ReasonRewritePipelineLoad     = "rewrite_pipeline_load"
	ReasonRewriteControlInput     = "rewrite_control_input"
	ReasonRewritePreparation      = "rewrite_preparation"
	ReasonRewritePromptBuild      = "rewrite_prompt_build"
	ReasonRewriteTruncated        = "rewrite_truncated"
	ReasonRewriteTimeout          = "rewrite_timeout"
	ReasonRewriteCancelled        = "rewrite_cancelled"
	ReasonSpellingCoverageInvalid = "spelling_coverage_invalid"
	ReasonSpellingChoiceInvalid   = "spelling_choice_invalid"
	ReasonSpellingSpanMismatch    = "spelling_span_mismatch"

	ReasonCloudPrefillUnsupported = "cloud_prefill_unsupported"
	ReasonCloudModelMissing       = "cloud_model_missing"
	ReasonCloudNotConnected       = "cloud_not_connected"
	ReasonCloudTokenRejected      = "cloud_token_rejected"
	ReasonCloudTokenNetwork       = "cloud_token_network"
	ReasonCloudTokenInvalid       = "cloud_token_invalid"
	ReasonCloudProviderOpen       = "cloud_provider_open_failed"
	ReasonCloudModelLookup        = "cloud_model_lookup"
	ReasonCloudRoleUnsupported    = "cloud_role_unsupported"
	ReasonCloudInvalidUTF8        = "cloud_invalid_utf8"
	ReasonCloudReasoningRequired  = "cloud_reasoning_required"
	ReasonCloudRejected           = "cloud_rejected"
	ReasonCloudForbidden          = "cloud_forbidden"
	ReasonCloudQuotaExhausted     = "cloud_quota_exhausted"
	ReasonCloudRateLimited        = "cloud_rate_limited"
	ReasonCloudModelNotFound      = "cloud_model_not_found"
	ReasonCloudBadRequest         = "cloud_bad_request"
	ReasonCloudServerError        = "cloud_server_error"
	ReasonCloudConnection         = "cloud_connection"
	ReasonCloudTimeout            = "cloud_timeout"
	ReasonCloudResponseInvalid    = "cloud_response_invalid"
	ReasonCloudUnsupported        = "cloud_unsupported"
	ReasonCloudNetwork            = "cloud_network"

	ReasonWorkerMissing     = "worker_missing"
	ReasonWorkerUnsupported = "worker_unsupported"
	ReasonWorkerStart       = "worker_start_failed"
	ReasonWorkerWrite       = "worker_write_failed"
	ReasonWorkerExited      = "worker_exited"
	ReasonWorkerFailed      = "worker_failed"
)

// DetailTranslationModel identifies a voice-translation session whose model
// is missing or failed to run.
const DetailTranslationModel = "translation model"

const (
	CodePermissionDenied       ErrorCode = "PERMISSION_DENIED"
	CodeModelMissing           ErrorCode = "MODEL_MISSING"
	CodeModelUnsupported       ErrorCode = "MODEL_UNSUPPORTED"
	CodeAudioDiscontinuity     ErrorCode = "AUDIO_DISCONTINUITY"
	CodeTargetChanged          ErrorCode = "TARGET_CHANGED"
	CodeTargetUnsupported      ErrorCode = "TARGET_UNSUPPORTED"
	CodeStaleRevision          ErrorCode = "STALE_REVISION"
	CodeCancelled              ErrorCode = "CANCELLED"
	CodeInsertionUncertain     ErrorCode = "INSERTION_UNCERTAIN"
	CodeClipboardUnavailable   ErrorCode = "CLIPBOARD_UNAVAILABLE"
	CodeClipboardRestoreFailed ErrorCode = "CLIPBOARD_RESTORE_FAILED"
	CodeSyncUnauthorized       ErrorCode = "SYNC_UNAUTHORIZED"
	CodeSyncConflict           ErrorCode = "SYNC_CONFLICT"
	CodeContractUnsupported    ErrorCode = "CONTRACT_UNSUPPORTED"
	CodeDeviceUnavailable      ErrorCode = "DEVICE_UNAVAILABLE"
	CodeInvalidTransition      ErrorCode = "INVALID_TRANSITION"
	CodeSessionBusy            ErrorCode = "SESSION_BUSY"
	CodeNoSpeech               ErrorCode = "NO_SPEECH"
	CodeArtifactUnverified     ErrorCode = "ARTIFACT_UNVERIFIED"
	CodeProvisioningFailed     ErrorCode = "PROVISIONING_FAILED"
	CodeSecureFieldBlocked     ErrorCode = "SECURE_FIELD_BLOCKED"
	CodeEvaluationUnavailable  ErrorCode = "EVALUATION_UNAVAILABLE"
	CodeRewriteRejected        ErrorCode = "REWRITE_REJECTED"
	CodeRewriteTimeout         ErrorCode = "REWRITE_TIMEOUT"
)

// Error carries a stable code and a diagnostic category. Detail must never
// contain transcripts, context, vocabulary, paths or secrets (PRV-05).
type Error struct {
	Code   ErrorCode
	Detail string
}

func (e *Error) Error() string {
	if e.Detail == "" {
		return string(e.Code)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Detail)
}

// Is matches errors by code so callers can use errors.Is with a sentinel.
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && other.Code == e.Code
}

// Fail builds a coded error.
func Fail(code ErrorCode, detail string) *Error { return &Error{Code: code, Detail: detail} }

// CodeOf extracts a stable code from an error chain, or "" when uncoded.
func CodeOf(err error) ErrorCode {
	var coded *Error
	if errors.As(err, &coded) && coded != nil {
		return coded.Code
	}
	return ""
}

type diagnosticError struct {
	code   ErrorCode
	phase  string
	reason string
	err    error
}

func (e *diagnosticError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	if e.reason != "" {
		return e.reason
	}
	return string(e.code)
}

func (e *diagnosticError) Unwrap() error { return e.err }

func (e *diagnosticError) DiagnosticCode() ErrorCode {
	if e.code != "" {
		return e.code
	}
	return DiagnosticCodeOf(e.err)
}

func (e *diagnosticError) DiagnosticPhase() string {
	if e.phase != "" {
		return e.phase
	}
	return PhaseOf(e.err)
}

func (e *diagnosticError) DiagnosticReason() string {
	if e.reason != "" {
		return e.reason
	}
	return ReasonOf(e.err)
}

// WithDiagnostic wraps err with closed diagnostic code, phase and reason values
// without altering its message or inner error chain.
func WithDiagnostic(code ErrorCode, phase, reason string, err error) error {
	if err == nil && code == "" && phase == "" && reason == "" {
		return nil
	}
	return &diagnosticError{code: code, phase: phase, reason: reason, err: err}
}

// WithPhase attaches a closed pipeline phase to err while preserving any inner
// diagnostic code and reason.
func WithPhase(phase string, err error) error {
	if err == nil {
		return nil
	}
	return &diagnosticError{code: DiagnosticCodeOf(err), phase: phase, reason: ReasonOf(err), err: err}
}
