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
	if err := windows.StartHoldKeys([]application.HoldKey{application.HoldRightCtrl}, func(_ application.HoldKey, e holdkey.Event) { events <- e }); err != nil {
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

// A configured modifier remains pending until the full chord is known.
func TestWindowsModifierPrefixChordHook(t *testing.T) {
	type observed struct {
		key   application.HoldKey
		event holdkey.Event
	}
	for _, modifierFirst := range []bool{false, true} {
		events := make(chan observed, 16)
		if err := windows.StartHoldKeys([]application.HoldKey{application.HoldRightCtrl, "ctrl+f8"}, func(key application.HoldKey, event holdkey.Event) {
			events <- observed{key, event}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(windows.StopHold)
		procKeybdEvent.Call(vkRControl, 0, keyeventExtended, 0)
		procKeybdEvent.Call(0x77, 0, 0, 0)
		if modifierFirst {
			procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
			procKeybdEvent.Call(0x77, 0, keyUp, 0)
		} else {
			procKeybdEvent.Call(0x77, 0, keyUp, 0)
			procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
		}
		for i, want := range []holdkey.Event{holdkey.Pending, holdkey.Down, holdkey.Up} {
			key := application.HoldKey("ctrl+f8")
			if i == 0 {
				key = application.HoldRightCtrl
			}
			select {
			case got := <-events:
				if got.key != key || got.event != want {
					t.Fatalf("modifierFirst=%v: got %+v, want %s %v", modifierFirst, got, key, want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("missing native priority event")
			}
		}
		select {
		case got := <-events:
			t.Fatalf("trailing release activated the single key: %+v", got)
		case <-time.After(100 * time.Millisecond):
		}
		windows.StopHold()
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

// Exercise custom triggers through the actual Windows hook, including
// releasing a modifier before the trigger and immediate listener teardown.
func TestWindowsCustomShortcutHook(t *testing.T) {
	for _, tc := range []struct {
		name                string
		key                 application.HoldKey
		ctrl, modifierFirst bool
	}{
		{"function", "f8", false, false},
		{"chord-trigger-release", "ctrl+f8", true, false},
		{"chord-modifier-release", "ctrl+f8", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := make(chan holdkey.Event, 16)
			if err := windows.StartHoldKeys([]application.HoldKey{tc.key}, func(_ application.HoldKey, event holdkey.Event) { events <- event }); err != nil {
				t.Fatal(err)
			}
			defer windows.StopHold()
			const vkF8 = 0x77
			if tc.ctrl {
				procKeybdEvent.Call(vkRControl, 0, keyeventExtended, 0)
				defer procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
			}
			procKeybdEvent.Call(vkF8, 0, 0, 0)
			defer procKeybdEvent.Call(vkF8, 0, keyUp, 0)
			if tc.modifierFirst {
				procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
			}
			procKeybdEvent.Call(vkF8, 0, keyUp, 0)
			if tc.ctrl && !tc.modifierFirst {
				procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
			}
			for _, want := range []holdkey.Event{holdkey.Down, holdkey.Up} {
				select {
				case got := <-events:
					if got != want {
						t.Fatalf("got %v, want %v", got, want)
					}
				case <-time.After(5 * time.Second):
					t.Fatalf("missing %v", want)
				}
			}
			select {
			case event := <-events:
				t.Fatalf("extra event %v", event)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
	for i := 0; i < 20; i++ {
		if err := windows.StartHoldKeys([]application.HoldKey{"f8"}, func(application.HoldKey, holdkey.Event) {}); err != nil {
			t.Fatal(err)
		}
		windows.StopHold()
	}
}

func TestWindowsShortcutRecordingHook(t *testing.T) {
	type captured struct {
		key   application.HoldKey
		event holdkey.Event
	}
	events := make(chan captured, 16)
	if err := windows.StartHoldKeys([]application.HoldKey{holdkey.RecordKeys}, func(key application.HoldKey, event holdkey.Event) { events <- captured{key, event} }); err != nil {
		t.Fatal(err)
	}
	defer windows.StopHold()
	const vkF8 = 0x77
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended, 0)
	procKeybdEvent.Call(vkF8, 0, 0, 0)
	procKeybdEvent.Call(vkF8, 0, keyUp, 0)
	procKeybdEvent.Call(vkRControl, 0, keyeventExtended|keyUp, 0)
	select {
	case got := <-events:
		if got.event != holdkey.Recorded || got.key != "ctrl+f8" {
			t.Fatalf("got %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no recorded shortcut")
	}
	select {
	case extra := <-events:
		t.Fatalf("extra capture or activation: %+v", extra)
	case <-time.After(100 * time.Millisecond):
	}
}
