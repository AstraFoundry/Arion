//go:build !js

package e2e

import (
	"bytes"
	"image/png"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestContentWindowTransforms exercises the real surface, capture and
// pointer delivery through a nested affine transform.
func TestContentWindowTransforms(t *testing.T) {
	var point atomic.Pointer[[2]float32]
	var local atomic.Pointer[[2]float32]
	var bounds atomic.Pointer[ui.Rect]
	var clicks atomic.Int32
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(250, 250, 250)).Children(func() {
			ui.Box(c).Absolute().Left(60).Top(50).Size(150, 100).TransformOrigin(0, 0).Scale(1.2, 1.1).Children(func() {
				e := ui.Box(c).Absolute().Left(15).Top(20).Size(80, 40).Rotate(30).Background(ui.RGB(220, 30, 40))
				if e.Clicked() {
					clicks.Add(1)
				}
				e.HandleInput(func(ev ui.InputEvent) bool {
					if ev.Kind == ui.InputPointerDown {
						local.Store(&[2]float32{ev.X, ev.Y})
					}
					return false
				})
				e.Draw(func(_ *ui.Painter, _ ui.Rect) {
					x, y := e.LocalToWindow(20, 15)
					point.Store(&[2]float32{x, y})
					r := e.Bounds()
					bounds.Store(&r)
				})
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Transforms", Width: 340, Height: 240, Content: ui.View(view)})
	eventually(t, "transformed geometry", func() bool { return point.Load() != nil })
	p := point.Load()
	if !click(w, float64(p[0]), float64(p[1])) {
		t.Skip("surface click automation unavailable")
	}
	eventually(t, "transformed click", func() bool { return clicks.Load() == 1 })
	if got := local.Load(); got == nil || math.Abs(float64(got[0]-20)) > 1 || math.Abs(float64(got[1]-15)) > 1 {
		t.Fatalf("local pointer coordinates: %v", got)
	}
	r := bounds.Load()
	click(w, float64(r.X+1), float64(r.Y+1))
	mygo.RunOnMain(func() {})
	if clicks.Load() != 1 {
		t.Fatal("bounding-box corner outside rotated element took a click")
	}
	scale := deviceScale(w)
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, ok := surfaceOnScreen(w)
		if !ok {
			t.Skip("surface capture unavailable")
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		red, green, _, _ := img.At(int(float64(p[0])*scale), int(float64(p[1])*scale)).RGBA()
		if red>>8 > 180 && green>>8 < 70 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("transformed interior was not presented: red=%d green=%d", red>>8, green>>8)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestContentWindowPerspective exercises a real surface, its capture and
// pointer delivery through a 3D turn seen in perspective.
func TestContentWindowPerspective(t *testing.T) {
	var point atomic.Pointer[[2]float32]
	var clicks atomic.Int32
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(250, 250, 250)).Children(func() {
			e := ui.Box(c).Absolute().Left(70).Top(50).Size(200, 120).Background(ui.RGB(220, 30, 40)).
				RotateY(45).Perspective(500).BackfaceHidden()
			if e.Clicked() {
				clicks.Add(1)
			}
			e.Draw(func(_ *ui.Painter, _ ui.Rect) {
				// Near its far edge, which perspective moves the most.
				x, y := e.LocalToWindow(185, 60)
				point.Store(&[2]float32{x, y})
			})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Perspective", Width: 340, Height: 240, Content: ui.View(view)})
	eventually(t, "turned geometry", func() bool { return point.Load() != nil })
	p := point.Load()
	if !click(w, float64(p[0]), float64(p[1])) {
		t.Skip("surface click automation unavailable")
	}
	eventually(t, "a click through perspective", func() bool { return clicks.Load() == 1 })
	scale := deviceScale(w)
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, ok := surfaceOnScreen(w)
		if !ok {
			t.Skip("surface capture unavailable")
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		red, green, _, _ := img.At(int(float64(p[0])*scale), int(float64(p[1])*scale)).RGBA()
		if red>>8 > 180 && green>>8 < 70 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the turned element was not presented: red=%d green=%d", red>>8, green>>8)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
