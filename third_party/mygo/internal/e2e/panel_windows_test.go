//go:build windows

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
)

// TestPanelAltF4 asks to close a panel on Alt+F4, which has no window menu
// (TestPanelStyles), with a material behind its page.
func TestPanelAltF4(t *testing.T) {
	panel := newWindow(t, mygo.WindowOptions{Panel: true, Frameless: true, Transparent: true, Vibrancy: mygo.VibrancyPopover, Width: 400, Height: 200})
	panel.Page().LoadHTML(`<body style="background:transparent">panel</body>`, "")
	eventually(t, "the panel focused", panel.IsFocused)
	var closes atomic.Int32
	panel.OnClose(func(e *mygo.CloseEvent) { closes.Add(1); e.PreventDefault() })
	mygo.RunOnMain(func() { win.TestPressKeys(0x12, 0x73) }) // VK_MENU, VK_F4
	eventually(t, "Alt+F4 asks to close", func() bool { return closes.Load() > 0 })
}

// A panel the app maximizes fills the work area of its display, as other
// windows do, rather than the whole display: Windows gives a window
// without a maximize box the whole monitor, and the panel, topmost, would
// hide the taskbar. What shows of it, without its invisible borders, is the
// work area; the test reads the window's rectangles in pixels, as Windows
// keeps them, rather than in DIPs through the page or its bounds.
func TestPanelMaximizedInWorkArea(t *testing.T) {
	panel := newWindow(t, mygo.WindowOptions{Panel: true, Frameless: true, Transparent: true, Vibrancy: mygo.VibrancyAcrylic, X: 100, Y: 100, Width: 400, Height: 200})
	panel.Page().LoadHTML(`<body style="background:transparent">panel</body>`, "")
	eventually(t, "the panel shown", panel.IsVisible)
	type rects struct {
		window, shown, client, monitor, work [4]int32
		zoomed                               bool
	}
	read := func() (r rects) {
		mygo.RunOnMain(func() {
			r.window, r.shown, r.client, r.monitor, r.work, r.zoomed = win.TestMaximizedRects(panel.NativeHandle())
		})
		return r
	}
	if r := read(); r.work == r.monitor {
		t.Skip("the monitor has no work area apart from its bounds: no taskbar to keep clear")
	}
	panel.Maximize()
	eventually(t, "the panel maximized", panel.IsMaximized)
	var last rects
	defer func() {
		if t.Failed() {
			t.Logf("last rectangles (left, top, right, bottom, pixels): %+v", last)
		}
	}()
	eventually(t, "the panel in the work area", func() bool {
		last = read()
		return last.zoomed && last.shown == last.work && last.client == last.work
	})
}
