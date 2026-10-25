package imebridge

import (
	"context"
	"errors"
	"testing"
	"time"

	"arion.local/core/contracts"
)

func TestCaptureWaitsForNativeCompositionAcquisition(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	m.RequireBeginAcknowledgement()
	h := attach(t, m, "activation")
	done := make(chan error, 1)
	go func() {
		_, err := m.Capture(context.Background(), contracts.Identity{SessionID: "session", Generation: 1})
		done <- err
	}()
	begin := h.read(t)
	select {
	case err := <-done:
		t.Fatalf("unacquired native composition returned: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	h.send(t, message{Op: "begun", Activation: begin.Activation, Session: begin.Session})
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestCaptureCancellationRevokesPendingNativeAcquisition(t *testing.T) {
	m := New("", nil, nil)
	defer m.Close()
	m.RequireBeginAcknowledgement()
	h := attach(t, m, "activation")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := m.Capture(ctx, contracts.Identity{SessionID: "cancelled", Generation: 1})
		done <- err
	}()
	begin := h.read(t)
	cancel()
	if message := h.read(t); message.Op != "cancel" || message.Session != begin.Session {
		t.Fatalf("pending acquisition not revoked: %+v", message)
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("capture cancellation: %v", err)
	}
	h.send(t, message{Op: "begun", Activation: begin.Activation, Session: begin.Session})
	m.mu.Lock()
	owned := m.token != ""
	m.mu.Unlock()
	if owned {
		t.Fatal("late native acquisition restored a cancelled capability")
	}
}
