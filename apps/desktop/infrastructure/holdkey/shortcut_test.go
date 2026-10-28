package holdkey

import (
	"arion.local/core/application"
	"testing"
)

func TestEveryOfferedKeyHasNativeMapping(t *testing.T) {
	for _, platform := range []string{"darwin", "windows"} {
		for _, key := range application.HoldKeysFor(platform) {
			if _, ok := newMatcher(platform, []application.HoldKey{key}); !ok {
				t.Fatalf("%s lacks %s", platform, key)
			}
		}
		for _, key := range application.ShortcutKeysFor(platform) {
			chord, ok := application.HoldShortcut(platform, application.KeyCtrl, key)
			if !ok {
				t.Fatalf("%s rejects %s", platform, key)
			}
			if _, ok := newMatcher(platform, []application.HoldKey{chord}); !ok {
				t.Fatalf("%s lacks %s", platform, chord)
			}
		}
	}
}

func TestMacCustomChord(t *testing.T) {
	m, _ := NewMac("ctrl+space")
	steps := []struct {
		kind   int
		code   int64
		flags  uint64
		repeat bool
		want   step
	}{
		{MacFlagsChanged, 0x3b, 0x40001, false, step{}},
		{MacKeyDown, 0x31, 0x40001, false, step{Down, true}},
		{MacKeyDown, 0x31, 0x40001, true, step{}},
		{MacKeyUp, 0x31, 0x40001, false, step{Up, true}},
		{MacKeyDown, 0x31, 0x40001, false, step{Down, true}},
		// Releasing any constituent ends the hold exactly once.
		{MacFlagsChanged, 0x3b, 0, false, step{Up, true}},
		{MacKeyUp, 0x31, 0, false, step{}},
		// Ordinary typing and extra modifiers must not start capture.
		{MacKeyDown, 0x31, 0, false, step{}},
		{MacKeyUp, 0x31, 0, false, step{}},
		{MacKeyDown, 0x31, 0x60003, false, step{}},
	}
	for i, s := range steps {
		key, event, ok := m.InterpretKey(s.kind, s.code, s.flags, s.repeat)
		if ok != s.want.ok || ok && (event != s.want.got || key != "ctrl+space") {
			t.Fatalf("step %d: %s %v %v", i, key, event, ok)
		}
	}
}

func TestWindowsCustomChordAndAltGr(t *testing.T) {
	w, _ := NewWindows("ctrl+space", "alt+enter")
	steps := []struct {
		message  uintptr
		vk, scan uint32
		key      application.HoldKey
		want     step
	}{
		{WinKeyDown, 0xa3, 0x11d, "", step{}},
		{WinKeyDown, 0x20, 0x39, "ctrl+space", step{Down, true}},
		{WinKeyDown, 0x20, 0x39, "", step{}},
		{WinKeyUp, 0xa3, 0x11d, "ctrl+space", step{Up, true}},
		{WinKeyUp, 0x20, 0x39, "", step{}},
		{WinKeyDown, vkLControl, altGrScan, "", step{}},
		{WinSysKeyDown, 0xa5, 0x138, "", step{}},
		{WinSysKeyDown, 0x0d, 0x1c, "alt+enter", step{Down, true}},
		{WinKeyDown, 0x41, 0x1e, "alt+enter", step{Other, true}},
		{WinSysKeyUp, 0x0d, 0x1c, "alt+enter", step{Up, true}},
	}
	for i, s := range steps {
		key, event, ok := w.InterpretKey(s.message, s.vk, s.scan)
		if ok != s.want.ok || ok && (event != s.want.got || key != s.key) {
			t.Fatalf("step %d: %s %v %v", i, key, event, ok)
		}
	}
}

func TestFunctionKeyAndPreheldModifiers(t *testing.T) {
	m, _ := NewMac("f8")
	if _, event, ok := m.InterpretKey(MacKeyDown, 0x64, 0x800000, false); !ok || event != Down {
		t.Fatal("Mac F8 down")
	}
	if _, event, ok := m.InterpretKey(MacKeyUp, 0x64, 0, false); !ok || event != Up {
		t.Fatal("Mac F8 up")
	}
	w, _ := NewWindows("f24")
	w.SetModifier(0xa0, true)
	if _, _, ok := w.InterpretKey(WinKeyDown, 0x87, 0); ok {
		t.Fatal("F24 matched with preheld Shift")
	}
	w.InterpretKey(WinKeyUp, 0x87, 0)
	w.InterpretKey(WinKeyUp, 0xa0, 0)
	if _, event, ok := w.InterpretKey(WinKeyDown, 0x87, 0); !ok || event != Down {
		t.Fatal("Windows F24 down")
	}
	if _, event, ok := w.InterpretKey(WinKeyUp, 0x87, 0); !ok || event != Up {
		t.Fatal("Windows F24 up")
	}
}

func TestAltGrReleaseClearsModifierSampledAtStartup(t *testing.T) {
	w, _ := NewWindows("ctrl+space")
	w.SetModifier(vkLControl, true)
	w.SetModifier(0xa5, true)
	w.InterpretKey(WinSysKeyUp, 0xa5, 0x138)
	w.InterpretKey(WinKeyUp, vkLControl, altGrScan)
	if _, _, ok := w.InterpretKey(WinKeyDown, 0x20, 0x39); ok {
		t.Fatal("AltGr left phantom Ctrl after hook startup")
	}
}
