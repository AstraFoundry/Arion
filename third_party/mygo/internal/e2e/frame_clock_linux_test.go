//go:build linux && (amd64 || arm64)

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// GTK's frame clock waits the refresh interval of the last frame's timings
// before the next frame, 1/60 s while none tells it, and GDK learns it
// only from the compositor: never for a window it composites with OpenGL
// on Wayland, nor on X11 without a compositor. A GtkGLArea drew 60 frames
// a second on a 165 Hz display under Niri (#1219). The frames of a window
// tell the clock the rate of the monitor showing it: 16666 µs under Xvfb
// or weston, whose displays refresh at 60 Hz, where GTK's own guess is
// 16667.
func TestFrameClockFollowsMonitor(t *testing.T) {
	var frames atomic.Int32
	var moving atomic.Bool
	moving.Store(true)
	defer moving.Store(false)
	w := newWindow(t, mygo.WindowOptions{Title: "Frame clock", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		if moving.Load() {
			c.AnimationFrame()
		}
		frames.Add(1)
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255))
	})})
	eventually(t, "frames while animating", func() bool { return frames.Load() >= 10 })
	clock := func() (interval int64, mhz int32) {
		mygo.RunOnMain(func() { interval, mhz = linux.TestFrameClockInterval(w.NativeHandle()) })
		return interval, mhz
	}
	if _, mhz := clock(); mhz <= 0 {
		t.Skip("GDK knows no refresh rate of the display: the frame clock keeps 60 Hz")
	}
	eventually(t, "the monitor's refresh interval in the frame clock", func() bool {
		interval, mhz := clock()
		return interval == 1_000_000_000/int64(mhz)
	})
}
