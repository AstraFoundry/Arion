//go:build linux && (amd64 || arm64)

package linux

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestMenuHides checks that a window with a hidden title bar keeps its menu
// bar out of sight, so that the window buttons stay at the top of the
// window, in the page's title bar (#730).
func TestMenuHides(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts platform.WindowOptions
		full bool
		want bool
	}{
		{"default", platform.WindowOptions{}, false, false},
		{"frameless", platform.WindowOptions{Frameless: true}, false, false},
		{"auto hide", platform.WindowOptions{AutoHideMenu: true}, false, true},
		{"full screen", platform.WindowOptions{}, true, true},
		{"hidden title bar", platform.WindowOptions{TitleBarStyle: "hidden"}, false, true},
		{"hidden inset title bar", platform.WindowOptions{TitleBarStyle: "hiddenInset"}, false, true},
	} {
		o := tc.opts
		w := &window{opts: &o, autoHideMenu: o.AutoHideMenu}
		if tc.full {
			w.state |= stateFullscreen
		}
		if got := w.menuHides(); got != tc.want {
			t.Errorf("%s: menuHides() = %v, want %v", tc.name, got, tc.want)
		}
	}
}
