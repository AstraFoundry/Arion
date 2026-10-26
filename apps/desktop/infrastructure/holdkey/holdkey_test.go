package holdkey

import (
	"testing"

	"arion.local/core/application"
)

type step struct {
	got Event
	ok  bool
}

func TestMacFn(t *testing.T) {
	m, ok := NewMac(application.HoldFn)
	if !ok {
		t.Fatal("fn unsupported")
	}
	steps := []struct {
		name       string
		eventType  int
		keycode    int64
		flags      uint64
		autorepeat bool
		want       step
	}{
		{"fn down", MacFlagsChanged, 0x3F, 0x800000, false, step{Down, true}},
		{"key repeat ignored", MacKeyDown, 0x00, 0x800000, true, step{}},
		{"shift while held", MacFlagsChanged, 0x38, 0x820000, false, step{Other, true}},
		{"letter while held", MacKeyDown, 0x00, 0x800000, false, step{Other, true}},
		{"fn up", MacFlagsChanged, 0x3F, 0, false, step{Up, true}},
		{"duplicate up", MacFlagsChanged, 0x3F, 0, false, step{}},
		{"shift alone", MacFlagsChanged, 0x38, 0x20000, false, step{}},
	}
	for _, s := range steps {
		_, got, ok := m.InterpretKey(s.eventType, s.keycode, s.flags, s.autorepeat)
		if ok != s.want.ok || (ok && got != s.want.got) {
			t.Fatalf("%s: got %v %v, want %v", s.name, got, ok, s.want)
		}
	}
}

// Right Option tracks its own device bit: the left Option keeps the shared
// Alternate flag set and must not hold the key down.
func TestMacRightOptionUsesDeviceBit(t *testing.T) {
	m, _ := NewMac(application.HoldRightOption)
	if _, got, ok := m.InterpretKey(MacFlagsChanged, 0x3D, 0x80040, false); !ok || got != Down {
		t.Fatalf("down: %v %v", got, ok)
	}
	// Right released while left Option still held: Alternate (0x80000)
	// remains, the right device bit (0x40) clears.
	if _, got, ok := m.InterpretKey(MacFlagsChanged, 0x3D, 0x80020, false); !ok || got != Up {
		t.Fatalf("up: %v %v", got, ok)
	}
}

func TestWindowsRightCtrl(t *testing.T) {
	w, ok := NewWindows(application.HoldRightCtrl)
	if !ok {
		t.Fatal("right ctrl unsupported")
	}
	if _, got, ok := w.InterpretKey(WinKeyDown, 0xA3, 0x11D); !ok || got != Down {
		t.Fatalf("down: %v %v", got, ok)
	}
	if _, _, ok := w.InterpretKey(WinKeyDown, 0xA3, 0x11D); ok {
		t.Fatal("auto-repeat reported")
	}
	if _, got, ok := w.InterpretKey(WinKeyDown, 0x41, 0x1E); !ok || got != Other {
		t.Fatalf("chord: %v %v", got, ok)
	}
	if _, _, ok := w.InterpretKey(WinKeyUp, 0x41, 0x1E); ok {
		t.Fatal("other key release reported")
	}
	if _, got, ok := w.InterpretKey(WinKeyUp, 0xA3, 0x11D); !ok || got != Up {
		t.Fatalf("up: %v %v", got, ok)
	}
	if _, _, ok := w.InterpretKey(WinKeyDown, 0x41, 0x1E); ok {
		t.Fatal("typing without a hold reported")
	}
}

// AltGr layouts send a synthetic left Ctrl with right Alt; it must not
// count as a chord while right Alt repeats.
func TestWindowsAltGrIsNotAChord(t *testing.T) {
	w, _ := NewWindows(application.HoldRightAlt)
	if _, _, ok := w.InterpretKey(WinKeyDown, vkLControl, altGrScan); ok {
		t.Fatal("synthetic ctrl reported before hold")
	}
	if _, got, ok := w.InterpretKey(WinSysKeyDown, 0xA5, 0x138); !ok || got != Down {
		t.Fatalf("down: %v %v", got, ok)
	}
	if _, _, ok := w.InterpretKey(WinKeyDown, vkLControl, altGrScan); ok {
		t.Fatal("synthetic ctrl repeat reported as chord")
	}
	if _, got, ok := w.InterpretKey(WinSysKeyUp, 0xA5, 0x138); !ok || got != Up {
		t.Fatalf("up: %v %v", got, ok)
	}
}

func TestUnsupportedKeys(t *testing.T) {
	if _, ok := NewMac(application.HoldRightAlt); ok {
		t.Fatal("macOS accepted a Windows key")
	}
	if _, ok := NewWindows(application.HoldFn); ok {
		t.Fatal("Windows accepted fn")
	}
	if _, ok := NewMac(application.HoldOff); ok {
		t.Fatal("macOS accepted HoldOff alone")
	}
	if _, ok := NewWindows(application.HoldOff); ok {
		t.Fatal("Windows accepted HoldOff alone")
	}
}

func TestMacDualHoldKeys(t *testing.T) {
	m, ok := NewMac(application.HoldRightOption, application.HoldRightCommand)
	if !ok {
		t.Fatal("dual macOS hold keys unsupported")
	}
	// Right Option press and release.
	if key, ev, ok := m.InterpretKey(MacFlagsChanged, 0x3D, 0x40, false); !ok || key != application.HoldRightOption || ev != Down {
		t.Fatalf("right option down: %q %v %v", key, ev, ok)
	}
	if key, ev, ok := m.InterpretKey(MacFlagsChanged, 0x3D, 0, false); !ok || key != application.HoldRightOption || ev != Up {
		t.Fatalf("right option up: %q %v %v", key, ev, ok)
	}
	// Right Command press and release.
	if key, ev, ok := m.InterpretKey(MacFlagsChanged, 0x36, 0x10, false); !ok || key != application.HoldRightCommand || ev != Down {
		t.Fatalf("right command down: %q %v %v", key, ev, ok)
	}
	// Pressing Right Option while Right Command is held is a chord on Right Command, not a Right Option start.
	if key, ev, ok := m.InterpretKey(MacFlagsChanged, 0x3D, 0x50, false); !ok || key != application.HoldRightCommand || ev != Other {
		t.Fatalf("chord while right command held: %q %v %v", key, ev, ok)
	}
	// Releasing the chorded Right Option produces no transition.
	if _, _, ok := m.InterpretKey(MacFlagsChanged, 0x3D, 0x10, false); ok {
		t.Fatal("releasing chorded second key reported a transition")
	}
	if key, ev, ok := m.InterpretKey(MacFlagsChanged, 0x36, 0, false); !ok || key != application.HoldRightCommand || ev != Up {
		t.Fatalf("right command up: %q %v %v", key, ev, ok)
	}
}

func TestWindowsDualHoldKeys(t *testing.T) {
	w, ok := NewWindows(application.HoldRightAlt, application.HoldRightCtrl)
	if !ok {
		t.Fatal("dual Windows hold keys unsupported")
	}
	if key, ev, ok := w.InterpretKey(WinSysKeyDown, 0xA5, 0x138); !ok || key != application.HoldRightAlt || ev != Down {
		t.Fatalf("right alt down: %q %v %v", key, ev, ok)
	}
	if key, ev, ok := w.InterpretKey(WinSysKeyUp, 0xA5, 0x138); !ok || key != application.HoldRightAlt || ev != Up {
		t.Fatalf("right alt up: %q %v %v", key, ev, ok)
	}
	if key, ev, ok := w.InterpretKey(WinKeyDown, 0xA3, 0x11D); !ok || key != application.HoldRightCtrl || ev != Down {
		t.Fatalf("right ctrl down: %q %v %v", key, ev, ok)
	}
	// Pressing Right Alt while Right Ctrl is held is a chord on Right Ctrl.
	if key, ev, ok := w.InterpretKey(WinSysKeyDown, 0xA5, 0x138); !ok || key != application.HoldRightCtrl || ev != Other {
		t.Fatalf("chord while right ctrl held: %q %v %v", key, ev, ok)
	}
	if _, _, ok := w.InterpretKey(WinSysKeyUp, 0xA5, 0x138); ok {
		t.Fatal("releasing chorded right alt reported a transition")
	}
	if key, ev, ok := w.InterpretKey(WinKeyUp, 0xA3, 0x11D); !ok || key != application.HoldRightCtrl || ev != Up {
		t.Fatalf("right ctrl up: %q %v %v", key, ev, ok)
	}
}
