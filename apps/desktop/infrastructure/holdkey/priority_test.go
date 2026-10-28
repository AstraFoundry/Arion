package holdkey

import (
	"testing"

	"arion.local/core/application"
)

func TestModifierPrefixGivesPriorityToChord(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		for _, modifierFirst := range []bool{false, true} {
			for _, reverse := range []bool{false, true} {
				solo, chord := application.HoldRightCommand, application.HoldKey("super+1")
				if platform == "windows" {
					solo, chord = application.HoldRightCtrl, "ctrl+1"
				}
				keys := []application.HoldKey{solo, chord}
				if reverse {
					keys[0], keys[1] = keys[1], keys[0]
				}
				var send func(application.HoldKey, bool) (application.HoldKey, Event, bool)
				if platform == "darwin" {
					m, _ := NewMac(keys...)
					held := false
					send = func(key application.HoldKey, down bool) (application.HoldKey, Event, bool) {
						kind := MacKeyUp
						if down {
							kind = MacKeyDown
						}
						if key == solo {
							held, kind = down, MacFlagsChanged
						}
						flags := uint64(0)
						if held {
							flags = 0x100010
						}
						return m.InterpretKey(kind, keySpecs[key].mac, flags, false)
					}
				} else {
					w, _ := NewWindows(keys...)
					send = func(key application.HoldKey, down bool) (application.HoldKey, Event, bool) {
						kind := uintptr(WinKeyUp)
						if down {
							kind = WinKeyDown
						}
						return w.InterpretKey(kind, keySpecs[key].win, 0)
					}
				}
				check := func(key application.HoldKey, down bool, wantKey application.HoldKey, want Event, wantOK bool) {
					t.Helper()
					got, event, ok := send(key, down)
					if ok != wantOK || ok && (got != wantKey || event != want) {
						t.Fatalf("%s modifierFirst=%v reverse=%v key=%s down=%v: %s %v %v", platform, modifierFirst, reverse, key, down, got, event, ok)
					}
				}
				check(solo, true, solo, Pending, true)
				check("1", true, chord, Down, true)
				check("1", true, "", 0, false)
				first, last := application.HoldKey("1"), solo
				if modifierFirst {
					first, last = last, first
				}
				check(first, false, chord, Up, true)
				check(last, false, "", 0, false)
				// A fresh standalone tap remains recognizable after the chord.
				check(solo, true, solo, Pending, true)
				check(solo, false, solo, Up, true)
			}
		}
	}
}

func TestModifierPrefixAllowsAdditionalChordModifiers(t *testing.T) {
	m, _ := NewMac(application.HoldRightCommand, "shift+super+1")
	if key, event, ok := m.InterpretKey(MacFlagsChanged, 0x36, 0x100010, false); !ok || key != application.HoldRightCommand || event != Pending {
		t.Fatal("single modifier was not deferred")
	}
	m.InterpretKey(MacFlagsChanged, 0x38, 0x120012, false)
	if key, event, ok := m.InterpretKey(MacKeyDown, 0x12, 0x120012, false); !ok || key != "shift+super+1" || event != Down {
		t.Fatal("additional chord modifier blocked priority")
	}
}
