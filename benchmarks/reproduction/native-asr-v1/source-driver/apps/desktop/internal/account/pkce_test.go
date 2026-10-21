package account

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestPKCEUsesS256(t *testing.T) {
	verifier, challenge, err := newPKCE()
	if err != nil {
		t.Fatal(err)
	}
	if verifier == "" || challenge == "" || verifier == challenge {
		t.Fatalf("verifier %q challenge %q", verifier, challenge)
	}
	sum := sha256.Sum256([]byte(verifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge != want {
		t.Fatalf("challenge = %s, want s256 %s", challenge, want)
	}
	state, err := newState()
	if err != nil || state == "" {
		t.Fatal(err)
	}
	if err := matchState(state, state); err != nil {
		t.Fatal(err)
	}
	if err := matchState(state, state+"x"); err != ErrStateMismatch {
		t.Fatalf("mismatch = %v", err)
	}
}
