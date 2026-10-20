package application

import (
	"testing"
	"time"
)

var t0 = time.Unix(1_700_000_000, 0)

func at(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }

// Holding past the threshold dictates until release.
func TestHoldDictatesUntilRelease(t *testing.T) {
	var g HoldGesture
	if got := g.Press(at(0), HoldIdle); got != HoldStart {
		t.Fatalf("press: %v", got)
	}
	if got := g.Press(at(50), HoldCapturing); got != HoldNone {
		t.Fatalf("auto-repeat: %v", got)
	}
	if got := g.Release(at(900)); got != HoldStop {
		t.Fatalf("release: %v", got)
	}
}

// A quick tap keeps dictating hands-free; the next press ends it and its
// release does nothing more.
func TestTapLatchesHandsFree(t *testing.T) {
	var g HoldGesture
	g.Press(at(0), HoldIdle)
	if got := g.Release(at(120)); got != HoldNone {
		t.Fatalf("tap release: %v", got)
	}
	if got := g.Press(at(4000), HoldCapturing); got != HoldStop {
		t.Fatalf("second press: %v", got)
	}
	if got := g.Release(at(4600)); got != HoldNone {
		t.Fatalf("second release: %v", got)
	}
}

// Another key during the hold is a chord: the session it started is
// cancelled and the release does not stop anything.
func TestChordCancels(t *testing.T) {
	var g HoldGesture
	g.Press(at(0), HoldIdle)
	if got := g.Other(); got != HoldCancel {
		t.Fatalf("chord: %v", got)
	}
	if got := g.Other(); got != HoldNone {
		t.Fatalf("second key: %v", got)
	}
	if got := g.Release(at(900)); got != HoldNone {
		t.Fatalf("release after chord: %v", got)
	}
}

// Keys pressed while not holding, or while holding to end a session, are
// ordinary typing.
func TestOtherKeysOutsideAHoldStartedSession(t *testing.T) {
	var g HoldGesture
	if got := g.Other(); got != HoldNone {
		t.Fatalf("idle key: %v", got)
	}
	g.Press(at(0), HoldCapturing)
	if got := g.Other(); got != HoldNone {
		t.Fatalf("key while ending: %v", got)
	}
}

// While a session finalizes, presses are ignored instead of queueing.
func TestBusyIgnoresPress(t *testing.T) {
	var g HoldGesture
	if got := g.Press(at(0), HoldBusy); got != HoldNone {
		t.Fatalf("busy press: %v", got)
	}
	if got := g.Release(at(900)); got != HoldNone {
		t.Fatalf("busy release: %v", got)
	}
}

func TestHoldKeyNormalization(t *testing.T) {
	p := DefaultPreferences()
	p.HoldKey = "caps_lock"
	p, repaired := p.Normalize()
	if p.HoldKey != HoldOff || len(repaired) != 1 || repaired[0] != "holdKey" {
		t.Fatalf("got %q %v", p.HoldKey, repaired)
	}
	for _, platform := range []string{"darwin", "windows"} {
		for _, key := range HoldKeysFor(platform) {
			q := DefaultPreferences()
			q.HoldKey = key
			if _, repaired := q.Normalize(); len(repaired) != 0 {
				t.Fatalf("%s repaired %v", key, repaired)
			}
		}
	}
}
