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

// Holding the key through the production app starts a session that shows
// the release hint; releasing after the threshold leaves no live capture.
func TestWindowsHoldKeyDictates(t *testing.T) {
	editor := startNotepad(t)
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
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended, 0)
	eventually(t, "hold started a session", func() bool {
		var holding bool
		var id string
		app.Update(func(s *arion.State) { holding, id = s.Hold.Holding, s.Dictation.SessionID })
		return holding && id != ""
	})
	time.Sleep(application.DefaultHoldThreshold + 200*time.Millisecond)
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
	eventually(t, "release ended the hold", func() bool {
		var holding bool
		app.Update(func(s *arion.State) { holding = s.Hold.Holding })
		return !holding && !app.Coordinator().Snapshot().Active
	})
	if text := editor.text(); text != "" {
		t.Fatalf("text inserted without a model: %q", text)
	}
}
