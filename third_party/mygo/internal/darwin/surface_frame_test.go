//go:build darwin

package darwin

import (
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

// frameHandler is a window handler that leaves the paced frames it is
// sent to draw later while wait is true.
type frameHandler struct {
	platform.WindowHandler
	wait  bool
	paced []bool
}

func (h *frameHandler) SurfaceEvent(ev platform.SurfaceEvent) bool {
	if ev.Kind != platform.SurfaceFrame {
		return false
	}
	h.paced = append(h.paced, ev.Paced)
	return ev.Paced && h.wait
}

// TestDeferredFrames checks that the display link keeps running while the
// content leaves a paced frame to draw later, and only then: once FrameNow
// draws it, and once the content draws any other frame instead (as a
// resize's, or a paced one it can draw), which draws what the one left
// would have, as the content then never calls FrameNow for it.
func TestDeferredFrames(t *testing.T) {
	h := &frameHandler{}
	s := &surface{w: &window{h: h}}
	h.wait = true
	s.frameNow(true)
	if !s.deferred {
		t.Fatal("a paced frame left to draw later: not deferred")
	}
	h.wait = false
	s.FrameNow()
	if s.deferred || h.paced[len(h.paced)-1] {
		t.Fatalf("drawn at once: deferred %v, sent paced %v", s.deferred, h.paced[len(h.paced)-1])
	}

	// A frame AppKit displays, not paced, draws the one left.
	h.wait = true
	s.frameNow(true)
	s.frameNow(false)
	if s.deferred {
		t.Fatal("a frame drawn meanwhile left the link running")
	}

	// A paced frame the content draws, as after its renderer was let go.
	s.frameNow(true)
	h.wait = false
	s.frameNow(true)
	if s.deferred {
		t.Fatal("a paced frame drawn left the link running")
	}

	// Refreshes passing while the content waits keep it deferred.
	h.wait = true
	s.frameNow(true)
	s.frameNow(true)
	if !s.deferred {
		t.Fatal("refreshes passing while the content waits: not deferred")
	}
}

// TestDrawSoon checks that a frame asked for after a pause draws from the
// backend's queue, outside AppKit's display of views (#883), once however
// often it is asked for meanwhile, and not at all when another drew
// first, as a resize's does when AppKit displays the view, unless it was
// asked for again after that one.
func TestDrawSoon(t *testing.T) {
	h := &frameHandler{}
	b := &Backend{bySurface: map[id]*surface{}}
	s := &surface{w: &window{b: b, h: h}, view: 1}
	b.bySurface[s.view] = s
	s.drawSoon()
	s.drawSoon()
	if len(h.paced) != 0 {
		t.Fatal("drew at once")
	}
	drainMainQueue()
	if len(h.paced) != 1 || h.paced[0] {
		t.Fatalf("frames drawn from the queue: %v, want one not paced", h.paced)
	}

	s.drawSoon()
	s.frameNow(false)
	drainMainQueue()
	if len(h.paced) != 2 {
		t.Fatalf("drew %d frames, want the one drawn meanwhile alone", len(h.paced)-1)
	}

	// One asked for after the frame drawn meanwhile, as once the window
	// whose first frame AppKit drew while the window was out of sight
	// shows (SurfaceShown), still draws: dropped with the first, what
	// moves would stop for good.
	s.drawSoon()
	s.frameNow(false)
	s.drawSoon()
	drainMainQueue()
	if len(h.paced) != 4 || s.soon {
		t.Fatalf("asked for after the frame drawn meanwhile: drew %d frames from the queue, want 1", len(h.paced)-3)
	}

	// A window closed meanwhile draws none.
	s.drawSoon()
	s.w.closed = true
	drainMainQueue()
	if len(h.paced) != 4 || s.soon {
		t.Fatalf("a closed window drew %d frames (soon %v)", len(h.paced)-4, s.soon)
	}
}
