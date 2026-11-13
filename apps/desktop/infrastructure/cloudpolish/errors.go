package cloudpolish

import (
	"errors"
	"net/http"

	"arion.local/core/domain"
	"github.com/AstraFoundry/republic"
)

// Code is a content-free failure the UI explains.
type Code string

const (
	CodeSignIn         Code = "SIGN_IN"
	CodeCancelled      Code = "CANCELLED"
	CodeTimeout        Code = "TIMEOUT"
	CodePortInUse      Code = "PORT_IN_USE"
	CodeNoSubscription Code = "NO_SUBSCRIPTION"
	// CodeRejected: the provider refused the key or the sign-in expired.
	CodeRejected Code = "REJECTED"
	// CodeAddress: a compatible provider's base URL is missing or not http(s).
	CodeAddress Code = "ADDRESS"
	CodeNetwork Code = "NETWORK"
	CodeStore   Code = "SECURE_STORE"
	CodeModels  Code = "MODELS"
)

// Error carries a Code and its cause, which may hold provider text and is
// never shown or logged.
type Error struct {
	Code   Code
	reason string
	err    error
}

func (e *Error) Error() string { return "cloud polish: " + string(e.Code) }
func (e *Error) Unwrap() error { return e.err }

func fail(code Code, err error) error { return &Error{Code: code, err: err} }

func failReason(code Code, reason string, err error) error {
	return &Error{Code: code, reason: reason, err: err}
}

// DiagnosticReason returns the closed, content-free failure reason for logs.
func (e *Error) DiagnosticReason() string {
	if e.reason != "" {
		return e.reason
	}
	switch e.Code {
	case CodeRejected:
		return domain.ReasonCloudRejected
	case CodeNetwork:
		return domain.ReasonCloudNetwork
	case CodeModels:
		return domain.ReasonCloudModelLookup
	case CodeTimeout:
		return domain.ReasonCloudTimeout
	case CodeCancelled:
		return domain.ReasonRewriteCancelled
	}
	return ""
}

// DiagnosticCode maps a cloud failure to a 004 error code for diagnostics.
func (e *Error) DiagnosticCode() domain.ErrorCode {
	switch e.Code {
	case CodeCancelled:
		return domain.CodeCancelled
	case CodeTimeout:
		return domain.CodeRewriteTimeout
	case CodeModels:
		return domain.CodeModelMissing
	case CodeRejected, CodeNetwork, CodeSignIn, CodeNoSubscription, CodeAddress:
		return domain.CodeRewriteRejected
	}
	return ""
}

// CodeOf returns err's Code, or "" when it has none.
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// FailureDetail is the content-free category the UI explains, or "" when the
// code has no actionable explanation.
func (e *Error) FailureDetail() string {
	switch e.Code {
	case CodeRejected:
		return domain.PolishCredentialRejected
	case CodeTimeout:
		return domain.PolishTimeout
	}
	return ""
}

// completionError is a completion failure with a category the user can act
// on. It is shown only as a note on a rewrite, never as a connection status.
type completionError struct {
	detail string
	reason string
	code   domain.ErrorCode
	err    error
}

func (e *completionError) Error() string {
	if e.detail != "" {
		return "cloud polish: " + e.detail
	}
	return "cloud polish: " + e.reason
}
func (e *completionError) Unwrap() error            { return e.err }
func (e *completionError) FailureDetail() string    { return e.detail }
func (e *completionError) DiagnosticReason() string { return e.reason }
func (e *completionError) DiagnosticCode() domain.ErrorCode {
	if e.code != "" {
		return e.code
	}
	return domain.CodeRewriteRejected
}

// classifyCompletion refines a completion failure into the categories a user
// can act on and closed diagnostic reasons. Anything else keeps the generic
// classification.
func classifyCompletion(err error) error {
	var r *republic.Error
	if err == nil || !errors.As(err, &r) {
		return err
	}
	switch {
	case reasoningMandatory(err):
		return &completionError{detail: domain.PolishReasoningRequired, reason: domain.ReasonCloudReasoningRequired, code: domain.CodeModelUnsupported, err: err}
	case r.Kind == republic.ErrAuthentication || r.StatusCode == http.StatusUnauthorized:
		return failReason(CodeRejected, domain.ReasonCloudRejected, err)
	case r.StatusCode == http.StatusForbidden:
		return &completionError{reason: domain.ReasonCloudForbidden, code: domain.CodeRewriteRejected, err: err}
	case r.StatusCode == http.StatusPaymentRequired:
		return &completionError{detail: domain.PolishQuotaExhausted, reason: domain.ReasonCloudQuotaExhausted, code: domain.CodeRewriteRejected, err: err}
	case r.StatusCode == http.StatusTooManyRequests:
		return &completionError{detail: domain.PolishRateLimited, reason: domain.ReasonCloudRateLimited, code: domain.CodeRewriteRejected, err: err}
	case r.StatusCode == http.StatusNotFound:
		return &completionError{detail: domain.PolishModelNotFound, reason: domain.ReasonCloudModelNotFound, code: domain.CodeModelMissing, err: err}
	case r.StatusCode >= 400 && r.StatusCode < 500:
		return &completionError{reason: domain.ReasonCloudBadRequest, code: domain.CodeRewriteRejected, err: err}
	case r.StatusCode >= 500:
		return &completionError{detail: domain.PolishModelUnavailable, reason: domain.ReasonCloudServerError, code: domain.CodeEvaluationUnavailable, err: err}
	case r.Kind == republic.ErrConnection:
		return failReason(CodeNetwork, domain.ReasonCloudConnection, err)
	case r.Kind == republic.ErrTimeout:
		return &completionError{reason: domain.ReasonCloudTimeout, code: domain.CodeRewriteTimeout, err: err}
	case r.Kind == republic.ErrAPIResponse || r.Kind == republic.ErrStreamIncomplete || r.Kind == republic.ErrStreamNotFinished:
		return &completionError{reason: domain.ReasonCloudResponseInvalid, code: domain.CodeRewriteRejected, err: err}
	case r.Kind == republic.ErrProviderNotFound || r.Kind == republic.ErrUnsupportedFormat || r.Kind == republic.ErrUnsupportedFeature || r.Kind == republic.ErrClosed:
		return &completionError{reason: domain.ReasonCloudUnsupported, code: domain.CodeModelUnsupported, err: err}
	}
	return err
}
