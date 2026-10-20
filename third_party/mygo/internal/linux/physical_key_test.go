//go:build linux && (amd64 || arm64)

package linux

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/egoist/mygo/internal/platform"
)

func TestEventPhysicalKey(t *testing.T) {
	// The field order/types from GTK 3's gdk/gdkevents.h, without cgo.
	// In particular, the deprecated string pointer precedes hardware_keycode.
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
	var event gdkEventKey
	if offset := unsafe.Offsetof(event.hardwareKeycode); offset != 48 {
		t.Fatalf("hardware_keycode offset = %d, want 48", offset)
	}
	for _, tc := range []struct {
		code uint16
		want platform.Key
	}{
		{0, platform.KeyUnknown},
		{7, platform.KeyUnknown},
		{31 + 8, platform.KeyS},
		{41 + 8, platform.KeyBackquote},
		{53 + 8, platform.KeySlash},
		{98 + 8, platform.KeyUnknown}, // Keypad slash is not the main slash.
	} {
		event.hardwareKeycode = tc.code
		if got := eventPhysicalKey(ptr(unsafe.Pointer(&event))); got != tc.want {
			t.Errorf("hardware_keycode=%d: got %v, want %v", tc.code, got, tc.want)
		}
		runtime.KeepAlive(&event)
	}
}
