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
