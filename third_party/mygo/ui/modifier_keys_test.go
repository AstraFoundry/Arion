package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// keysTester builds an element with the focus that notes its keys, taking
// those take says.
func keysTester(take func(InputEvent) bool) (*Tester, *[]InputEvent) {
	var got []InputEvent
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 100).Focusable().AutoFocus().HandleInput(func(ev InputEvent) bool {
			if ev.Kind != InputKeyDown && ev.Kind != InputKeyUp {
				return false
			}
			got = append(got, ev)
			return take != nil && take(ev)
		})
	}, 200, 200)
	return tt, &got
}

// A tap of Super alone comes down and up, and tells its side.
func TestModifierKeySuperTap(t *testing.T) {
	tt, got := keysTester(nil)
	tt.CodeDown(Super, CodeSuperRight)
	tt.CodeUp(0, CodeSuperRight)
	want := []InputEvent{
		{Kind: InputKeyDown, Code: CodeSuperRight, Mods: Super},
		{Kind: InputKeyUp, Code: CodeSuperRight},
	}
	if len(*got) != len(want) {
		t.Fatalf("got %d events, want %d: %+v", len(*got), len(want), *got)
	}
	for i, w := range want {
		g := (*got)[i]
		if g.Kind != w.Kind || g.Code != w.Code || g.Mods != w.Mods || g.Key != KeyUnknown || g.PhysicalKey != KeyUnknown {
			t.Errorf("event %d: %+v, want %+v", i, g, w)
		}
	}
}

// The left and right of a modifier are told apart, with the modifiers held
// as they change.
func TestModifierKeySides(t *testing.T) {
	tt, got := keysTester(nil)
	tt.CodeDown(Shift, CodeShiftLeft)
	tt.CodeDown(Shift, CodeShiftRight)
	tt.CodeUp(Shift, CodeShiftLeft)
	tt.CodeUp(0, CodeShiftRight)
	for i, w := range []struct {
		kind InputKind
		code KeyCode
		mods Modifiers
	}{
		{InputKeyDown, CodeShiftLeft, Shift},
		{InputKeyDown, CodeShiftRight, Shift},
		{InputKeyUp, CodeShiftLeft, Shift},
		{InputKeyUp, CodeShiftRight, 0},
	} {
		if g := (*got)[i]; g.Kind != w.kind || g.Code != w.code || g.Mods != w.mods {
			t.Errorf("event %d: %+v, want %+v", i, g, w)
		}
	}
}

// Keys of no Key reach the element: the lock keys, the keypad's, the 102nd.
func TestKeysOfNoName(t *testing.T) {
	for _, code := range []KeyCode{CodeCapsLock, CodeNumLock, CodeScrollLock, CodePrintScreen, CodePause, CodeIntlBackslash, CodeNumpad5, CodeNumpadAdd, CodeAltRight} {
		tt, got := keysTester(nil)
		tt.CodeDown(0, code)
		tt.CodeUp(0, code)
		if len(*got) != 2 || (*got)[0].Code != code || (*got)[0].Kind != InputKeyDown || (*got)[1].Code != code || (*got)[1].Kind != InputKeyUp {
			t.Errorf("key %#x: %+v", uint16(code), *got)
		}
	}
}

// Taking one tells the backend, which keeps it from the system; leaving it
// goes on as before, and the modifiers held change either way.
func TestModifierKeyTaken(t *testing.T) {
	tt, _ := keysTester(func(ev InputEvent) bool { return ev.Code == CodeAltLeft })
	if !tt.rt.event(platform.SurfaceEvent{Kind: platform.RawKeyPressed, Code: platform.CodeAltLeft, Mods: platform.ModAlt}) {
		t.Error("Alt taken, not told")
	}
	if tt.rt.event(platform.SurfaceEvent{Kind: platform.RawKeyPressed, Code: platform.CodeShiftLeft, Mods: platform.ModShift}) {
		t.Error("Shift left, told taken")
	}
	var mods Modifiers
	tt = coreNewTester(func(c *context) { mods = c.Modifiers() }, 200, 200)
	if tt.rt.event(platform.SurfaceEvent{Kind: platform.RawKeyPressed, Code: platform.CodeControlLeft, Mods: platform.ModCtrl}) {
		t.Error("no element, told taken")
	}
	tt.Frame()
	if mods != Ctrl {
		t.Errorf("modifiers %v, want Ctrl", mods)
	}
}

// Only the element with the focus gets them, and those around it observing.
func TestModifierKeyFocus(t *testing.T) {
	var handled, observed, outside []InputEvent
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(50, 50).HandleInput(func(ev InputEvent) bool {
			outside = append(outside, ev)
			return false
		})
		coreBox(c).ObserveInput(func(ev InputEvent) {
			if ev.Kind == InputKeyDown {
				observed = append(observed, ev)
			}
		}).Children(func() {
			coreBox(c).Size(50, 50).Focusable().AutoFocus().HandleInput(func(ev InputEvent) bool {
				if ev.Kind == InputKeyDown {
					handled = append(handled, ev)
				}
				return false
			})
		})
	}, 200, 200)
	tt.CodeDown(Shift, CodeShiftLeft)
	if len(handled) != 1 || handled[0].Code != CodeShiftLeft {
		t.Errorf("the focus got %+v", handled)
	}
	if len(observed) != 1 || observed[0].Code != CodeShiftLeft {
		t.Errorf("around the focus: %+v", observed)
	}
	if len(outside) != 0 {
		t.Errorf("an element without the focus got %+v", outside)
	}
}

// Keys of a name carry their position too.
func TestInputEventCode(t *testing.T) {
	tt, got := keysTester(nil)
	tt.send(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.KeyEnter, Code: platform.CodeNumpadEnter})
	tt.send(platform.SurfaceEvent{Kind: platform.KeyReleased, Key: platform.KeyEnter, Code: platform.CodeNumpadEnter})
	if len(*got) != 2 {
		t.Fatalf("%+v", *got)
	}
	for _, ev := range *got {
		if ev.Key != KeyEnter || ev.Code != CodeNumpadEnter {
			t.Errorf("%+v", ev)
		}
	}
}

// Modifier keys leave the window's handling alone: a shortcut still runs
// with the modifier its key brings, and a focused text input's text stays.
func TestModifierKeysKeepShortcuts(t *testing.T) {
	saved := 0
	text := "ab"
	tt := coreNewTester(func(c *context) {
		coreTextInput(c, &text).AutoFocus()
		if c.Shortcut(Ctrl, KeyS) {
			saved++
		}
	}, 200, 200)
	tt.CodeDown(Ctrl, CodeControlLeft)
	tt.Key(Ctrl, KeyS)
	tt.CodeUp(0, CodeControlLeft)
	tt.CodeDown(Shift, CodeShiftLeft)
	tt.CodeUp(0, CodeShiftLeft)
	tt.Frame()
	if saved != 1 {
		t.Errorf("the shortcut ran %d times", saved)
	}
	if text != "ab" {
		t.Errorf("the text is %q", text)
	}
}

// The held keys' releases, made up by a backend, tell the position too.
func TestHeldKeysLiftCode(t *testing.T) {
	var h platform.HeldKeys
	var sent []platform.SurfaceEvent
	send := func(ev platform.SurfaceEvent) bool {
		h.Track(ev)
		sent = append(sent, ev)
		return true
	}
	send(platform.SurfaceEvent{Kind: platform.KeyPressed, Key: platform.Key5, Code: platform.CodeNumpad5})
	h.Lift(platform.Key5, platform.KeyUnknown, 0, send)
	if last := sent[len(sent)-1]; last.Kind != platform.KeyReleased || last.Code != platform.CodeNumpad5 {
		t.Errorf("released %+v", last)
	}
}
