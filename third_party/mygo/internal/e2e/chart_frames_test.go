//go:build !js

package e2e

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestContentPolylineFrames checks continuous GPU drawing, idle and restart.
func TestContentPolylineFrames(t *testing.T) {
	var frames atomic.Int32
	var moving atomic.Bool
	moving.Store(true)
	w := newWindow(t, mygo.WindowOptions{Title: "Polyline frames", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		if moving.Load() {
			c.AnimationFrame()
		}
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(p *ui.Painter, r ui.Rect) {
			p.StrokePolyline([][2]float32{{20, 30}, {60, 90}, {100, 30}, {140, 90}}, 3, ui.RGB(20, 60, 230))
			frames.Add(1)
		})
	})})
	eventually(t, "continuous frames", func() bool { return frames.Load() >= 10 })
	moving.Store(false)
	w.Invalidate()
	time.Sleep(300 * time.Millisecond)
	before := frames.Load()
	time.Sleep(100 * time.Millisecond)
	if frames.Load() != before {
		t.Fatal("idle chart keeps drawing")
	}
	if how, pix, _, _, ok := glSurface(w); ok && how == "opengl" {
		ink := 0
		for i := 0; i+3 < len(pix); i += 4 {
			if pix[i] > 180 && pix[i+2] < 120 {
				ink++
			}
		}
		if ink < 150 {
			t.Fatal("idle GPU texture lost the line", ink)
		}
	}
	moving.Store(true)
	w.Invalidate()
	eventually(t, "frames after idle", func() bool { return frames.Load() >= before+10 })
}

// TestContentWindowAnimationFrames checks that frames a frame asks for as
// it is drawn keep coming. Named for the CI step that draws native UI with
// OpenGL: on Linux the GtkGLArea renders them, and a render asked for
// within its render signal used to be lost after the first.
func TestContentWindowAnimationFrames(t *testing.T) {
	var frames atomic.Int32
	newWindow(t, mygo.WindowOptions{Title: "Animation frames", Width: 400, Height: 300, Content: ui.View(func(c *ui.Context) {
		c.AnimationFrame()
		frames.Add(1)
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255))
	})})
	eventually(t, "frames while animating", func() bool { return frames.Load() >= 30 })
}
