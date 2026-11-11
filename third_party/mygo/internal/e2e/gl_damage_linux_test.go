//go:build linux && (amd64 || arm64)

package e2e

import (
	"image"
	"os"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/linux"
	"github.com/egoist/mygo/ui"
)

// GTK 3 tells a Wayland compositor what each paint of a window it
// composites with OpenGL changed in the window's coordinates unscaled,
// where EGL takes the buffer's pixels: at a scale of 2 the compositor
// learned of a change at the top of the window in its lower half, at half
// its size, and showed title buttons that flickered and a shadow missing
// on some sides (#1071). The window tells it what GTK repaints, in the
// surface's coordinates: a box that changes color within the GtkGLArea,
// below GTK's title bar and shadow, is in it. It runs on Wayland at a
// scale of 2 or more (weston --scale=2), with MYGO_GPU=1.
func TestGLDamageWayland(t *testing.T) {
	if os.Getenv("MYGO_GPU") != "1" {
		t.Skip("needs the GtkGLArea: MYGO_GPU=1")
	}
	var frames atomic.Int32
	var lit atomic.Bool
	box := ui.Rect{X: 300, Y: 200, W: 40, H: 30}
	view := func(c *ui.Context) {
		frames.Add(1)
		ui.Box(c).Fill().Background(ui.RGB(255, 255, 255)).Draw(func(pt *ui.Painter, r ui.Rect) {
			color := ui.RGB(200, 200, 200)
			if lit.Load() {
				color = ui.RGB(255, 0, 0)
			}
			pt.Fill(ui.Rect{X: r.X + box.X, Y: r.Y + box.Y, W: box.W, H: box.H}, color, 0)
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Damage", Width: 400, Height: 300, Content: ui.View(view)})
	eventually(t, "a frame with OpenGL", func() bool {
		how, _, _, _, _ := glSurface(w)
		return how == "opengl"
	})
	glDamage := func() (d []image.Rectangle, area image.Point, scale int, ok bool) {
		mygo.RunOnMain(func() { d, area, scale, ok = linux.TestGLDamage(w.NativeHandle()) })
		return d, area, scale, ok
	}
	_, area, scale, ok := glDamage()
	if !ok {
		t.Skip("GTK tells the compositor what changed itself but on Wayland")
	}
	if scale < 2 {
		t.Skipf("GTK tells the compositor what changed itself at a scale of %d", scale)
	}
	want := image.Rect(int(box.X), int(box.Y), int(box.X+box.W), int(box.Y+box.H)).Add(area)
	var damage []image.Rectangle
	before := frames.Load()
	w.Update(func() { lit.Store(true) })
	eventually(t, "the box's change told to the compositor", func() bool {
		d, _, _, _ := glDamage()
		damage = append(damage, d...)
		if frames.Load() == before {
			return false
		}
		for _, r := range damage {
			if want.In(r) {
				return true
			}
		}
		return false
	})
}
