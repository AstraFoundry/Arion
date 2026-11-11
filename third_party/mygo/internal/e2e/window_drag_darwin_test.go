//go:build darwin

package e2e

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/internal/wait"
	"github.com/egoist/mygo/ui"
)

// TestDragRegionMovesWindow presses a drag region and drags it, of a page
// and of native UI, in a window with a hidden title bar and in a frameless
// one: the window follows the pointer, whether the pointer moved before
// the drag was asked (a quick press and drag, issue #528) or after. The
// window server cannot move a window for a press the app made up, so
// these drags are MyGo's own: the window server's take a hand on a Mac.
func TestDragRegionMovesWindow(t *testing.T) {
	for _, c := range []struct {
		name string
		opts mygo.WindowOptions
	}{
		{"hidden title bar", mygo.WindowOptions{TitleBarStyle: mygo.TitleBarHidden, TitleBarHeight: 40}},
		{"frameless", mygo.WindowOptions{Frameless: true}},
	} {
		c.opts.X, c.opts.Y, c.opts.Width, c.opts.Height = 120, 120, 420, 260
		t.Run(c.name+"/page", func(t *testing.T) {
			w := newWindow(t, c.opts)
			w.Page().LoadHTML(`<body style="margin:0"><div id="bar" style="--app-region:drag;height:40px" onmousedown="window.presses=(window.presses||0)+1"></div></body>`, "")
			waitFor(t, w, "document.getElementById('bar')")
			dragRegion(t, w, func(n int) { waitFor(t, w, fmt.Sprintf("window.presses === %d", n)) })
		})
		t.Run(c.name+"/native UI", func(t *testing.T) {
			var frames atomic.Int32
			opts := c.opts
			opts.Content = ui.View(func(c *ui.Context) {
				frames.Add(1)
				ui.Column(c).Fill().Children(func() {
					ui.Box(c).Height(40).Shrink(0).DragWindow()
					ui.Text(c, "Drag the top")
				})
			})
			w := newWindow(t, opts)
			eventually(t, "a frame", func() bool { return frames.Load() > 0 })
			dragRegion(t, w, func(int) {})
		})
	}
}

// dragRegion drags w by its drag region, 40 points tall at the top of its
// content, slowly then quickly; pressed waits for the content to see its
// nth press.
func dragRegion(t *testing.T, w *mygo.Window, pressed func(n int)) {
	t.Helper()
	w.Focus()
	eventually(t, "the window focused", w.IsFocused)
	waitOnScreen(w)
	h := w.NativeHandle()
	phase := func() (p string) {
		mygo.RunOnMain(func() { p = darwin.TestWindowDrag(h) })
		return p
	}
	mouse := func(steps ...darwin.TestMouseStep) {
		mygo.RunOnMain(func() { darwin.TestAppMouseAt(h, steps...) })
	}
	at := func(x, y int) darwin.TestMouseStep { return darwin.TestMouseStep{X: float64(x), Y: float64(y)} }
	movedTo := func(what string, want mygo.Rectangle) {
		t.Helper()
		var got mygo.Rectangle
		if !wait.Poll(func() bool {
			got = w.ContentBounds()
			return abs(got.X-want.X) <= 1 && abs(got.Y-want.Y) <= 1
		}) {
			t.Fatalf("%s: the window's content is at %+v, want %+v", what, got, want)
		}
	}
	shifted := func(r mygo.Rectangle, dx, dy int) mygo.Rectangle {
		r.X += dx
		r.Y += dy
		return r
	}

	// Slowly: the pointer is still when the drag is asked, which goes to
	// the window server. It does not move the window for a made up press,
	// so the press's drags still come: past a few points, and a few drags
	// for the window server to say it begins, MyGo moves it.
	start := w.ContentBounds()
	x, y := start.X+200, start.Y+20
	down := at(x, y)
	down.Down = true
	mouse(down)
	pressed(1)
	eventually(t, "the drag asked of the window server", func() bool { return phase() == "native" })
	mouse(at(x+1, y+1))
	if got := w.ContentBounds(); got != start {
		t.Errorf("a drag of a point moved the window's content to %+v from %+v", got, start)
	}
	mouse(at(x+30, y+20))
	if got := w.ContentBounds(); got != start {
		t.Errorf("the drag after the ask moved the window's content to %+v from %+v, before the window server could begin", got, start)
	}
	mouse(at(x+60, y+40))
	movedTo("a slow drag", shifted(start, 60, 40))
	up := at(x+60, y+40)
	up.Up = true
	mouse(up)
	if p := phase(); p != "idle" {
		t.Errorf("after the release the press is %s, want idle", p)
	}

	// Quickly: the pointer moves before a page's drag is asked. The
	// window catches up with the pointer once it is, without waiting for
	// another drag, and follows it. Native UI asks with the press, so
	// its window server has the first drags to begin, and MyGo takes
	// over with the third.
	start = w.ContentBounds()
	x, y = start.X+200, start.Y+20
	down = at(x, y)
	down.Down = true
	mouse(down, at(x+6, y+3), at(x+12, y+6), at(x+20, y+10))
	pressed(2)
	eventually(t, "the window moved by MyGo", func() bool { return phase() == "manual" })
	if got := w.ContentBounds(); got == start {
		t.Errorf("the window's content stayed at %+v after the drag was asked", got)
	}
	mouse(at(x-30, y+50))
	movedTo("a quick drag", shifted(start, -30, 50))
	up = at(x-30, y+50)
	up.Up = true
	mouse(up)
	if p := phase(); p != "idle" {
		t.Errorf("after the release the press is %s, want idle", p)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
