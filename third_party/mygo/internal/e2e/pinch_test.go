//go:build !js

package e2e

import (
	"math"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// A touchpad's pinch over an element of native UI comes to it as
// InputZoom, with the scale and where the pointer is over the element.
func TestPinchZooms(t *testing.T) {
	type zoom struct{ x, y, scale float32 }
	var got atomic.Pointer[zoom]
	var drawn atomic.Bool
	var missed atomic.Pointer[zoom] // a pinch outside the element
	view := func(c *ui.Context) {
		ui.Box(c).Fill().HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputZoom {
				missed.Store(&zoom{ev.X, ev.Y, ev.Scale})
			}
			return false
		}).Children(func() {
			ui.Box(c).Absolute().Left(40).Top(30).Size(200, 120).HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind != ui.InputZoom {
					return false
				}
				z := zoom{ev.X, ev.Y, ev.Scale}
				if p := got.Load(); p != nil {
					z.scale *= p.scale
				}
				got.Store(&z)
				return true
			}).Draw(func(*ui.Painter, ui.Rect) { drawn.Store(true) })
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Pinch", Width: 320, Height: 220, Content: ui.View(view)})
	eventually(t, "the first frame", drawn.Load)
	want, ok := pinch(w, 100, 90, 1.5)
	if !ok {
		t.Skip("pinch automation unavailable")
	}
	eventually(t, "the pinch", func() bool {
		if m := missed.Load(); m != nil {
			t.Fatalf("the pinch zoomed by %v at (%v, %v) of the window, outside the element", m.scale, m.x, m.y)
		}
		return got.Load() != nil
	})
	z := got.Load()
	if math.Abs(float64(z.scale)-want) > 1e-3 || math.Abs(float64(z.x-60)) > 1 || math.Abs(float64(z.y-60)) > 1 {
		t.Fatalf("the pinch zoomed by %v at (%v, %v), want %v at (60, 60)", z.scale, z.x, z.y, want)
	}
}
