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
		got, ok := m.Interpret(s.eventType, s.keycode, s.flags, s.autorepeat)
		if ok != s.want.ok || (ok && got != s.want.got) {
			t.Fatalf("%s: got %v %v, want %v", s.name, got, ok, s.want)
		}
	}
}

// Right Option tracks its own device bit: the left Option keeps the shared
// Alternate flag set and must not hold the key down.
func TestMacRightOptionUsesDeviceBit(t *testing.T) {
	m, _ := NewMac(application.HoldRightOption)
	if got, ok := m.Interpret(MacFlagsChanged, 0x3D, 0x80040, false); !ok || got != Down {
		t.Fatalf("down: %v %v", got, ok)
	}
	// Right released while left Option still held: Alternate (0x80000)
	// remains, the right device bit (0x40) clears.
	if got, ok := m.Interpret(MacFlagsChanged, 0x3D, 0x80020, false); !ok || got != Up {
		t.Fatalf("up: %v %v", got, ok)
	}
}

func TestWindowsRightCtrl(t *testing.T) {
	w, ok := NewWindows(application.HoldRightCtrl)
	if !ok {
		t.Fatal("right ctrl unsupported")
	}
	if got, ok := w.Interpret(WinKeyDown, 0xA3, 0x11D); !ok || got != Down {
		t.Fatalf("down: %v %v", got, ok)
	}
	if _, ok := w.Interpret(WinKeyDown, 0xA3, 0x11D); ok {
		t.Fatal("auto-repeat reported")
	}
	if got, ok := w.Interpret(WinKeyDown, 0x41, 0x1E); !ok || got != Other {
		t.Fatalf("chord: %v %v", got, ok)
	}
	if _, ok := w.Interpret(WinKeyUp, 0x41, 0x1E); ok {
		t.Fatal("other key release reported")
	}
	if got, ok := w.Interpret(WinKeyUp, 0xA3, 0x11D); !ok || got != Up {
		t.Fatalf("up: %v %v", got, ok)
	}
	if _, ok := w.Interpret(WinKeyDown, 0x41, 0x1E); ok {
		t.Fatal("typing without a hold reported")
	}
}

// AltGr layouts send a synthetic left Ctrl with right Alt; it must not
// count as a chord while right Alt repeats.
func TestWindowsAltGrIsNotAChord(t *testing.T) {
	w, _ := NewWindows(application.HoldRightAlt)
	if _, ok := w.Interpret(WinKeyDown, vkLControl, altGrScan); ok {
		t.Fatal("synthetic ctrl reported before hold")
	}
	if got, ok := w.Interpret(WinSysKeyDown, 0xA5, 0x138); !ok || got != Down {
		t.Fatalf("down: %v %v", got, ok)
	}
	if _, ok := w.Interpret(WinKeyDown, vkLControl, altGrScan); ok {
		t.Fatal("synthetic ctrl repeat reported as chord")
	}
	if got, ok := w.Interpret(WinSysKeyUp, 0xA5, 0x138); !ok || got != Up {
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
}
