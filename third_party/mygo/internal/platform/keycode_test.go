package platform

import "testing"

// The constants are USB HID usage IDs, which apps may rely on.
func TestKeyCodeUsages(t *testing.T) {
	for _, c := range []struct {
		code  KeyCode
		usage uint16
	}{
		{CodeKeyA, 0x04}, {CodeKeyZ, 0x1D}, {CodeDigit1, 0x1E}, {CodeDigit0, 0x27}, {CodeEnter, 0x28},
		{CodeBackslash, 0x31}, {CodeSemicolon, 0x33}, {CodeCapsLock, 0x39}, {CodeF1, 0x3A}, {CodeF12, 0x45},
		{CodePrintScreen, 0x46}, {CodeScrollLock, 0x47}, {CodePause, 0x48}, {CodeArrowUp, 0x52},
		{CodeNumLock, 0x53}, {CodeNumpadEnter, 0x58}, {CodeNumpad1, 0x59}, {CodeNumpad0, 0x62},
		{CodeIntlBackslash, 0x64}, {CodeContextMenu, 0x65}, {CodeNumpadEqual, 0x67}, {CodeF13, 0x68},
		{CodeF24, 0x73}, {CodeAudioVolumeMute, 0x7F}, {CodeAudioVolumeDown, 0x81}, {CodeNumpadComma, 0x85},
		{CodeControlLeft, 0xE0}, {CodeShiftLeft, 0xE1}, {CodeAltLeft, 0xE2}, {CodeSuperLeft, 0xE3},
		{CodeControlRight, 0xE4}, {CodeShiftRight, 0xE5}, {CodeAltRight, 0xE6}, {CodeSuperRight, 0xE7},
	} {
		if uint16(c.code) != c.usage {
			t.Errorf("code %#x, want usage %#x", uint16(c.code), c.usage)
		}
	}
}

// Each system's code of the same key gives the same KeyCode.
func TestKeyCodeTables(t *testing.T) {
	for _, c := range []struct {
		name  string
		want  KeyCode
		evdev uint16
		xt    uint8
		xtExt bool
		mac   uint16
		dom   string
		noMac bool // a Mac's keyboard has no such key
	}{
		{name: "A", want: CodeKeyA, evdev: 30, xt: 0x1E, mac: 0x00, dom: "KeyA"},
		{name: "5", want: CodeDigit5, evdev: 6, xt: 0x06, mac: 0x17, dom: "Digit5"},
		{name: "0", want: CodeDigit0, evdev: 11, xt: 0x0B, mac: 0x1D, dom: "Digit0"},
		{name: "Caps Lock", want: CodeCapsLock, evdev: 58, xt: 0x3A, mac: 0x39, dom: "CapsLock"},
		{name: "left Shift", want: CodeShiftLeft, evdev: 42, xt: 0x2A, mac: 0x38, dom: "ShiftLeft"},
		{name: "right Shift", want: CodeShiftRight, evdev: 54, xt: 0x36, mac: 0x3C, dom: "ShiftRight"},
		{name: "left Control", want: CodeControlLeft, evdev: 29, xt: 0x1D, mac: 0x3B, dom: "ControlLeft"},
		{name: "right Control", want: CodeControlRight, evdev: 97, xt: 0x1D, xtExt: true, mac: 0x3E, dom: "ControlRight"},
		{name: "left Alt", want: CodeAltLeft, evdev: 56, xt: 0x38, mac: 0x3A, dom: "AltLeft"},
		{name: "right Alt", want: CodeAltRight, evdev: 100, xt: 0x38, xtExt: true, mac: 0x3D, dom: "AltRight"},
		{name: "left Super", want: CodeSuperLeft, evdev: 125, xt: 0x5B, xtExt: true, mac: 0x37, dom: "MetaLeft"},
		{name: "right Super", want: CodeSuperRight, evdev: 126, xt: 0x5C, xtExt: true, mac: 0x36, dom: "MetaRight"},
		{name: "keypad 0", want: CodeNumpad0, evdev: 82, xt: 0x52, mac: 0x52, dom: "Numpad0"},
		{name: "keypad 7", want: CodeNumpad7, evdev: 71, xt: 0x47, mac: 0x59, dom: "Numpad7"},
		{name: "keypad /", want: CodeNumpadDivide, evdev: 98, xt: 0x35, xtExt: true, mac: 0x4B, dom: "NumpadDivide"},
		{name: "keypad *", want: CodeNumpadMultiply, evdev: 55, xt: 0x37, mac: 0x43, dom: "NumpadMultiply"},
		{name: "keypad Enter", want: CodeNumpadEnter, evdev: 96, xt: 0x1C, xtExt: true, mac: 0x4C, dom: "NumpadEnter"},
		{name: "Num Lock", want: CodeNumLock, evdev: 69, xt: 0x45, xtExt: true, mac: 0x47, dom: "NumLock"},
		{name: "Pause", want: CodePause, evdev: 119, xt: 0x45, noMac: true, dom: "Pause"},
		{name: "Print Screen", want: CodePrintScreen, evdev: 99, xt: 0x37, xtExt: true, noMac: true, dom: "PrintScreen"},
		{name: "Scroll Lock", want: CodeScrollLock, evdev: 70, xt: 0x46, noMac: true, dom: "ScrollLock"},
		{name: "102nd key", want: CodeIntlBackslash, evdev: 86, xt: 0x56, mac: 0x0A, dom: "IntlBackslash"},
		{name: "Home", want: CodeHome, evdev: 102, xt: 0x47, xtExt: true, mac: 0x73, dom: "Home"},
		{name: "Delete", want: CodeDelete, evdev: 111, xt: 0x53, xtExt: true, mac: 0x75, dom: "Delete"},
		{name: "up", want: CodeArrowUp, evdev: 103, xt: 0x48, xtExt: true, mac: 0x7E, dom: "ArrowUp"},
		{name: "F1", want: CodeF1, evdev: 59, xt: 0x3B, mac: 0x7A, dom: "F1"},
		{name: "F12", want: CodeF12, evdev: 88, xt: 0x58, mac: 0x6F, dom: "F12"},
		{name: "F13", want: CodeF13, evdev: 183, xt: 0x64, mac: 0x69, dom: "F13"},
		{name: "context menu", want: CodeContextMenu, evdev: 127, xt: 0x5D, xtExt: true, mac: 0x6E, dom: "ContextMenu"},
	} {
		if got := KeyCodeForEvdev(c.evdev); got != c.want {
			t.Errorf("%s: evdev %d gives %#x, want %#x", c.name, c.evdev, got, c.want)
		}
		if got := KeyCodeForXT(c.xt, c.xtExt); got != c.want {
			t.Errorf("%s: XT %#x (extended %v) gives %#x, want %#x", c.name, c.xt, c.xtExt, got, c.want)
		}
		if !c.noMac {
			if got := KeyCodeForMac(c.mac); got != c.want {
				t.Errorf("%s: macOS %#x gives %#x, want %#x", c.name, c.mac, got, c.want)
			}
		}
		if got := KeyCodeForDOM(c.dom); got != c.want {
			t.Errorf("%s: DOM %q gives %#x, want %#x", c.name, c.dom, got, c.want)
		}
	}
}

func TestKeyCodeUnmapped(t *testing.T) {
	if got := KeyCodeForEvdev(0); got != CodeUnknown {
		t.Errorf("evdev 0: %#x", got)
	}
	if got := KeyCodeForEvdev(1000); got != CodeUnknown {
		t.Errorf("evdev 1000: %#x", got)
	}
	if got := KeyCodeForXT(0, false); got != CodeUnknown {
		t.Errorf("XT 0: %#x", got)
	}
	if got := KeyCodeForXT(0x10, true); got != CodeUnknown {
		t.Errorf("extended XT 0x10: %#x", got)
	}
	if got := KeyCodeForMac(0x3F); got != CodeUnknown { // Fn
		t.Errorf("macOS Fn: %#x", got)
	}
	for _, code := range []string{"", "Unidentified", "Keya", "Digit", "F", "F0", "F25", "Fn", "Numpad", "NumpadX"} {
		if got := KeyCodeForDOM(code); got != CodeUnknown {
			t.Errorf("DOM %q: %#x", code, got)
		}
	}
}

// Windows's codes that differ from the keyboard's: Alt+Print Screen,
// Ctrl+Pause, and right Shift under a CJK input method.
func TestKeyCodeWindowsQuirks(t *testing.T) {
	if got := KeyCodeForXT(0x54, false); got != CodePrintScreen {
		t.Errorf("Alt+Print Screen: %#x", got)
	}
	if got := KeyCodeForXT(0x46, true); got != CodePause {
		t.Errorf("Ctrl+Pause: %#x", got)
	}
	if got := KeyCodeForXT(0x36, true); got != CodeShiftRight {
		t.Errorf("right Shift with the extended bit: %#x", got)
	}
}

func TestMacFlagsChanged(t *testing.T) {
	const shift, command = 1 << 17, 1 << 20
	for _, c := range []struct {
		name    string
		keyCode uint16
		flags   uint
		code    KeyCode
		down    bool
	}{
		{"left Command down", 0x37, command | 0x08, CodeSuperLeft, true},
		{"left Command up", 0x37, 0, CodeSuperLeft, false},
		{"right Command down", 0x36, command | 0x10, CodeSuperRight, true},
		{"right Shift down, left held", 0x3C, shift | 0x02 | 0x04, CodeShiftRight, true},
		{"right Shift up, left held", 0x3C, shift | 0x02, CodeShiftRight, false},
		{"left Shift up, right held", 0x38, shift | 0x04, CodeShiftLeft, false},
		{"made by a program, no sides", 0x38, shift, CodeShiftLeft, true},
		{"made by a program, up", 0x38, 0, CodeShiftLeft, false},
		{"right Option down", 0x3D, 1<<19 | 0x40, CodeAltRight, true},
		{"right Control down", 0x3E, 1<<18 | 0x2000, CodeControlRight, true},
		{"right Control up, left held", 0x3E, 1<<18 | 0x01, CodeControlRight, false},
		{"Caps Lock on", 0x39, 1 << 16, CodeCapsLock, true},
		{"Caps Lock off", 0x39, 0, CodeCapsLock, true},
		{"Fn", 0x3F, 1 << 23, CodeUnknown, false},
		{"of no modifier", 0x00, command | 0x08, CodeUnknown, false},
	} {
		code, down := MacFlagsChanged(c.keyCode, c.flags)
		if code != c.code || down != c.down {
			t.Errorf("%s: got %#x %v, want %#x %v", c.name, code, down, c.code, c.down)
		}
	}
}
