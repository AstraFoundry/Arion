package account

import "arion.local/desktop/internal/oauthloopback"

func newPKCE() (verifier, challenge string, err error) { return oauthloopback.NewPKCE() }

func newState() (string, error) { return oauthloopback.NewState() }

func matchState(expected, got string) error {
	if !oauthloopback.MatchState(expected, got) {
		return ErrStateMismatch
	}
	return nil
}
