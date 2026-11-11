//go:build darwin

package darwin

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestCursors gives every cursor an NSCursor: AppKit's own, a private
// one, or the nearest public one where macOS lacks it.
func TestCursors(t *testing.T) {
	load()
	withPool(func() {
		arrow := send(class("NSCursor"), "arrowCursor")
		// Without a window server, as over SSH, AppKit gives no arrow or
		// I-beam, which it draws from the system's cursors: the others
		// must still come, and none may raise (_moveCursor did).
		server := arrow != 0
		if !server {
			t.Log("no window server: the arrow and the I-beam are nil")
		}
		for c := platform.CursorDefault; c <= platform.CursorNoDrop; c++ {
			cur := nsCursor(c)
			if cur == 0 && (server || c != platform.CursorDefault && c != platform.CursorText) {
				t.Errorf("cursor %d: no NSCursor", c)
			}
			if c != platform.CursorDefault && cur == arrow {
				t.Logf("cursor %d: the arrow (macOS has none of its own)", c)
			}
		}
		for _, c := range []platform.Cursor{platform.CursorNoDrop, platform.CursorCell, platform.CursorAllScroll} {
			if nsCursor(c) == arrow {
				t.Errorf("cursor %d is the arrow, though its fallback is public", c)
			}
		}
	})
}
