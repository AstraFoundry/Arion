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
	Code Code
	err  error
}

func (e *Error) Error() string { return "cloud polish: " + string(e.Code) }
func (e *Error) Unwrap() error { return e.err }

func fail(code Code, err error) error { return &Error{Code: code, err: err} }

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
	err    error
}

func (e *completionError) Error() string         { return "cloud polish: " + e.detail }
func (e *completionError) Unwrap() error         { return e.err }
func (e *completionError) FailureDetail() string { return e.detail }

// classifyCompletion refines a completion failure into the categories a user
// can act on. Anything else keeps the generic classification.
func classifyCompletion(err error) error {
	var r *republic.Error
	if err == nil || !errors.As(err, &r) {
		return err
	}
	switch {
	case reasoningMandatory(err):
		return &completionError{domain.PolishReasoningRequired, err}
	case r.Kind == republic.ErrAuthentication || r.StatusCode == http.StatusUnauthorized:
		return fail(CodeRejected, err)
	case r.StatusCode == http.StatusPaymentRequired:
		return &completionError{domain.PolishQuotaExhausted, err}
	case r.StatusCode == http.StatusTooManyRequests:
		return &completionError{domain.PolishRateLimited, err}
	case r.StatusCode == http.StatusNotFound:
		return &completionError{domain.PolishModelNotFound, err}
	case r.StatusCode >= 500:
		return &completionError{domain.PolishModelUnavailable, err}
	}
	return err
}
