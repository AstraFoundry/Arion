package application

import "errors"

// Account-layer outcomes. The domain package does not define these.
const (
	CodeSessionRevoked      = "SESSION_REVOKED"
	CodeEntitlementRequired = "ENTITLEMENT_REQUIRED"
	CodeTenantFrozen        = "TENANT_FROZEN"
	CodeQuotaExceeded       = "QUOTA_EXCEEDED"
	CodeNotFound            = "NOT_FOUND"
)

var messages = map[string]string{
	CodeSessionRevoked:      "session revoked",
	CodeEntitlementRequired: "entitlement required",
	CodeTenantFrozen:        "tenant frozen",
	CodeQuotaExceeded:       "quota exceeded",
	CodeNotFound:            "not found",
}

// Error is a stable coded failure from the account gate or quota check.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && other.Code == e.Code
}

// Fail returns the coded error for a known application code.
func Fail(code string) *Error {
	return &Error{Code: code, Message: messages[code]}
}

// SETError is an RFC 8935 rejection. Description is content-free.
type SETError struct {
	Code        string
	Description string
}

func (e *SETError) Error() string { return e.Code }

// SET rejects a security event with an RFC 8935 error code.
func SET(code, description string) *SETError {
	return &SETError{Code: code, Description: description}
}

// AsSET reports a security-event rejection.
func AsSET(err error) (*SETError, bool) {
	var se *SETError
	if errors.As(err, &se) {
		return se, true
	}
	return nil, false
}
