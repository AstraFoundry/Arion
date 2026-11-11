//go:build darwin

package e2e

import (
	"image"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// The squares the windows below show, red on a light background, with
// their margin, as the window fits them.
const (
	squareSide   = 60
	squareMargin = 10
	squaresTall  = squareSide + 2*squareMargin
)

// squaresWidth is the width of a row of n squares.
func squaresWidth(n int) int { return squareMargin + n*(squareSide+squareMargin) }

// squares shows n squares in a row.
func squares(c *ui.Context, n int) {
	ui.Row(c).Fill().Padding(squareMargin).Gap(squareMargin).Background(ui.RGB(235, 235, 250)).Children(func() {
		for range n {
			ui.Box(c).Size(squareSide, squareSide).Background(ui.RGB(220, 30, 30))
		}
	})
}

// firstSquare measures the first red square of a window's content as the
// screen shows it: its left edge and its width, along the middle of its
// rows, and its height; x is -1 without one.
func firstSquare(img *image.RGBA) (x, width, height int) {
	red := func(x, y int) bool {
		c := img.RGBAAt(x, y)
		return c.R > 150 && c.G < 100 && c.B < 100
	}
	b := img.Bounds()
	top, bottom := -1, -1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if red(x, y) {
				if top < 0 {
					top = y
				}
				bottom = y
				break
			}
		}
	}
	if top < 0 {
		return -1, 0, 0
	}
	mid := (top + bottom) / 2
	for x = b.Min.X; x < b.Max.X && !red(x, mid); x++ {
	}
	end := x
	for end < b.Max.X && red(end, mid) {
		end++
	}
	return x, end - x, bottom - top + 1
}

// TestContentWindowResizeRightAfterFrame resizes a window of native UI
// right after a frame that Core Animation presented with its transaction,
// as the frames that change the GPU renderer's layer are, and checks that
// the screen shows the frame of the new size: the squares keep their size
// and place. That frame, presented with the transaction too as AppKit
// displayed the view, never showed: the window kept showing the last
// frame, stretched to its new size, and nothing drew again (#1044).
func TestContentWindowResizeRightAfterFrame(t *testing.T) {
	// shows waits for the screen to show the window w squares wide, its
	// first square in its place and of its size.
	shows := func(t *testing.T, w *mygo.Window, width int) {
		t.Helper()
		var img *image.RGBA
		var ok bool
		if !wait.Poll(func() bool {
			mygo.RunOnMain(func() { img, ok = darwin.TestWindowPixels(w.NativeHandle()) })
			if !ok {
				return true
			}
			x, sw, sh := firstSquare(img)
			return abs(img.Bounds().Dx()-width) <= 1 && abs(x-squareMargin) <= 2 && abs(sw-squareSide) <= 2 && abs(sh-squareSide) <= 2
		}) || !ok {
			if !ok {
				t.Skip("the window server gives no image of the window")
			}
			x, sw, sh := firstSquare(img)
			t.Fatalf("the screen never showed the window %d points wide with its first square at x %d, %d×%d (waited %v): %d wide, the square at x %d, %d×%d",
				width, squareMargin, squareSide, squareSide, wait.Guard, img.Bounds().Dx(), x, sw, sh)
		}
	}

	// A window that fits itself to its content, from its view: the frame
	// that resizes it is drawn at the old size, and presented with the
	// transaction, as the layer takes the new one during it.
	t.Run("from its view", func(t *testing.T) {
		var w *mygo.Window
		n, fit := 5, 5 // squares, set on the main thread
		w = newWindow(t, mygo.WindowOptions{
			Title: "Resized from its view", Width: squaresWidth(n), Height: squaresTall, UseContentSize: true,
			Content: ui.View(func(c *ui.Context) {
				if n != fit {
					fit = n
					w.SetContentSize(squaresWidth(fit), squaresTall)
				}
				squares(c, fit)
			}),
		})
		shows(t, w, squaresWidth(5))
		for _, k := range []int{2, 5} {
			w.Update(func() { n = k })
			shows(t, w, squaresWidth(k))
		}
	})

	// A window that fits itself to its content once a frame measured it,
	// as a panel of a menu bar app does, after drawing nothing for a while:
	// the GPU renderer gave its drawables back, and the frame of the
	// change takes them again, which changes the layer.
	t.Run("after an idle frame", func(t *testing.T) {
		var w *mygo.Window
		var n atomic.Int32 // squares, set by the test
		n.Store(5)
		asked := squaresWidth(5) // the width last asked for, on the main thread
		w = newWindow(t, mygo.WindowOptions{
			Title: "Resized after a frame", Width: asked, Height: squaresTall, UseContentSize: true,
			Content: ui.View(func(c *ui.Context) {
				k := int(n.Load())
				if width := squaresWidth(k); width != asked {
					asked = width
					w.Update(func() { w.SetContentSize(width, squaresTall) })
				}
				squares(c, k)
			}),
		})
		shows(t, w, squaresWidth(5))
		for _, k := range []int{2, 5} {
			idle(t, w)
			n.Store(int32(k))
			w.Invalidate()
			shows(t, w, squaresWidth(k))
		}
	})
}

// idle waits for the GPU renderer of a window to give its drawables back,
// as it does once the window draws nothing for a while.
func idle(t *testing.T, w *mygo.Window) {
	t.Helper()
	var trimmed, ok bool
	if !wait.Poll(func() bool {
		mygo.RunOnMain(func() { trimmed, ok = darwin.TestDrawablesTrimmed(w.NativeHandle()) })
		return trimmed || !ok
	}) {
		t.Fatalf("the window's drawables were never given back (waited %v)", wait.Guard)
	}
	if !ok {
		t.Skip("the window draws without the GPU")
	}
}
