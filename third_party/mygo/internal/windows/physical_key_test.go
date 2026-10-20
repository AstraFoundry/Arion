//go:build windows

package windows

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestPhysicalKeyDoesNotEscapeIME(t *testing.T) {
	// IMEs (including Korean and Japanese) own VK_PROCESSKEY events.
	// Their scan codes must not turn back into application keys, even on
	// release or repeat. VK_PACKET likewise carries injected Unicode text.
	for _, vk := range []uintptr{vkProcessKey, vkPacket} {
		for _, flags := range []uintptr{0, 1 << 30, 1<<30 | 1<<31} {
			for _, release := range []bool{false, true} {
				k, physical, ok := keyAndPosition(vk, 31<<16|flags, release, platform.ModCtrl|platform.ModAlt)
				if ok || k != platform.KeyUnknown || physical != platform.KeyUnknown {
					t.Errorf("vk=%#x flags=%#x release=%v: (%v, %v, %v), want no key", vk, flags, release, k, physical, ok)
				}
			}
		}
	}
}

func TestPhysicalKeyDistinguishesKeypadSlash(t *testing.T) {
	for _, release := range []bool{false, true} {
		for _, tc := range []struct {
			vk, flags uintptr
			physical  platform.Key
		}{
			{0xBF, 0, platform.KeySlash},
			{0x6F, 1 << 24, platform.KeyUnknown},
		} {
			k, physical, ok := keyAndPosition(tc.vk, 53<<16|tc.flags, release, 0)
			if !ok || k != platform.KeySlash || physical != tc.physical {
				t.Errorf("vk=%#x release=%v: (%v, %v, %v), want (%v, %v, true)", tc.vk, release, k, physical, ok, platform.KeySlash, tc.physical)
			}
		}
	}
}

func TestKeyAndPosition(t *testing.T) {
	for _, tc := range []struct {
		name          string
		vk, scan      uintptr
		mods          platform.Modifiers
		release       bool
		key, physical platform.Key
		ok            bool
	}{
		{"Russian VK_S already identifies S", 'S', 31, platform.ModCtrl, false, platform.KeyS, platform.KeyS, true},
		{"known layout key wins over position", 'Z', 21, platform.ModCtrl, false, platform.KeyZ, platform.KeyY, true},
		{"unknown layout chord", 0xDF, 41, platform.ModCtrl, false, platform.KeyUnknown, platform.KeyBackquote, true},
		{"unknown layout ordinary typing", 0xDF, 41, 0, false, platform.KeyUnknown, platform.KeyBackquote, false},
		{"unknown layout shifted typing", 0xDF, 41, platform.ModShift, false, platform.KeyUnknown, platform.KeyBackquote, false},
		{"release after modifiers lifted", 0xDF, 41, 0, true, platform.KeyUnknown, platform.KeyBackquote, true},
		{"unmapped scan code", 0xDF, 0, platform.ModCtrl, false, platform.KeyUnknown, platform.KeyUnknown, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key, physical, ok := keyAndPosition(tc.vk, tc.scan<<16, tc.release, tc.mods)
			if key != tc.key || physical != tc.physical || ok != tc.ok {
				t.Fatalf("got (%v, %v, %v), want (%v, %v, %v)", key, physical, ok, tc.key, tc.physical, tc.ok)
			}
		})
	}
}
