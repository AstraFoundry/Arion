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

func (p *holdPlatform) StartHoldKeys(keys []application.HoldKey, sink func(application.HoldKey, holdkey.Event)) error {
	var key application.HoldKey
	if len(keys) > 0 {
		key = keys[0]
	}
	p.sink = func(e holdkey.Event) { sink(key, e) }
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

func TestTranslateKeyRegistersAlongsideHoldKeyAndRejectsConflicts(t *testing.T) {
	p := &multiHoldPlatform{}
	a := &App{platform: p, core: &application.Coordinator{}, prefStore: settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}}
	a.prefs = application.DefaultPreferences(runtime.GOOS)
	a.state.Preferences = a.prefs
	a.startHoldKey()
	wantHold := application.DefaultHoldKey(runtime.GOOS)
	wantTranslate := application.DefaultTranslateKey(runtime.GOOS)
	if len(p.keys) != 2 || p.keys[0] != wantHold || p.keys[1] != wantTranslate {
		t.Fatalf("expected both HoldKey and TranslateKey registered, got %v", p.keys)
	}
	starts := p.starts
	before := a.state.Preferences
	(actions{a}).SetTranslateKey(wantHold)
	(actions{a}).SetHoldKey(wantTranslate)
	if a.state.Preferences.HoldKey != before.HoldKey || a.state.Preferences.TranslateKey != before.TranslateKey {
		t.Fatalf("conflicting selection changed assignments: %+v", a.state.Preferences)
	}
	if p.starts != starts || len(p.keys) != 2 || p.keys[0] != wantHold || p.keys[1] != wantTranslate {
		t.Fatal("conflicting selection replaced the listener")
	}
}

func TestChangingShortcutCancelsPreparationBeforeListenerReplacement(t *testing.T) {
	for _, translate := range []bool{false, true} {
		a := readinessApp(t)
		p := &multiHoldPlatform{}
		a.platform = p
		a.prefStore = settings.Store{Path: filepath.Join(t.TempDir(), "preferences.json")}
		a.prefs = application.DefaultPreferences(runtime.GOOS)
		a.state.Preferences = a.prefs
		a.core.Start()
		if a.core.Snapshot().State != domain.Preparing {
			t.Fatal("not preparing")
		}
		// UI state has not received the preparing event yet. Cancellation must
		// use the coordinator rather than a stale rendered snapshot.
		if translate {
			(actions{a}).SetTranslateKey("ctrl+f8")
		} else {
			(actions{a}).SetHoldKey("ctrl+space")
		}
		if a.core.Snapshot().Active {
			t.Fatal("old shortcut session survived replacement")
		}
	}
}

func TestShortcutRecordingReplacesActivationAndRetiresCallbacks(t *testing.T) {
	a := readinessApp(t)
	p := &multiHoldPlatform{}
	a.platform = p
	a.state.Preferences = application.DefaultPreferences(runtime.GOOS)
	a.core.Start()
	(actions{a}).RecordShortcut("field")
	if a.core.Snapshot().Active || len(p.keys) != 1 || p.keys[0] != holdkey.RecordKeys {
		t.Fatal("activation not replaced with recording")
	}
	p.multiSink("ctrl+shift+k", holdkey.Recorded)
	a.queue[0]()
	a.queue = nil
	if a.state.Hold.RecordedKey != "ctrl+shift+k" || a.state.Hold.RecordedSeq != 1 || a.core.Snapshot().Active {
		t.Fatal("recording started dictation or lost the key")
	}
	previous := p.multiSink
	(actions{a}).RecordShortcut("")
	previous("f8", holdkey.Recorded)
	a.queue[0]()
	if a.state.Hold.RecordedKey != "" || a.state.Hold.Recording != "" || p.keys[0] != a.state.Preferences.HoldKey {
		t.Fatal("stale capture survived restoration")
	}
}

func TestShortcutListenerRetryPreservesAssignments(t *testing.T) {
	a := readinessApp(t)
	p := &multiHoldPlatform{failure: domain.Fail(domain.CodeDeviceUnavailable, "fixture listener failure")}
	a.platform = p
	a.prefs = application.DefaultPreferences(runtime.GOOS)
	a.state.Preferences = a.prefs
	a.startHoldKey()
	if a.state.Hold.Active {
		t.Fatal("failed listener reported success")
	}
	p.failure = nil
	before := a.state.Preferences
	(actions{a}).RetryShortcutListener()
	if !a.state.Hold.Active || a.state.Hold.Failure != "" || p.starts != 2 {
		t.Fatal("retry did not recover shared listening")
	}
	if a.state.Preferences.HoldKey != before.HoldKey || a.state.Preferences.TranslateKey != before.TranslateKey ||
		a.state.Preferences.PreviousTranslateKey != before.PreviousTranslateKey || a.state.Preferences.TranslateLanguage != before.TranslateLanguage {
		t.Fatal("retry changed shortcut preferences")
	}
}
