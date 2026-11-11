package ui

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// TestWindowHidden hides and shows a Tester's window: the view reads it
// in the frames that follow.
func TestWindowHidden(t *testing.T) {
	var hidden []bool
	tt := coreNewTester(func(c *context) {
		hidden = append(hidden, c.WindowHidden())
		coreText(c, "clock")
	}, 200, 100)
	if len(hidden) == 0 || hidden[len(hidden)-1] {
		t.Fatalf("a window shown is hidden: %v", hidden)
	}
	tt.SetWindowHidden(true)
	if !hidden[len(hidden)-1] {
		t.Fatalf("hidden, the view reads %v", hidden)
	}
	tt.SetWindowHidden(false)
	if hidden[len(hidden)-1] {
		t.Fatalf("shown again, the view reads %v", hidden)
	}
}

// TestWindowHiddenBuilds checks that a window whose view asks whether it
// is hidden builds it once as it hides, though frames wait while nothing
// of it shows, so that the view can stop what it does, and again as it
// shows; and that a window whose view never asks asks for no frame as it
// hides.
func TestWindowHiddenBuilds(t *testing.T) {
	for _, asks := range []bool{false, true} {
		o := newOccludedHost(t, false)
		var hidden bool
		view := o.rt.view
		o.rt.view = func(c *context) {
			if asks {
				hidden = c.WindowHidden()
			}
			view(c)
		}
		o.frame()
		o.s.occluded = true
		asked, built := o.s.requested, o.builds
		o.event(platform.SurfaceEvent{Kind: platform.SurfaceHidden})
		if (o.s.requested > asked) != asks {
			t.Fatalf("asks %v: hiding asked for %d frames", asks, o.s.requested-asked)
		}
		o.frame()
		if asks && (o.builds != built+1 || !hidden) {
			t.Fatalf("hiding built %d times, the view reads hidden %v", o.builds-built, hidden)
		}
		if !asks && o.builds != built {
			t.Fatalf("a view that never asks built %d times out of sight", o.builds-built)
		}
		// The next frame waits.
		built = o.builds
		o.count = 1
		o.rt.changed()
		o.frame()
		if o.builds != built || !o.rt.held {
			t.Fatalf("asks %v: a change out of sight built %d times, held %v", asks, o.builds-built, o.rt.held)
		}
		o.s.occluded = false
		asked = o.s.requested
		o.event(platform.SurfaceEvent{Kind: platform.SurfaceShown})
		o.frame()
		if o.s.requested == asked || o.builds != built+1 || hidden {
			t.Fatalf("asks %v: shown again, asked for %d frames, built %d, hidden %v", asks, o.s.requested-asked, o.builds-built, hidden)
		}
	}
}
