//go:build !js

package e2e

import (
	"math"
	"os"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// TestContentWindowHiddenReadyToShow checks that a window of native UI
// made hidden draws its first frame while hidden, which no paint or
// refresh of the display asks for, and so becomes ready to show:
// OnReadyToShow(w.Show) never showed it (#905). It shows once shown, with
// its content drawn as any window's: on the GPU where it draws there.
func TestContentWindowHiddenReadyToShow(t *testing.T) {
	var frames atomic.Int32
	var firstWidth atomic.Uint64 // of the first frame drawn
	var w *mygo.Window
	readyHidden := make(chan bool, 1)
	mygo.RunOnMain(func() {
		w = newWindow(t, mygo.WindowOptions{Width: 320, Height: 240, Hidden: true, Content: ui.View(func(c *ui.Context) {
			frames.Add(1)
			c.Root().Background(ui.RGB(30, 30, 30)).Draw(func(_ *ui.Painter, r ui.Rect) {
				firstWidth.CompareAndSwap(0, math.Float64bits(float64(r.W)))
			})
		})})
		w.OnReadyToShow(func() {
			readyHidden <- !w.IsVisible()
			w.Show()
		})
	})
	if !wait.For(t, "ready to show", readyHidden) {
		t.Fatalf("ready to show once visible, not while hidden (%d builds)", frames.Load())
	}
	// Drawn at the window's size, not at none before it ever showed.
	if got := math.Float64frombits(firstWidth.Load()); got < 100 {
		t.Errorf("the first frame drawn %v wide", got)
	}
	eventually(t, "the window shown", func() bool { return w.IsVisible() && firstFrameRevealed(w) })
	// With MYGO_GPU=1 Linux draws with OpenGL without a GPU too: the first
	// frame, drawn in memory while the window was hidden, leaves the frames
	// once shown to OpenGL.
	if os.Getenv("MYGO_GPU") == "1" {
		if _, _, _, _, ok := glSurface(w); ok {
			var how string
			var pix []byte
			eventually(t, "a frame drawn in the GtkGLArea", func() bool {
				how, pix, _, _, _ = glSurface(w)
				return how != ""
			})
			if how != "opengl" || len(pix) < 4 {
				t.Fatalf("the surface draws %q, not with OpenGL", how)
			}
			if c := pix[:4]; c[0] != 30 || c[1] != 30 || c[2] != 30 {
				t.Errorf("the background is %v (BGRA) in the GtkGLArea", c)
			}
		}
	}
}

// TestContentViewRunsAfterNewWindow checks that the view of a window of
// native UI that NewWindow shows first runs once NewWindow returned on the
// main thread, so that it can use the window kept from NewWindow: systems
// draw a window's first frame as it shows, as AppKit may in orderFront and
// Windows in the resize of a window maximized (#1053). The window shows
// all the same, with its first frame.
func TestContentViewRunsAfterNewWindow(t *testing.T) {
	for _, maximized := range []bool{false, true} {
		var w *mygo.Window
		var frames, early atomic.Int32
		mygo.RunOnMain(func() {
			w = newWindow(t, mygo.WindowOptions{Title: "After NewWindow", Width: 320, Height: 240, Maximized: maximized, Content: ui.View(func(c *ui.Context) {
				frames.Add(1)
				if w == nil {
					early.Add(1)
					return
				}
				ui.Text(c, w.Title())
			})})
			if n := frames.Load(); n != 0 {
				t.Errorf("maximized %v: %d frames built in NewWindow", maximized, n)
			}
		})
		eventually(t, "the window shown with its first frame", func() bool {
			return frames.Load() > 0 && w.IsVisible() && firstFrameRevealed(w)
		})
		if n := early.Load(); n != 0 {
			t.Errorf("maximized %v: the view ran %d times before NewWindow returned", maximized, n)
		}
		w.Destroy()
	}
}
