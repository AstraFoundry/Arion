//go:build windows && (amd64 || arm64)

package e2e

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	win "github.com/egoist/mygo/internal/windows"
	"github.com/egoist/mygo/ui"
)

// TestNativeUIFramelessEdges checks that a frameless window of native UI
// shows its content up to its edges, on the screen: on Windows 10, a white
// line went around it, between the content and the shadow, the visible
// pixel of the borders DWM draws in the non-client area, and the frame
// extended for the shadow along the top (#916). Windows 11 draws a border
// of its own over the outermost pixels, inside which the content shows.
// The content is the issue's: the root has no background, a column filling
// it has one.
func TestNativeUIFramelessEdges(t *testing.T) {
	var frames atomic.Int32
	w := newWindow(t, mygo.WindowOptions{X: 80, Y: 80, Width: 480, Height: 360, Frameless: true, AlwaysOnTop: true,
		Content: ui.View(func(c *ui.Context) {
			frames.Add(1)
			ui.Column(c).Fill().Background(ui.RGB(180, 20, 20))
		})})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	red := func(c [3]uint8) bool {
		near := func(v, want uint8) bool { return int(v) >= int(want)-8 && int(v) <= int(want)+8 }
		return near(c[0], 180) && near(c[1], 20) && near(c[2], 20)
	}
	eventually(t, "the content on the screen", func() bool {
		r, g, b, ok := screenColor(w, 240, 180)
		return ok && red([3]uint8{r, g, b})
	})
	var last string
	t.Cleanup(func() {
		if t.Failed() && last != "" {
			t.Logf("at the edges, %s", last)
		}
	})
	eventually(t, "the content at the window's edges", func() bool {
		var edges [4][3]uint8
		var in int32
		var ok bool
		mygo.RunOnMain(func() { edges, in, ok = win.TestWindowEdges(w.NativeHandle()) })
		if !ok {
			last = "unreadable"
			return false
		}
		last = ""
		for i, name := range []string{"left", "top", "right", "bottom"} {
			if c := edges[i]; !red(c) {
				last += fmt.Sprintf(" %s #%02x%02x%02x", name, c[0], c[1], c[2])
			}
		}
		if last != "" {
			last = fmt.Sprintf("%d pixels in:%s", in, last)
		}
		return last == ""
	})
}
