//go:build windows

package tsf

import (
	"context"
	"errors"
	"testing"

	"arion.local/core/contracts"
	"arion.local/core/domain"
)

func TestCapturePreservesCurrentPayloadVerification(t *testing.T) {
	m := &IME{validated: true, buildErr: errors.New("current package is damaged")}
	_, err := m.Capture(context.Background(), contracts.Identity{SessionID: "test"})
	var failure *domain.Error
	if !errors.As(err, &failure) || failure.Code != domain.CodeArtifactUnverified || failure.Detail != domain.DetailInputMethodArtifactUnverified {
		t.Fatalf("current package proof was bypassed or misclassified: %v", err)
	}
}
