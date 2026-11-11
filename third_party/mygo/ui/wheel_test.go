package ui

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/internal/platform"
)

func TestWheelGlides(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Move(100, 100)
	// A notch of a wheel glides there, as browsers scroll it.
	tt.Wheel(100, 100, 0, 100)
	tt.Frame()
	if s.Y != 0 {
		t.Errorf("a notch scrolled at once, to %v", s.Y)
	}
	tt.Advance(50 * time.Millisecond)
	if s.Y <= 0 || s.Y >= 100 {
		t.Errorf("a notch on its way: at %v", s.Y)
	}
	// Notches on the way add up.
	tt.Wheel(100, 100, 0, 100)
	tt.Advance(time.Second)
	if s.Y != 200 || tt.FramePending() {
		t.Errorf("after two notches: at %v, frames pending %v", s.Y, tt.FramePending())
	}
	// Past the end, they stop there.
	for range 10 {
		tt.Wheel(100, 100, 0, 100)
	}
	tt.Advance(time.Second)
	if s.Y != 800 {
		t.Errorf("past the end: at %v", s.Y)
	}
	// The app setting the offset takes over.
	tt.Wheel(100, 100, 0, -100)
	tt.Advance(50 * time.Millisecond)
	s.Y = 10
	tt.Advance(time.Second)
	if s.Y != 10 {
		t.Errorf("set to 10 on the way: at %v", s.Y)
	}
	// A touchpad scrolls at once.
	tt.Scroll(100, 100, 0, 30)
	if s.Y != 40 {
		t.Errorf("a touchpad scrolled to %v", s.Y)
	}
}

func TestWheelMomentumReachesInput(t *testing.T) {
	var got InputEvent
	tt := coreNewTester(func(c *context) {
		coreBox(c).Fill().HandleInput(func(ev InputEvent) bool {
			got = ev
			return true
		})
	}, 200, 200)
	tt.Move(100, 100)
	tt.send(platform.SurfaceEvent{Kind: platform.PointerScroll, X: 100, Y: 100, DY: -40, Precise: true, Momentum: true})
	if got.Kind != InputScroll || !got.Precise || !got.Momentum {
		t.Errorf("input got kind %v, precise %v, momentum %v; want InputScroll, true, true", got.Kind, got.Precise, got.Momentum)
	}
}

func TestWheelReducedMotion(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.SetPreferences(Preferences{ReduceMotion: true, TextScale: 1})
	tt.Move(100, 100)
	tt.Wheel(100, 100, 0, 100)
	if s.Y != 100 {
		t.Errorf("with less motion, a notch scrolled to %v", s.Y)
	}
}

func TestWheelInnerAtEnd(t *testing.T) {
	var outer, inner ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&outer).Children(func() {
			coreScroll(c).Height(100).Shrink(0).TrackScroll(&inner).Children(func() {
				coreBox(c).Height(150).Shrink(0)
			})
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Move(100, 50)
	// The inner container takes the notches it has room for, the outer
	// the next.
	tt.Wheel(100, 50, 0, 40)
	tt.Wheel(100, 50, 0, 40)
	tt.Advance(time.Second)
	if inner.Y != 50 || outer.Y != 0 {
		t.Errorf("two notches: inner at %v, outer at %v", inner.Y, outer.Y)
	}
	tt.Wheel(100, 50, 0, 40)
	tt.Advance(time.Second)
	if inner.Y != 50 || outer.Y != 40 {
		t.Errorf("three notches: inner at %v, outer at %v", inner.Y, outer.Y)
	}
}

// The app scrolling a List, or an element coming into view, takes over
// from the notches of a wheel on their way, as setting a ScrollState does.
func TestWheelListScrollTakesOver(t *testing.T) {
	var s ListState
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 200, func(i int) { coreBox(c).Height(20) }).Fill()
	}, 200, 200)
	offset := func() float64 { return tt.rt.states[tt.rt.c.root.first.id].scrollY }
	tt.Move(100, 100)
	tt.Wheel(100, 100, 0, 300)
	tt.Advance(30 * time.Millisecond)
	if offset() <= 0 {
		t.Fatalf("a notch on its way: at %v", offset())
	}
	s.ScrollTo(0, Start)
	tt.Advance(time.Second)
	if offset() != 0 {
		t.Errorf("ScrollTo(0) on the way: at %v", offset())
	}
	// A notch after the app's ScrollTo glides as usual.
	tt.Wheel(100, 100, 0, 100)
	tt.Advance(time.Second)
	if math.Abs(offset()-100) > 0.01 {
		t.Errorf("a notch after ScrollTo: at %v", offset())
	}
	tt.Wheel(100, 100, 0, -100)
	tt.Advance(30 * time.Millisecond)
	s.ScrollToEnd()
	tt.Advance(time.Second)
	if end := float64(200*20 - 200); math.Abs(offset()-end) > 0.01 {
		t.Errorf("ScrollToEnd on the way up: at %v, want %v", offset(), end)
	}
}

func TestWheelRevealTakesOver(t *testing.T) {
	reveal := false
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().Children(func() {
			for i := range 50 {
				e := coreBox(c).Height(40).Shrink(0)
				if i == 0 && reveal {
					e.ScrollIntoView()
				}
			}
		})
	}, 200, 200)
	offset := func() float64 { return tt.rt.states[tt.rt.c.root.first.id].scrollY }
	tt.Move(100, 100)
	tt.Wheel(100, 100, 0, 400)
	tt.Advance(30 * time.Millisecond)
	if offset() <= 0 {
		t.Fatalf("a notch on its way: at %v", offset())
	}
	reveal = true
	tt.Frame()
	reveal = false
	tt.Advance(time.Second)
	if offset() != 0 {
		t.Errorf("the first box brought into view on the way: at %v", offset())
	}
}

// TestWheelReverseAtEnd reproduces a wheel gliding to the end of the
// content, and the user turning it the other way while the notches are
// still on their way: the remaining travel must not absorb the turn, or
// the wheel seems to not respond until the old notches run out.
func TestWheelReverseAtEnd(t *testing.T) {
	var s ScrollState
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Fill().TrackScroll(&s).Children(func() {
			coreBox(c).Height(1000).Shrink(0)
		})
	}, 200, 200)
	tt.Move(100, 100)
	// The notches of a fast wheel add up on their way.
	for range 10 {
		tt.Wheel(100, 100, 0, 100)
	}
	tt.Advance(50 * time.Millisecond)
	if s.Y <= 0 || s.Y >= 800 {
		t.Fatalf("setup: gliding at %v, want between 0 and 800", s.Y)
	}
	// Keep scrolling to the end, then turn the wheel the other way at once.
	for range 10 {
		tt.Wheel(100, 100, 0, 100)
	}
	// Advance frame by frame until the content reaches the end, rather than
	// relying on a fixed time that may land it a hair short: arm64 fuses
	// float multiply-add into FMA, so the notches have not run out at 50 ms
	// and the content is still on its way (e.g. at 770). Reaching the end
	// is what matters, not how long it took.
	for i := 0; s.Y < 800 && i < 100; i++ {
		tt.Advance(16 * time.Millisecond)
	}
	if s.Y != 800 {
		t.Fatalf("setup: at the end %v, want 800", s.Y)
	}
	tt.Wheel(100, 100, 0, -100)
	tt.Advance(50 * time.Millisecond)
	if s.Y >= 800 {
		t.Fatalf("the wheel did not turn back at the end: still at %v, want < 800", s.Y)
	}
}
