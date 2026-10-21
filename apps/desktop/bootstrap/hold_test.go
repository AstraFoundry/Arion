package bootstrap

import (
	"testing"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
	arion "arion.local/desktop/ui/shared/runtime"
)

type holdPlatform struct {
	Platform
	failure       error
	starts, stops int
	sink          func(holdkey.Event)
}

func (p *holdPlatform) StartHoldKey(_ application.HoldKey, sink func(holdkey.Event)) error {
	p.sink = sink
	p.starts++
	return p.failure
}
func (p *holdPlatform) StopHoldKey() { p.stops++ }

func TestHoldKeyRecoversAfterConsentWithoutReselecting(t *testing.T) {
	p := &holdPlatform{failure: domain.Fail(domain.CodePermissionDenied, "input monitoring")}
	a := &App{platform: p}
	a.state.Preferences.HoldKey = application.HoldRightOption
	a.startHoldKey()
	if a.state.Hold.Active || a.state.Hold.Failure != domain.CodePermissionDenied {
		t.Fatalf("denied listener reported success: %+v", a.state.Hold)
	}
	p.failure = nil
	a.state.Permissions.InputMonitoring = arion.ConsentGranted
	a.refreshHoldConsent(arion.ConsentDenied)
	if !a.state.Hold.Active || a.state.Hold.Failure != "" || p.starts != 2 {
		t.Fatalf("consent did not restart chosen key: %+v, starts=%d", a.state.Hold, p.starts)
	}
	a.refreshHoldConsent(arion.ConsentGranted)
	if p.starts != 2 {
		t.Fatal("stable consent restarted the listener")
	}
}

func TestHoldKeyRevocationStopsListener(t *testing.T) {
	p := &holdPlatform{}
	a := &App{platform: p, core: &application.Coordinator{}}
	a.state.Preferences.HoldKey = application.HoldRightOption
	a.state.Hold.Active, a.state.Hold.Holding = true, true
	a.state.Permissions.InputMonitoring = arion.ConsentDenied
	a.refreshHoldConsent(arion.ConsentGranted)
	if a.state.Hold.Active || a.state.Hold.Holding || a.state.Hold.Failure != domain.CodePermissionDenied || p.stops != 1 {
		t.Fatalf("revoked listener stayed active: %+v, stops=%d", a.state.Hold, p.stops)
	}
	a.state.Permissions.InputMonitoring = arion.ConsentGranted
	a.refreshHoldConsent(arion.ConsentDenied)
	if !a.state.Hold.Active || p.starts != 1 {
		t.Fatal("listener did not recover from revocation")
	}
}

func TestHoldKeyRestartRejectsQueuedEventsFromOldListener(t *testing.T) {
	p := &holdPlatform{}
	a := &App{platform: p, testingHold: true}
	a.state.Preferences.HoldKey = application.HoldRightOption
	a.startHoldKey()
	old := p.sink
	old(holdkey.Down)
	a.startHoldKey()
	a.queue[0]()
	if !a.state.Hold.Pressed.IsZero() {
		t.Fatal("stale listener event reached the new key selection")
	}
}
