package cloudpolish

import "errors"

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
	CodeNetwork  Code = "NETWORK"
	CodeStore    Code = "SECURE_STORE"
	CodeModels   Code = "MODELS"
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
