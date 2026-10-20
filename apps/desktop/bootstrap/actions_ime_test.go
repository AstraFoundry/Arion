package bootstrap

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/ui/shared/copy"
	arion "arion.local/desktop/ui/shared/runtime"
)

type failedRemovalIME struct {
	InputMethod
	failure error
}

func (m failedRemovalIME) Remove() error               { return m.failure }
func (failedRemovalIME) State() arion.InputMethodState { return arion.InputMethodInstalled }

func TestFailedIMERemovalPreservesChosenInputMode(t *testing.T) {
	failure := errors.New("input source could not be disabled")
	a := &App{ime: failedRemovalIME{failure: failure}}
	a.state.Preferences.Mode = arion.ModeSystemIME
	if err := (actions{a}).RemoveInputMethod(); !errors.Is(err, failure) {
		t.Fatalf("removal error lost: %v", err)
	}
	if a.state.Preferences.Mode != arion.ModeSystemIME || a.state.InputMethod != arion.InputMethodInstalled {
		t.Fatalf("failed removal changed input mode or hid installed source: %+v", a.state)
	}
}

type installingIME struct {
	InputMethod
	gate            chan struct{}
	calls           atomic.Int32
	settingsCalls   int
	settingsFailure error
	failure         error
	state           arion.InputMethodState
}

func (m *installingIME) Install() error {
	m.calls.Add(1)
	<-m.gate
	return m.failure
}
func (m *installingIME) UpdateAvailable() bool         { return false }
func (m *installingIME) State() arion.InputMethodState { return m.state }
func (m *installingIME) OpenSettings() error           { m.settingsCalls++; return m.settingsFailure }

func TestIMEInstallationPublishesProgressAndFinalStateWithoutBlocking(t *testing.T) {
	for _, tc := range []struct {
		name            string
		state           arion.InputMethodState
		failure         error
		settingsFailure error
		wantError       copy.Key
	}{
		{"pending system activation", arion.InputMethodPending, nil, nil, ""},
		{"ready", arion.InputMethodInstalled, nil, nil, ""},
		{"registration failure", arion.InputMethodAbsent, errors.New("private path must not reach UI"), nil, copy.IMEFailed},
		{"untrusted package preserves pending host", arion.InputMethodPending, domain.Fail(domain.CodeArtifactUnverified, "signature invalid"), nil, copy.IMEPackageUnverified},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &installingIME{gate: make(chan struct{}), failure: tc.failure, state: tc.state, settingsFailure: tc.settingsFailure}
			a := &App{ime: adapter, queueReady: make(chan struct{}, 1)}
			a.state.InputMethod = arion.InputMethodAbsent
			a.state.Preferences.Mode = arion.ModeShortcut
			x := actions{a}
			if err := x.InstallInputMethod(); err != nil || !a.state.InputMethodInstalling {
				t.Fatalf("installation did not return with immediate progress: %v", err)
			}
			_ = x.InstallInputMethod()
			_ = x.RemoveInputMethod() // Installation owns the bundle until completion.
			close(adapter.gate)
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
				t.Fatal("installation completion not posted")
			}
			if adapter.calls.Load() != 1 || a.state.InputMethodInstalling || a.state.InputMethod != tc.state {
				t.Fatalf("installation completion/duplicate guard failed: %+v", a.state)
			}
			if a.state.InputMethodError != tc.wantError {
				t.Fatalf("failure message lost or pending misreported as failure: %s", a.state.InputMethodError)
			}
			if a.state.Preferences.Mode != arion.ModeShortcut {
				t.Fatal("installation changed the selected input mode")
			}
			if adapter.settingsCalls != 0 {
				t.Fatal("installation navigated away before native consent completed")
			}
			if tc.state == arion.InputMethodPending && tc.failure == nil {
				adapter.state = arion.InputMethodInstalled
				a.refreshInputMethod()
				if a.state.InputMethod != arion.InputMethodInstalled || adapter.calls.Load() != 1 {
					t.Fatal("OS activation did not refresh without reinstalling")
				}
			}
		})
	}
}

func TestIMESettingsNavigationFailureKeepsPendingSource(t *testing.T) {
	adapter := &installingIME{state: arion.InputMethodPending, settingsFailure: errors.New("settings unavailable")}
	a := &App{ime: adapter}
	a.state.InputMethod = arion.InputMethodPending
	if err := (actions{a}).OpenInputMethodSettings(); err == nil {
		t.Fatal("settings failure hidden")
	}
	if adapter.calls.Load() != 0 || a.state.InputMethod != arion.InputMethodPending {
		t.Fatal("navigation reinstalled or lost pending source")
	}
}
