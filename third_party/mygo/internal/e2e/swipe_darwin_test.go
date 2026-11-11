//go:build darwin

package e2e

import (
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/internal/darwin"
	"github.com/egoist/mygo/ui"
)

// TestRouterSwipe goes back and forward with swipes, as a touchpad swiping
// between pages sends them, and as mice's side buttons do through their
// makers' software (Logi Options+): they come as KeyBack and KeyForward.
func TestRouterSwipe(t *testing.T) {
	var frames atomic.Int32
	var path atomic.Value
	r := ui.NewRouter("/a")
	r.Push("/b")
	r.Transition = ui.TransitionNone
	view := func(c *ui.Context) {
		frames.Add(1)
		r.View(c, func(rt *ui.Route) {
			path.Store(rt.Path())
			ui.Text(c, rt.Path())
		})
	}
	w := newWindow(t, mygo.WindowOptions{Title: "Swipe", Width: 300, Height: 200, Content: ui.View(view)})
	eventually(t, "a frame", func() bool { return frames.Load() > 0 })
	swipe := func(back bool) {
		t.Helper()
		var ok bool
		mygo.RunOnMain(func() { ok = darwin.TestSwipe(w.NativeHandle(), back) })
		if !ok {
			t.Fatal("couldn't make a swipe event of the right direction")
		}
	}
	swipe(true)
	eventually(t, "back to /a", func() bool { return path.Load() == "/a" })
	swipe(false)
	eventually(t, "forward to /b", func() bool { return path.Load() == "/b" })
}
