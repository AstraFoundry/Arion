package holdkey

import (
	"arion.local/core/application"
	"testing"
)

func TestNativeShortcutRecording(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		t.Run(platform, func(t *testing.T) {
			emit := func(m *Mac, w *Windows, key application.HoldKey, down bool, mods application.KeyModifiers) (application.HoldKey, Event, bool) {
				spec := keySpecs[key]
				if platform == "windows" {
					message := uintptr(WinKeyUp)
					if down {
						message = WinKeyDown
					}
					return w.InterpretKey(message, spec.win, 0)
				}
				var flags uint64
				for _, family := range []struct {
					mod   application.KeyModifiers
					flags uint64
				}{{application.KeyCtrl, 0x40000}, {application.KeyAlt, 0x80000}, {application.KeyShift, 0x20000}, {application.KeySuper, 0x100000}} {
					if mods&family.mod != 0 {
						flags |= family.flags
					}
				}
				kind := MacKeyUp
				if down {
					kind = MacKeyDown
				}
				if spec.flag != 0 {
					kind = MacFlagsChanged
					if down {
						flags |= spec.flag
					}
				}
				return m.InterpretKey(kind, spec.mac, flags, false)
			}
			// A standalone modifier is captured on release, preserving its side.
			m, _ := NewMac(RecordKeys)
			w, _ := NewWindows(RecordKeys)
			if _, _, ok := emit(m, w, application.HoldRightCtrl, true, application.KeyCtrl); ok {
				t.Fatal("captured before release")
			}
			key, event, ok := emit(m, w, application.HoldRightCtrl, false, 0)
			if !ok || key != application.HoldRightCtrl || event != Recorded {
				t.Fatalf("modifier: %s %v %v", key, event, ok)
			}
			// Every constituent must be up before publishing a multi-modifier chord.
			for _, modifierFirst := range []bool{false, true} {
				m, _ = NewMac(RecordKeys)
				w, _ = NewWindows(RecordKeys)
				emit(m, w, application.HoldRightCtrl, true, application.KeyCtrl)
				emit(m, w, application.HoldLeftShift, true, application.KeyCtrl|application.KeyShift)
				if _, _, ok := emit(m, w, "k", true, application.KeyCtrl|application.KeyShift); ok {
					t.Fatal("captured held chord")
				}
				if modifierFirst {
					emit(m, w, application.HoldRightCtrl, false, application.KeyShift)
					emit(m, w, application.HoldLeftShift, false, 0)
					key, event, ok = emit(m, w, "k", false, 0)
				} else {
					emit(m, w, "k", false, application.KeyCtrl|application.KeyShift)
					emit(m, w, application.HoldLeftShift, false, application.KeyCtrl)
					key, event, ok = emit(m, w, application.HoldRightCtrl, false, 0)
				}
				if !ok || key != "ctrl+shift+k" || event != Recorded {
					t.Fatalf("chord: %s %v %v", key, event, ok)
				}
			}
			// Invalid typing stays in capture and permits a subsequent valid key.
			m, _ = NewMac(RecordKeys)
			w, _ = NewWindows(RecordKeys)
			emit(m, w, "a", true, 0)
			if _, event, ok := emit(m, w, "a", false, 0); !ok || event != RecordInvalid {
				t.Fatal("bare typing accepted")
			}
			emit(m, w, "f8", true, 0)
			if key, event, ok := emit(m, w, "f8", false, 0); !ok || event != Recorded || key != "f8" {
				t.Fatal("retry failed")
			}
		})
	}
}

func TestRecordingEscapeAndFn(t *testing.T) {
	m, _ := NewMac(RecordKeys)
	if _, event, ok := m.InterpretKey(MacKeyDown, 0x35, 0, false); !ok || event != RecordCancelled {
		t.Fatal("Escape did not cancel")
	}
	m, _ = NewMac(RecordKeys)
	m.InterpretKey(MacFlagsChanged, 0x3f, 0x800000, false)
	if key, event, ok := m.InterpretKey(MacFlagsChanged, 0x3f, 0, false); !ok || event != Recorded || key != application.HoldFn {
		t.Fatal("Fn not captured")
	}
	w, _ := NewWindows(RecordKeys)
	if _, event, ok := w.InterpretKey(WinKeyDown, 0x1b, 0); !ok || event != RecordCancelled {
		t.Fatal("Windows Escape did not cancel")
	}
}

func TestReplacedWindowsListenerIgnoresHeldTriggerRepeat(t *testing.T) {
	w, _ := NewWindows("f8")
	w.SetKeyDown(0x77, true)
	if _, _, ok := w.InterpretKey(WinKeyDown, 0x77, 0); ok {
		t.Fatal("held key activated after cancellation")
	}
	w.InterpretKey(WinKeyUp, 0x77, 0)
	if _, event, ok := w.InterpretKey(WinKeyDown, 0x77, 0); !ok || event != Down {
		t.Fatal("fresh press not activated")
	}
}

func TestRecorderDiscardsPreheldAltGrBeforeAcceptingANewChord(t *testing.T) {
	w, _ := NewWindows(RecordKeys)
	w.SetKeyDown(vkLControl, true)
	w.SetKeyDown(0xa5, true)
	for _, event := range []struct {
		message  uintptr
		vk, scan uint32
	}{
		{WinKeyDown, 0x4b, 0}, {WinKeyUp, 0x4b, 0}, {WinSysKeyUp, 0xa5, 0x138}, {WinKeyUp, vkLControl, altGrScan},
	} {
		if _, _, ok := w.InterpretKey(event.message, event.vk, event.scan); ok {
			t.Fatal("preheld AltGr was captured")
		}
	}
	w.InterpretKey(WinKeyDown, vkLControl, altGrScan)
	w.InterpretKey(WinSysKeyDown, 0xa5, 0x138)
	w.InterpretKey(WinKeyDown, 0x4b, 0)
	w.InterpretKey(WinKeyUp, 0x4b, 0)
	key, event, ok := w.InterpretKey(WinSysKeyUp, 0xa5, 0x138)
	if !ok || event != Recorded || key != "alt+k" {
		t.Fatalf("fresh AltGr chord: %s %v %v", key, event, ok)
	}
}
