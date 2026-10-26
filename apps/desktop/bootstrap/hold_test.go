package bootstrap

import (
	"path/filepath"
	"runtime"
	"testing"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/infrastructure/settings"
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

func (p *holdPlatform) StartHoldKeys(keys []application.HoldKey, sink func(application.HoldKey, holdkey.Event)) error {
	var key application.HoldKey
	if len(keys) > 0 {
		key = keys[0]
	}
	return p.StartHoldKey(key, func(e holdkey.Event) { sink(key, e) })
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

// The onboarding test hears both keys and says which went down, so each
// key step acknowledges only its own.
func TestHoldKeyTestRecordsWhichKeyWentDown(t *testing.T) {
	p := &multiHoldPlatform{}
	a := &App{platform: p, testingHold: true}
	a.state.Preferences.HoldKey = application.HoldRightOption
	a.state.Preferences.TranslateKey = application.HoldRightCommand
	a.startHoldKey()
	p.multiSink(application.HoldRightCommand, holdkey.Down)
	a.queue[0]()
	if a.state.Hold.Pressed.IsZero() || a.state.Hold.PressedKey != application.HoldRightCommand {
		t.Fatalf("translation key press not recorded: %+v", a.state.Hold)
	}
	if a.state.Hold.Holding {
		t.Fatal("test press started dictation")
	}
}

type multiHoldPlatform struct {
	holdPlatform
	keys      []application.HoldKey
	multiSink func(application.HoldKey, holdkey.Event)
}

func (p *multiHoldPlatform) StartHoldKeys(keys []application.HoldKey, sink func(application.HoldKey, holdkey.Event)) error {
	p.keys = append([]application.HoldKey(nil), keys...)
	p.multiSink = sink
	p.starts++
	return p.failure
}

func TestTranslateKeyRegistersAlongsideHoldKeyAndResolvesConflicts(t *testing.T) {
	p := &multiHoldPlatform{}
	a := &App{platform: p, prefStore: settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}}
	a.prefs = application.DefaultPreferences(runtime.GOOS)
	a.state.Preferences = a.prefs
	a.startHoldKey()
	wantHold := application.DefaultHoldKey(runtime.GOOS)
	wantTranslate := application.DefaultTranslateKey(runtime.GOOS)
	if len(p.keys) != 2 || p.keys[0] != wantHold || p.keys[1] != wantTranslate {
		t.Fatalf("expected both HoldKey and TranslateKey registered, got %v", p.keys)
	}
	// Choosing the current HoldKey as TranslateKey swaps HoldKey to the previous TranslateKey.
	(actions{a}).SetTranslateKey(wantHold)
	if a.state.Preferences.TranslateKey != wantHold || a.state.Preferences.HoldKey != wantTranslate {
		t.Fatalf("expected keys to swap, got hold=%q translate=%q", a.state.Preferences.HoldKey, a.state.Preferences.TranslateKey)
	}
	// Choosing the current TranslateKey as HoldKey turns TranslateKey off.
	(actions{a}).SetHoldKey(wantHold)
	if a.state.Preferences.HoldKey != wantHold || a.state.Preferences.TranslateKey != application.HoldOff {
		t.Fatalf("expected TranslateKey off on HoldKey conflict, got hold=%q translate=%q", a.state.Preferences.HoldKey, a.state.Preferences.TranslateKey)
	}
	if len(p.keys) != 1 || p.keys[0] != wantHold {
		t.Fatalf("expected only HoldKey registered when TranslateKey is off, got %v", p.keys)
	}
}
