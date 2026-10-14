package account

// RefreshTokenStore persists the refresh token and nothing else.
// Delete removes that item only. Implementations must not delete local
// encryption keys or local memory.
type RefreshTokenStore interface {
	Put(token string) error
	// Get returns ErrNotFound when no refresh token is stored.
	Get() (string, error)
	// Delete removes the refresh token. A missing item is success.
	Delete() error
}

// Browser opens the authorization URL in the system browser.
type Browser func(rawURL string) error
