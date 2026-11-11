//go:build linux && (amd64 || arm64)

package linux

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

// gdkEventKey is GTK 3's struct, as physical_key_test.go checks it.
type gdkEventKey struct {
	kind            int32
	window          uintptr
	sendEvent       int8
	time            uint32
	state           uint32
	keyval          uint32
	length          int32
	text            uintptr
	hardwareKeycode uint16
	group           uint8
	isModifier      uint32
}

const (
	gdkKeyPress   = 8
	gdkKeyRelease = 9

	gdkShiftMask = 1
	gdkSuperMask = 1 << 26
)

// keySink notes the events a surface sends.
type keySink struct {
	platform.WindowHandler
	got   []platform.SurfaceEvent
	taken func(platform.SurfaceEvent) bool
}

func (k *keySink) SurfaceEvent(ev platform.SurfaceEvent) bool {
	k.got = append(k.got, ev)
	return k.taken != nil && k.taken(ev)
}

// keySurface is a surface with no GTK behind it: what it calls of GDK for a
// key stands in for the library.
func keySurface(t *testing.T) (*surface, *keySink) {
	t.Helper()
	free, copyEvent, toUnicode := gdkEventFree, gdkEventCopy, gdkKeyvalToUnicode
	t.Cleanup(func() { gdkEventFree, gdkEventCopy, gdkKeyvalToUnicode = free, copyEvent, toUnicode })
	gdkEventFree = func(ptr) {}
	gdkEventCopy = func(ptr) ptr { return 0 }
	gdkKeyvalToUnicode = func(keyval uint32) uint32 {
		switch {
		case keyval == 0xffb5: // KP_5
			return '5'
		case keyval < 0x100:
			return keyval
		}
		return 0
	}
	sink := &keySink{}
	return &surface{w: &window{h: sink}}, sink
}

func sendKey(s *surface, kind int32, keyval uint32, evdev uint16, state uint32) bool {
	ev := gdkEventKey{kind: kind, keyval: keyval, hardwareKeycode: evdev + 8, state: state}
	defer runtime.KeepAlive(&ev)
	return s.keyEvent(ptr(unsafe.Pointer(&ev)))
}

// A modifier on its own tells the state as ever, then the key by position,
// with the side: a lone tap of Super comes through whole.
func TestKeyEventSuperTap(t *testing.T) {
	s, sink := keySurface(t)
	if sendKey(s, gdkKeyPress, 0xffec, 126, 0) { // Super_R
		t.Error("taken with nothing taking it")
	}
	sendKey(s, gdkKeyRelease, 0xffec, 126, gdkSuperMask)
	want := []platform.SurfaceEvent{
		{Kind: platform.ModifiersChanged, Mods: platform.ModSuper},
		{Kind: platform.RawKeyPressed, Code: platform.CodeSuperRight, Mods: platform.ModSuper},
		{Kind: platform.ModifiersChanged},
		{Kind: platform.RawKeyReleased, Code: platform.CodeSuperRight},
	}
	checkEvents(t, sink.got, want)
}

// Letting go of one Shift with the other held keeps Shift held.
func TestKeyEventBothShifts(t *testing.T) {
	s, sink := keySurface(t)
	sendKey(s, gdkKeyPress, 0xffe1, 42, 0)
	sendKey(s, gdkKeyPress, 0xffe2, 54, gdkShiftMask)
	sendKey(s, gdkKeyRelease, 0xffe1, 42, gdkShiftMask)
	sendKey(s, gdkKeyRelease, 0xffe2, 54, gdkShiftMask)
	want := []platform.SurfaceEvent{
		{Kind: platform.ModifiersChanged, Mods: platform.ModShift},
		{Kind: platform.RawKeyPressed, Code: platform.CodeShiftLeft, Mods: platform.ModShift},
		{Kind: platform.ModifiersChanged, Mods: platform.ModShift},
		{Kind: platform.RawKeyPressed, Code: platform.CodeShiftRight, Mods: platform.ModShift},
		{Kind: platform.ModifiersChanged, Mods: platform.ModShift},
		{Kind: platform.RawKeyReleased, Code: platform.CodeShiftLeft, Mods: platform.ModShift},
		{Kind: platform.ModifiersChanged},
		{Kind: platform.RawKeyReleased, Code: platform.CodeShiftRight},
	}
	checkEvents(t, sink.got, want)
}

// What keyvals give no Key for comes by position alone, and only raw: for
// the content, as before, no KeyPressed.
func TestKeyEventRawKeys(t *testing.T) {
	for _, c := range []struct {
		name     string
		keyval   uint32
		evdev    uint16
		code     platform.KeyCode
		physical platform.Key
	}{
		{"Caps Lock", 0xffe5, 58, platform.CodeCapsLock, platform.KeyUnknown},
		{"Num Lock", 0xff7f, 69, platform.CodeNumLock, platform.KeyUnknown},
		{"Scroll Lock", 0xff14, 70, platform.CodeScrollLock, platform.KeyUnknown},
		{"Print Screen", 0xff61, 99, platform.CodePrintScreen, platform.KeyUnknown},
		{"Pause", 0xff13, 119, platform.CodePause, platform.KeyUnknown},
		{"AltGr", 0xfe03, 100, platform.CodeAltRight, platform.KeyUnknown},
		{"102nd key", '<', 86, platform.CodeIntlBackslash, platform.KeyUnknown},
		{"keypad 5 without Num Lock", 0xff9d, 76, platform.CodeNumpad5, platform.KeyUnknown},
		{"keypad +", '+', 78, platform.CodeNumpadAdd, platform.KeyUnknown},
	} {
		s, sink := keySurface(t)
		sendKey(s, gdkKeyPress, c.keyval, c.evdev, 0)
		sendKey(s, gdkKeyRelease, c.keyval, c.evdev, 0)
		checkEvents(t, sink.got, []platform.SurfaceEvent{
			{Kind: platform.RawKeyPressed, Code: c.code, PhysicalKey: c.physical},
			{Kind: platform.RawKeyReleased, Code: c.code, PhysicalKey: c.physical},
		})
	}
}

// A key the layout types no Key for with a place on a US keyboard, as Shift+1
// or AltGr+E: the press comes raw, by position, and the release as it did.
func TestKeyEventTypedNoKey(t *testing.T) {
	s, sink := keySurface(t)
	sendKey(s, gdkKeyPress, '!', 2, gdkShiftMask)
	sendKey(s, gdkKeyRelease, '!', 2, gdkShiftMask)
	checkEvents(t, sink.got, []platform.SurfaceEvent{
		{Kind: platform.RawKeyPressed, PhysicalKey: platform.Key1, Code: platform.CodeDigit1, Mods: platform.ModShift},
		{Kind: platform.KeyReleased, PhysicalKey: platform.Key1, Code: platform.CodeDigit1, Mods: platform.ModShift},
	})
}

// Keys of a name come as they did, with their position added.
func TestKeyEventNamedKeys(t *testing.T) {
	s, sink := keySurface(t)
	if !sendKey(s, gdkKeyPress, 'a', 30, 0) {
		t.Error("a key of a name stops at the surface")
	}
	sendKey(s, gdkKeyRelease, 'a', 30, 0)
	sendKey(s, gdkKeyPress, 0xffb5, 76, 0) // KP_5, with Num Lock
	sendKey(s, gdkKeyPress, 0xff8d, 96, 0) // KP_Enter
	checkEvents(t, sink.got, []platform.SurfaceEvent{
		{Kind: platform.KeyPressed, Key: platform.KeyA, PhysicalKey: platform.KeyA, Code: platform.CodeKeyA},
		{Kind: platform.KeyReleased, Key: platform.KeyA, PhysicalKey: platform.KeyA, Code: platform.CodeKeyA},
		{Kind: platform.KeyPressed, Key: platform.Key5, Code: platform.CodeNumpad5},
		{Kind: platform.KeyPressed, Key: platform.KeyEnter, Code: platform.CodeNumpadEnter},
	})
}

// What the content takes stops at the surface, as Alt before GTK's menu bar
// sees it; what it leaves goes on as it did, and an unknown key is left out.
func TestKeyEventRawTaken(t *testing.T) {
	s, sink := keySurface(t)
	sink.taken = func(ev platform.SurfaceEvent) bool { return ev.Kind == platform.RawKeyPressed }
	if !sendKey(s, gdkKeyPress, 0xffe9, 56, 0) { // Alt_L
		t.Error("Alt taken goes on")
	}
	sink.taken = nil
	if sendKey(s, gdkKeyRelease, 0xffe9, 56, 1<<3) {
		t.Error("Alt left stops")
	}
	sink.got = nil
	if sendKey(s, gdkKeyPress, 0xffffff, 300, 0) || len(sink.got) != 0 {
		t.Errorf("a key of no code: %+v", sink.got)
	}
}

// Auto-repeat comes through for raw keys too, told from held keys.
func TestKeyEventRawRepeat(t *testing.T) {
	s, sink := keySurface(t)
	sendKey(s, gdkKeyPress, 0xff14, 70, 0)
	sendKey(s, gdkKeyPress, 0xff14, 70, 0)
	if !sink.got[1].Repeat || sink.got[0].Repeat {
		t.Errorf("repeats: %+v", sink.got)
	}
}

func checkEvents(t *testing.T, got, want []platform.SurfaceEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d events, want %d:\n%+v", len(got), len(want), got)
	}
	for i, w := range want {
		g := got[i]
		if g.Kind != w.Kind || g.Key != w.Key || g.PhysicalKey != w.PhysicalKey || g.Code != w.Code || g.Mods != w.Mods || g.Repeat != w.Repeat {
			t.Errorf("event %d: %+v\nwant %+v", i, g, w)
		}
	}
}
