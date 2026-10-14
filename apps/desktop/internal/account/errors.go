package account

import (
	"errors"
	"fmt"
)

// Code is a stable, content-free failure category. Detail is an OAuth error
// code, an HTTP status label, or another short token that is not a secret.
type Code string

const (
	CodeInProgress    Code = "SIGN_IN_IN_PROGRESS"
	CodeTimeout       Code = "SIGN_IN_TIMEOUT"
	CodeCancelled     Code = "SIGN_IN_CANCELLED"
	CodeStateMismatch Code = "STATE_MISMATCH"
	CodeAuthorization Code = "AUTHORIZATION"
	CodeToken         Code = "TOKEN"
	CodeDiscovery     Code = "DISCOVERY"
	CodeStore         Code = "SECURE_STORE"
	CodeNotSignedIn   Code = "NOT_SIGNED_IN"
	CodeRefresh       Code = "REFRESH"
	CodeRevoke        Code = "REVOKE"
	CodeWaitlist      Code = "WAITLIST"
	CodeEntitlement   Code = "ENTITLEMENT"
	CodeBrowser       Code = "BROWSER"
)

// Error is the failure type methods return. The Error string never includes
// tokens, authorization codes, or OAuth error_description text.
type Error struct {
	Code   Code
	Detail string
	err    error
}

func (e *Error) Error() string {
	switch {
	case e == nil:
		return ""
	case e.Detail == "" && e.err == nil:
		return string(e.Code)
	case e.Detail == "":
		return fmt.Sprintf("%s: %s", e.Code, e.err.Error())
	case e.err == nil:
		return fmt.Sprintf("%s: %s", e.Code, e.Detail)
	default:
		return fmt.Sprintf("%s: %s: %s", e.Code, e.Detail, e.err.Error())
	}
}

func (e *Error) Unwrap() error { return e.err }

// Is matches another *Error with the same code. A target detail, when set,
// must match too.
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	if !ok || other.Code != e.Code {
		return false
	}
	return other.Detail == "" || other.Detail == e.Detail
}

func wrap(code Code, detail string, err error) error {
	return &Error{Code: code, Detail: detail, err: err}
}

// Wrap attaches a cause to a content-free code. Callers that show the failure
// use CodeOf and DetailOf; the cause is for logs at the composition root.
func Wrap(code Code, detail string, err error) error {
	return wrap(code, detail, err)
}

// CodeOf returns the account code in err, or "" when err is not an *Error.
func CodeOf(err error) Code {
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	return ""
}

// DetailOf returns the content-free detail in err.
func DetailOf(err error) string {
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Detail
	}
	return ""
}

var (
	// ErrSignInInProgress is returned when a second sign-in or refresh starts
	// while one is still running.
	ErrSignInInProgress = &Error{Code: CodeInProgress}
	// ErrSignInTimeout is the browser wait exceeding SignInTimeout.
	ErrSignInTimeout = &Error{Code: CodeTimeout}
	// ErrSignInCancelled is a caller cancel or Service.Cancel. Stale exchange
	// results are discarded.
	ErrSignInCancelled = &Error{Code: CodeCancelled}
	// ErrStateMismatch means the callback state was not the one we generated.
	ErrStateMismatch = &Error{Code: CodeStateMismatch}
	// ErrNotSignedIn means a call needed an in-memory access token.
	ErrNotSignedIn = &Error{Code: CodeNotSignedIn}
	// ErrNotFound means the credential store has no refresh token.
	ErrNotFound = &Error{Code: CodeStore, Detail: "not_found"}
	// ErrUnsupported means this OS has no credential-store implementation.
	ErrUnsupported = &Error{Code: CodeStore, Detail: "unsupported"}
	// ErrRefreshTokenTooLarge means the token does not fit in Windows
	// Credential Manager.
	ErrRefreshTokenTooLarge = &Error{Code: CodeStore, Detail: "refresh_token_too_large"}

	errBadURL = errors.New("url must be http or https")
)
