package bootstrap

import (
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/holdkey"
	arion "arion.local/desktop/ui/shared/runtime"
)

func prefixApp(t *testing.T, translationSolo bool) *App {
	t.Helper()
	a := readinessApp(t)
	a.state.Model.State = arion.ModelReady
	a.state.Preferences.HoldKey, a.state.Preferences.TranslateKey = "super+1", application.HoldRightCommand
	a.state.Preferences.TranslateLanguage = arion.TargetEnglish
	if !translationSolo {
		a.state.Preferences.HoldKey, a.state.Preferences.TranslateKey = a.state.Preferences.TranslateKey, a.state.Preferences.HoldKey
	}
	t.Cleanup(a.clearDeferredHold)
	return a
}

func TestDeferredModifierDoesNotStartBeforeChord(t *testing.T) {
	for _, translationSolo := range []bool{false, true} {
		a := prefixApp(t, translationSolo)
		at := time.Now()
		a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Pending, at)
		pending := a.holdPending
		if a.core.Snapshot().Active || a.state.Hold.Holding {
			t.Fatal("prefix started a session")
		}
		a.onHoldKeyEvent("super+1", holdkey.Down, at.Add(80*time.Millisecond))
		a.activateDeferredHold(pending, a.holdGeneration)
		if !a.core.Snapshot().Active || a.holdKey != "super+1" || a.translating == translationSolo || a.holdPending != nil {
			t.Fatal("chord did not exclusively own activation")
		}
		a.onHoldKeyEvent("super+1", holdkey.Up, at.Add(180*time.Millisecond))
		a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Up, at.Add(190*time.Millisecond))
		if !a.core.Snapshot().Active || a.state.Hold.Holding || a.translating == translationSolo {
			t.Fatal("trailing modifier release changed a latched chord")
		}
	}
}

func TestDeferredModifierKeepsStandaloneTapAndHold(t *testing.T) {
	for _, held := range []bool{false, true} {
		a := prefixApp(t, false)
		at := time.Now()
		a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Pending, at)
		elapsed := 80 * time.Millisecond
		if held {
			a.activateDeferredHold(a.holdPending, a.holdGeneration)
			if !a.core.Snapshot().Active || !a.state.Hold.Holding {
				t.Fatal("standalone threshold did not start capture")
			}
			elapsed = 500 * time.Millisecond
		}
		a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Up, at.Add(elapsed))
		if a.core.Snapshot().Active == held || a.state.Hold.Holding {
			t.Fatalf("held=%v: incorrect release state", held)
		}
	}
}

func TestChordTakesOverAConfirmedSoloHold(t *testing.T) {
	a := prefixApp(t, true)
	at := time.Now()
	a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Pending, at)
	a.activateDeferredHold(a.holdPending, a.holdGeneration)
	if !a.translating || !a.core.Snapshot().Active {
		t.Fatal("solo translation did not start")
	}
	a.onHoldKeyEvent("super+1", holdkey.Down, at.Add(500*time.Millisecond))
	if a.translating || !a.core.Snapshot().Active || a.holdKey != "super+1" {
		t.Fatal("late chord did not replace provisional translation")
	}
}

func TestDeferredModifierCancellationInvalidatesTimer(t *testing.T) {
	for _, reset := range []string{"other", "restart", "reset", "revocation"} {
		a := prefixApp(t, true)
		a.platform = &multiHoldPlatform{}
		a.state.Permissions.InputMonitoring = arion.ConsentDenied
		a.state.Hold.Active = true
		a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Pending, time.Now())
		pending, generation := a.holdPending, a.holdGeneration
		switch reset {
		case "other":
			a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Other, time.Now())
		case "restart":
			a.startHoldKey()
		case "reset":
			a.resetHoldGesture()
		case "revocation":
			a.refreshHoldConsent(arion.ConsentGranted)
		}
		a.activateDeferredHold(pending, generation)
		a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Up, time.Now())
		if a.holdPending != nil || a.core.Snapshot().State != domain.Idle {
			t.Fatalf("%s: stale deferred activation survived cancellation", reset)
		}
	}
}

func TestDeferredModifierOldTimerCannotResolveANewerPress(t *testing.T) {
	a := prefixApp(t, true)
	a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Pending, time.Now())
	old := a.holdPending
	a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Other, time.Now())
	a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Pending, time.Now())
	current := a.holdPending
	a.activateDeferredHold(old, a.holdGeneration)
	if a.holdPending != current || a.core.Snapshot().Active {
		t.Fatal("old timer resolved a newer press")
	}
}

func TestDeferredModifierTimerDispatchesThroughMainQueue(t *testing.T) {
	a := prefixApp(t, false)
	a.queueReady = make(chan struct{}, 1)
	a.onHoldKeyEvent(application.HoldRightCommand, holdkey.Pending, time.Now())
	if a.core.Snapshot().Active {
		t.Fatal("timer activated before the threshold")
	}
	select {
	case <-a.queueReady:
	case <-time.After(time.Second):
		t.Fatal("deferred threshold was not posted")
	}
	if a.core.Snapshot().Active {
		t.Fatal("timer started capture off the main queue")
	}
	a.queueMu.Lock()
	batch := a.queue
	a.queue = nil
	a.queueMu.Unlock()
	for _, dispatch := range batch {
		dispatch()
	}
	if !a.core.Snapshot().Active || a.holdPending != nil {
		t.Fatal("main queue did not resolve the held modifier")
	}
}
