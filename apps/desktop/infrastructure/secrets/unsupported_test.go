//go:build !darwin && !windows

package secrets

import (
	"errors"
	"testing"
)

func TestKeyIsUnsupported(t *testing.T) {
	if _, err := Key("Arion", "vocabulary-key"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("key: %v", err)
	}
	if _, err := ExistingKey("Arion", "clipboard-key"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("existing: %v", err)
	}
}
