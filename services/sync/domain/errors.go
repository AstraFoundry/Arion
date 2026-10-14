package domain

import "errors"

// Stable, content-free failure codes. Account and entitlement outcomes live in
// the application layer; this package does not know about accounts.
const (
	CodeInvalidEnvelope     = "INVALID_ENVELOPE"
	CodeUnknownSchema       = "UNKNOWN_SCHEMA"
	CodeInvalidTag          = "INVALID_TAG"
	CodeRecoveryUnsupported = "RECOVERY_UNSUPPORTED"
	CodeUnauthorized        = "SYNC_UNAUTHORIZED"
	CodeDeviceRevoked       = "DEVICE_REVOKED"
	CodeScopeMismatch       = "SCOPE_MISMATCH"
	CodeConflict            = "SYNC_CONFLICT"
	CodeStaleKeyEpoch       = "STALE_KEY_EPOCH"
	CodeReplay              = "REPLAY"
	CodeOperationMismatch   = "OPERATION_MISMATCH"
	CodeCursorExpired       = "CURSOR_EXPIRED"
	CodePayloadTooLarge     = "PAYLOAD_TOO_LARGE"
	CodeNotFound            = "NOT_FOUND"
	CodeRateLimited         = "RATE_LIMITED"
)

var messages = map[string]string{
	CodeInvalidEnvelope:     "invalid envelope",
	CodeUnknownSchema:       "unknown schema",
	CodeInvalidTag:          "invalid tag",
	CodeRecoveryUnsupported: "recovery enrollment is not available",
	CodeUnauthorized:        "unauthorized",
	CodeDeviceRevoked:       "device revoked",
	CodeScopeMismatch:       "device scope mismatch",
	CodeConflict:            "conflicting revision",
	CodeStaleKeyEpoch:       "stale key epoch",
	CodeReplay:              "replayed envelope",
	CodeOperationMismatch:   "operation mismatch",
	CodeCursorExpired:       "cursor expired",
	CodePayloadTooLarge:     "payload too large",
	CodeNotFound:            "not found",
	CodeRateLimited:         "rate limited",
}

// Error is a stable coded failure. Message is a fixed phrase and must not
// carry ciphertext, tokens, or record contents.
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

// Fail returns the coded error for a known code.
func Fail(code string) *Error {
	return &Error{Code: code, Message: messages[code]}
}

// IsCode reports whether err carries code.
func IsCode(err error, code string) bool {
	var de *Error
	return errors.As(err, &de) && de.Code == code
}
