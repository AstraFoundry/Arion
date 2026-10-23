package bootstrap

import (
	"arion.local/core/domain"
	arion "arion.local/desktop/ui/shared/runtime"
	"errors"
	"testing"
	"time"
)

type automaticIME struct{ *installingIME }

func (m automaticIME) UpdateAutomatically() error { return m.Install() }

func TestAutomaticIMEUpdateWaitsForUnusedSourceAndDoesNotLoopOnFailure(t *testing.T) {
	m := &installingIME{gate: make(chan struct{}), state: arion.InputMethodInstalled, failure: errors.New("replacement failed")}
	a := &App{ime: automaticIME{m}, queueReady: make(chan struct{}, 1)}
	a.state.InputMethodUpdate = true
	for _, state := range []arion.InputMethodState{arion.InputMethodAbsent, arion.InputMethodPending, arion.InputMethodDiscovering, arion.InputMethodSelected} {
		a.state.InputMethod = state
		a.updateInputMethodAutomatically()
		if a.state.InputMethodInstalling {
			t.Fatalf("updated unavailable or selected source: %s", state)
		}
	}
	a.state.InputMethod = arion.InputMethodInstalled
	a.state.Dictation.State = domain.Capturing
	a.updateInputMethodAutomatically()
	if a.state.InputMethodInstalling {
		t.Fatal("updated during dictation")
	}
	a.state.Dictation.State = domain.Completed
	a.updateInputMethodAutomatically()
	if !a.state.InputMethodInstalling {
		t.Fatal("unused source did not start updating")
	}
	a.updateInputMethodAutomatically()
	close(m.gate)
	select {
	case <-a.queueReady:
		a.queueMu.Lock()
		queued := a.queue
		a.queue = nil
		a.queueMu.Unlock()
		for _, fn := range queued {
			fn()
		}
	case <-time.After(2 * time.Second):
		t.Fatal("missing update completion")
	}
	if !a.state.InputMethodUpdateFailed || m.calls.Load() != 1 || m.activationCalls.Load() != 0 {
		t.Fatal("update failed to retain failure or started activation")
	}
	a.state.InputMethodUpdate = true
	a.updateInputMethodAutomatically()
	if a.state.InputMethodInstalling {
		t.Fatal("failed update retried every poll")
	}
}
