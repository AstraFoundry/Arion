package bootstrap

import (
	"context"
	"errors"
	"sync"
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

func (m failedRemovalIME) Remove(context.Context) error { return m.failure }
func (failedRemovalIME) State() arion.InputMethodState  { return arion.InputMethodInstalled }

func TestFailedIMERemovalPreservesInstalledSource(t *testing.T) {
	failure := errors.New("input source could not be disabled")
	a := &App{ime: failedRemovalIME{failure: failure}, queueReady: make(chan struct{}, 1)}
	if err := (actions{a}).RemoveInputMethod(); err != nil || !a.state.InputMethodRemoving {
		t.Fatal(err)
	}
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
		t.Fatal("removal completion not posted")
	}
	if a.state.InputMethod != arion.InputMethodInstalled || a.state.InputMethodRemoving || a.state.InputMethodError != copy.IMERemoveFailed {
		t.Fatalf("failed removal hid installed source or failure: %+v", a.state)
	}
}

type installingIME struct {
	mu              sync.Mutex
	activationCalls atomic.Int32
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
func (m *installingIME) Activate(context.Context) error { m.activationCalls.Add(1); return nil }
func (m *installingIME) UpdateAvailable() bool          { return false }
func (m *installingIME) State() arion.InputMethodState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}
func (m *installingIME) setState(state arion.InputMethodState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
}
func (m *installingIME) OpenSettings() error { m.settingsCalls++; return m.settingsFailure }

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
			if adapter.settingsCalls != 0 {
				t.Fatal("installation navigated away before native consent completed")
			}
			if tc.state == arion.InputMethodPending && tc.failure == nil {
				adapter.setState(arion.InputMethodInstalled)
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

type activatingIME struct {
	InputMethod
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
	state   arion.InputMethodState
	failure error
}

func (m *activatingIME) Activate(ctx context.Context) error {
	m.calls.Add(1)
	close(m.started)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.release:
		return m.failure
	}
}
func (m *activatingIME) State() arion.InputMethodState { return m.state }

func drainIMECompletion(t *testing.T, a *App) {
	t.Helper()
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
		t.Fatal("activation completion not published")
	}
}

func TestIMEActivationCancellationRetainsBusyGuardAndInstalledSource(t *testing.T) {
	adapter := &activatingIME{started: make(chan struct{}), release: make(chan struct{}), state: arion.InputMethodPending}
	a := &App{ime: adapter, queueReady: make(chan struct{}, 1)}
	a.state.InputMethod = adapter.state
	x := actions{a}
	if err := x.ActivateInputMethod(); err != nil || !a.state.InputMethodActivating {
		t.Fatal("activation not asynchronous", err)
	}
	<-adapter.started
	_ = x.ActivateInputMethod()
	_ = x.InstallInputMethod()
	_ = x.RemoveInputMethod()
	x.CancelInputMethodSetup()
	if !a.state.InputMethodActivating {
		t.Fatal("competing operation allowed before cancelled worker exited")
	}
	drainIMECompletion(t, a)
	if adapter.calls.Load() != 1 || a.state.InputMethodActivating || a.state.InputMethod != arion.InputMethodPending || a.state.InputMethodError != "" {
		t.Fatalf("cancel lost state or left worker active: %+v", a.state)
	}
}

func TestIMEActivationKeepsFailureAndIgnoresObsoleteCompletion(t *testing.T) {
	for _, obsolete := range []bool{false, true} {
		adapter := &activatingIME{started: make(chan struct{}), release: make(chan struct{}), state: arion.InputMethodInstalled, failure: errors.New("selection failed")}
		a := &App{ime: adapter, queueReady: make(chan struct{}, 1)}
		_ = (actions{a}).ActivateInputMethod()
		<-adapter.started
		if obsolete {
			a.imeSetupGeneration++
			a.state.InputMethodError = copy.IMEPackageUnverified
		}
		close(adapter.release)
		drainIMECompletion(t, a)
		want := copy.IMEFailed
		if obsolete {
			want = copy.IMEPackageUnverified
		}
		if a.state.InputMethodError != want {
			t.Fatalf("stale/error outcome: %s", a.state.InputMethodError)
		}
	}
}

func TestIMEInstallContinuationRespectsSetupIntentAndUpgradeSelection(t *testing.T) {
	for _, tc := range []struct {
		name             string
		updating, closed bool
		activations      int32
	}{
		{name: "first install", activations: 1}, {name: "upgrade", updating: true}, {name: "window closed during copy", closed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &installingIME{gate: make(chan struct{}), state: arion.InputMethodSelected}
			a := &App{ime: adapter, queueReady: make(chan struct{}, 1)}
			a.state.InputMethod = arion.InputMethodAbsent
			if tc.updating {
				a.state.InputMethod = arion.InputMethodInstalled
				a.state.InputMethodUpdate = true
			}
			x := actions{a}
			if err := x.InstallInputMethod(); err != nil {
				t.Fatal(err)
			}
			if tc.closed {
				x.CancelInputMethodSetup()
			}
			close(adapter.gate)
			drainIMECompletion(t, a)
			if tc.activations > 0 {
				drainIMECompletion(t, a)
			}
			if adapter.activationCalls.Load() != tc.activations || a.state.InputMethodActivating {
				t.Fatalf("incorrect setup continuation: calls=%d busy=%v", adapter.activationCalls.Load(), a.state.InputMethodActivating)
			}
		})
	}
}
