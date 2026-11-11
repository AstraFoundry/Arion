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
	"github.com/egoist/mygo/internal/wait"
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
	deadline := time.Now().Add(wait.Guard)
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
	deadline := time.Now().Add(wait.Guard)
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

// TestContentWindowTransitionBlur presents an element entering through a
// blur (Motion.Blur), placed by a fraction of its own size
// (TranslatePercent), on the real surface: drawn by the GPU where the
// window has one.
func TestContentWindowTransitionBlur(t *testing.T) {
	var builds atomic.Int32
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(250, 250, 250)).Children(func() {
			// In the second build, in a parent the first showed, it enters,
			// blurred for as long as the test looks.
			if builds.Add(1) == 1 {
				c.Invalidate()
				return
			}
			ui.Box(c).Key("blurred").Absolute().Left(170).Top(120).Size(80, 40).TranslatePercent(-50, -50).
				Background(ui.RGB(220, 30, 40)).Transition(ui.ElementTransition{Duration: time.Hour, Ease: ui.Linear,
				Enter: &ui.Motion{Opacity: 1, Blur: 6}})
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Transition blur", Width: 340, Height: 240, Content: ui.View(view)})
	scale := deviceScale(w)
	deadline := time.Now().Add(wait.Guard)
	for {
		data, ok := surfaceOnScreen(w)
		if !ok {
			t.Skip("surface capture unavailable")
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		at := func(x, y float64) (uint32, uint32) {
			r, g, _, _ := img.At(int(x*scale), int(y*scale)).RGBA()
			return r >> 8, g >> 8
		}
		// Its middle where TranslatePercent centers it, and red spread
		// 4 DIPs beyond its right edge, at 210.
		r, g := at(170, 120)
		or, og := at(214, 120)
		if r > 180 && g < 70 && or > og+20 && og < 240 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the blurred element was not presented: middle %d %d, beyond its edge %d %d", r, g, or, og)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestContentWindowGroupOpacity presents two boxes overlapping in a column
// at half opacity on the real surface, drawn by the GPU where the window
// has one: they fade as one group, as CSS's opacity, the same where they
// overlap as where they do not.
func TestContentWindowGroupOpacity(t *testing.T) {
	red := ui.RGB(255, 0, 0)
	view := func(c *ui.Context) {
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Children(func() {
			ui.Box(c).Absolute().Left(40).Top(40).Size(120, 80).Children(func() {
				ui.Box(c).Absolute().Left(0).Top(0).Size(80, 80).Background(red)
				ui.Box(c).Absolute().Left(40).Top(0).Size(80, 80).Background(red)
			}).Opacity(0.5)
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Group opacity", Width: 340, Height: 240, Content: ui.View(view)})
	scale := deviceScale(w)
	deadline := time.Now().Add(wait.Guard)
	for {
		data, ok := surfaceOnScreen(w)
		if !ok {
			t.Skip("surface capture unavailable")
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		at := func(x, y float64) (uint32, uint32) {
			r, g, _, _ := img.At(int(x*scale), int(y*scale)).RGBA()
			return r >> 8, g >> 8
		}
		// Where they overlap, at 100, and where one is alone, at 60.
		r, g := at(100, 80)
		or, og := at(60, 80)
		near := func(a, b uint32) bool { return a+6 >= b && b+6 >= a }
		if r > 240 && near(g, 127) && or > 240 && near(og, 127) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the group was not presented at its opacity: overlapping %d %d, alone %d %d", r, g, or, og)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
