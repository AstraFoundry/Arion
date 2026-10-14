//go:build !darwin && !windows

package keystore

import (
	"errors"
	"testing"

	"arion.local/desktop/internal/account"
)

func TestOpenDoesNotTouchCredentials(t *testing.T) {
	store := Open()
	if err := store.Put("rt-canary-value"); !errors.Is(err, account.ErrUnsupported) {
		t.Fatalf("put = %v", err)
	}
	if _, err := store.Get(); !errors.Is(err, account.ErrUnsupported) {
		t.Fatalf("get = %v", err)
	}
	if err := store.Delete(); !errors.Is(err, account.ErrUnsupported) {
		t.Fatalf("delete = %v", err)
	}
}
