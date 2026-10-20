//go:build darwin

package darwin

import (
	"runtime"
	"testing"

	"github.com/ebitengine/purego"

	"github.com/egoist/mygo/internal/platform"
)

func TestKeyAtCodeANSI(t *testing.T) {
	// Carbon Events.h kVK_ANSI_* values.
	want := map[uint16]platform.Key{
		0x00: platform.KeyA, 0x01: platform.KeyS, 0x06: platform.KeyZ, 0x09: platform.KeyV, 0x0B: platform.KeyB,
		0x10: platform.KeyY, 0x2E: platform.KeyM, 0x2D: platform.KeyN, 0x1F: platform.KeyO, 0x23: platform.KeyP,
		0x12: platform.Key1, 0x13: platform.Key2, 0x14: platform.Key3, 0x15: platform.Key4, 0x17: platform.Key5,
		0x16: platform.Key6, 0x1A: platform.Key7, 0x1C: platform.Key8, 0x19: platform.Key9, 0x1D: platform.Key0,
		0x18: platform.KeyEqual, 0x1B: platform.KeyMinus, 0x1E: platform.KeyBracketRight, 0x21: platform.KeyBracketLeft,
		0x27: platform.KeyQuote, 0x29: platform.KeySemicolon, 0x2A: platform.KeyBackslash, 0x2B: platform.KeyComma,
		0x2C: platform.KeySlash, 0x2F: platform.KeyPeriod, 0x32: platform.KeyBackquote,
		// Non-text keys come from macKeys.
		0x24: platform.KeyEnter, 0x31: platform.KeySpace, 0x7B: platform.KeyLeft,
	}
	for code, k := range want {
		if got := keyAtCode(code); got != k {
			t.Errorf("code %#x: got %v, want %v", code, got, k)
		}
	}
	if keyAtCode(0x0A) != platform.KeyUnknown || keyAtCode(0x7F) != platform.KeyUnknown {
		t.Error("unmapped codes must be KeyUnknown")
	}
	// Letters cover all 26 keys exactly once.
	seen := map[platform.Key]bool{}
	for _, k := range ansiKeys {
		if seen[k] {
			t.Errorf("key %v mapped twice", k)
		}
		seen[k] = true
	}
	if len(ansiKeys) != 26+10+11 {
		t.Errorf("ansiKeys has %d entries, want 47", len(ansiKeys))
	}
}

func TestKeyboardLayoutType(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	lib, err := purego.Dlopen("/System/Library/Frameworks/Carbon.framework/Carbon", purego.RTLD_NOW)
	if err != nil {
		t.Fatal(err)
	}
	defer purego.Dlclose(lib)
	var layoutType func(int16) uint32
	purego.RegisterLibFunc(&layoutType, lib, "KBGetLayoutType")
	// Gestalt.h identifies these keyboard types as third-party ANSI/ISO.
	if got := layoutType(40); got != 0x414E5349 { // kKeyboardANSI
		t.Errorf("ANSI keyboard layout type: got %#x", got)
	}
	if got := layoutType(41); got != macISOLayoutType {
		t.Errorf("ISO keyboard layout type: got %#x", got)
	}
}

func TestKeyAtCodeISO(t *testing.T) {
	// ISO swaps the upper-left key's virtual code with the extra key beside
	// left Shift. The latter has no platform.Key counterpart.
	if got := keyAtCodeForLayout(0x0A, macISOLayoutType); got != platform.KeyBackquote {
		t.Errorf("ISO upper-left key: got %v, want KeyBackquote", got)
	}
	if got := keyAtCodeForLayout(0x32, macISOLayoutType); got != platform.KeyUnknown {
		t.Errorf("ISO extra key: got %v, want KeyUnknown", got)
	}
	if got := keyAtCodeForLayout(0x01, macISOLayoutType); got != platform.KeyS {
		t.Errorf("ISO S key: got %v, want KeyS", got)
	}
	for _, layout := range []uint32{0x414E5349, 0} { // kKeyboardANSI, unknown
		if got := keyAtCodeForLayout(0x32, layout); got != platform.KeyBackquote {
			t.Errorf("layout %#x upper-left key: got %v, want KeyBackquote", layout, got)
		}
		if got := keyAtCodeForLayout(0x0A, layout); got != platform.KeyUnknown {
			t.Errorf("layout %#x ISO-only key: got %v, want KeyUnknown", layout, got)
		}
	}
}
