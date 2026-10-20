package platform

import "testing"

func TestKeyForScancode(t *testing.T) {
	// Linux evdev KEY_* and Windows XT set 1 values of the main block.
	want := map[uint16]Key{
		2: Key1, 11: Key0, 12: KeyMinus, 13: KeyEqual,
		16: KeyQ, 25: KeyP, 26: KeyBracketLeft, 27: KeyBracketRight,
		30: KeyA, 31: KeyS, 38: KeyL, 39: KeySemicolon, 40: KeyQuote, 41: KeyBackquote, 43: KeyBackslash,
		44: KeyZ, 50: KeyM, 51: KeyComma, 52: KeyPeriod, 53: KeySlash,
	}
	for code, k := range want {
		if got := KeyForScancode(code); got != k {
			t.Errorf("scancode %d: got %v, want %v", code, got, k)
		}
	}
	// Escape, Backspace, Tab, Enter, Control, Shift, Space, the keypad.
	for _, code := range []uint16{0, 1, 14, 15, 28, 29, 42, 54, 55, 57, 71, 255} {
		if got := KeyForScancode(code); got != KeyUnknown {
			t.Errorf("scancode %d: got %v, want KeyUnknown", code, got)
		}
	}
	seen := map[Key]bool{}
	for code, k := range scancodeKeys {
		if seen[k] {
			t.Errorf("scancode %d: key %v mapped twice", code, k)
		}
		seen[k] = true
	}
	if len(scancodeKeys) != 26+10+11 {
		t.Errorf("scancodeKeys has %d entries, want 47", len(scancodeKeys))
	}
}

func TestKeyByPosition(t *testing.T) {
	for _, c := range []struct {
		name          string
		key, physical Key
		mods          Modifiers
		want          bool
	}{
		{"cyrillic chord", KeyUnknown, KeyS, ModCtrl, true},
		{"cyrillic super", KeyUnknown, KeyS, ModSuper, true},
		{"cyrillic alt", KeyUnknown, KeyS, ModAlt, true},
		{"cyrillic typing", KeyUnknown, KeyS, 0, false},
		{"cyrillic shifted typing", KeyUnknown, KeyS, ModShift, false},
		{"shifted exclamation chord", KeyForRune('!'), Key1, ModCtrl | ModShift, true},
		{"shifted plus chord", KeyForRune('+'), KeyEqual, ModCtrl | ModShift, true},
		{"German sharp S chord", KeyForRune('ß'), KeyMinus, ModCtrl, true},
		{"latin chord keeps the layout key", KeyS, KeyS, ModCtrl, false},
		{"known AZERTY layout key", KeyQ, KeyA, ModCtrl, false},
		{"known QWERTZ layout key", KeyZ, KeyY, ModCtrl, false},
		{"unmapped position", KeyUnknown, KeyUnknown, ModCtrl, false},
	} {
		if got := KeyByPosition(c.key, c.physical, c.mods); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
