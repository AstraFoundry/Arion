//go:build windows && (amd64 || arm64)

package windows

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestServedAtBase checks which base URLs LoadHTML serves its document at:
// not about: URLs, which WebView2 never requests, so that their pages show
// the document as those without a base URL do (#1029).
func TestServedAtBase(t *testing.T) {
	for base, want := range map[string]bool{
		"":                                false,
		"about:blank":                     false,
		"about:blank#top":                 false,
		"About:Blank":                     false,
		"about:srcdoc":                    false,
		"app://localhost/":                true,
		"https://example.com/about:blank": true,
		"file:///C:/app/index.html":       true,
	} {
		if got := servedAtBase(base); got != want {
			t.Errorf("servedAtBase(%q) = %v, want %v", base, got, want)
		}
	}
}

// TestUnhandledKey turns the virtual keys keys.js tells of into the keys
// the content takes, leaving out Tab, modifier keys and unknown keys; the
// script calls the binding Go adds.
func TestUnhandledKey(t *testing.T) {
	if !strings.Contains(keysScript, `"`+keysBinding+`"`) {
		t.Fatalf("keys.js does not call the binding %q: run bun run build", keysBinding)
	}
	w := &window{}
	ev, ok := unhandledKey(w, 'J', platform.ModCtrl|platform.ModShift, true)
	if !ok || ev.Kind != platform.WebViewKey || ev.Key != platform.KeyJ || ev.Mods != platform.ModCtrl|platform.ModShift || !ev.Repeat || ev.WebView != platform.WebView(w) {
		t.Errorf("J with Ctrl+Shift: %+v, %v", ev, ok)
	}
	for _, vk := range []uintptr{0x09, vkShift, vkControl, vkMenu, vkLWin, 0xA0, 0xFF} {
		if _, ok := unhandledKey(w, vk, 0, false); ok {
			t.Errorf("virtual key %#x reached the content", vk)
		}
	}
}
