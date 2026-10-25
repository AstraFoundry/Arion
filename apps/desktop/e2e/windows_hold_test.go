//go:build arion_e2e && windows

package e2e

import (
	"testing"
	"time"

	"arion.local/core/application"
	"arion.local/desktop/infrastructure/holdkey"
	"arion.local/desktop/infrastructure/windows"
	arion "arion.local/desktop/ui/shared/runtime"
	"github.com/egoist/mygo"
)

const (
	vkRControl       = 0xA3
	vkF24            = 0x87
	keyeventExtended = 0x0001
)

func tap(vk uintptr, flags uintptr) {
	procKeybdEvent.Call(vk, 0, flags, 0)
	procKeybdEvent.Call(vk, 0, flags|keyUp, 0)
}

// DSK-02: the low-level keyboard hook reports the hold key's press and
// release, and another key during the hold as a chord.
func TestWindowsHoldKeyHook(t *testing.T) {
	events := make(chan holdkey.Event, 16)
	if err := windows.StartHold(application.HoldRightCtrl, func(e holdkey.Event) { events <- e }); err != nil {
		t.Fatal(err)
	}
	defer windows.StopHold()
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended, 0)
	tap(vkF24, 0)
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
	for _, want := range []holdkey.Event{holdkey.Down, holdkey.Other, holdkey.Up} {
		select {
		case got := <-events:
			if got != want {
				t.Fatalf("event %v, want %v", got, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("no %v event", want)
		}
	}
	tap(vkF24, 0)
	select {
	case got := <-events:
		t.Fatalf("typing without a hold reported %v", got)
	case <-time.After(300 * time.Millisecond):
	}
}

// The isolated profile has no model. A real hold must not start a doomed
// session or arm capture for release; ready-model capture has separate tests.
func TestWindowsHoldKeyWithoutModel(t *testing.T) {
	editor := startWindowsEditor(t)
	editor.focus(t)
	var previous arion.HoldKey
	app.Update(func(s *arion.State) { previous = s.Preferences.HoldKey })
	mygo.RunOnMain(func() { app.Store().Actions.SetHoldKey(arion.HoldRightCtrl) })
	defer mygo.RunOnMain(func() { app.Store().Actions.SetHoldKey(previous) })
	var active bool
	app.Update(func(s *arion.State) { active = s.Hold.Active })
	if !active {
		t.Fatal("hold key listener not active")
	}
	var previousID string
	var ready bool
	app.Update(func(s *arion.State) {
		ready = s.Model.State == arion.ModelReady
		previousID = s.Dictation.SessionID
	})
	if ready {
		t.Fatal("fixture unexpectedly has a ready model")
	}
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended, 0)
	defer procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
	assertIdle := func() {
		var holding bool
		var id string
		app.Update(func(s *arion.State) { holding, id = s.Hold.Holding, s.Dictation.SessionID })
		if holding || id != previousID || app.Coordinator().Snapshot().Active {
			t.Fatal("unready model hold started or armed a session")
		}
	}
	time.Sleep(application.DefaultHoldThreshold + 200*time.Millisecond)
	assertIdle()
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
	time.Sleep(200 * time.Millisecond)
	assertIdle()
	if text := editor.text(); text != "" {
		t.Fatalf("text inserted without a model: %q", text)
	}
}
