package main

import (
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// A demo throws its pieces, which head off in the next frame and go once
// they faded; the stage throws a burst of its own from the pointer.
func TestBursts(t *testing.T) {
	s := newApp()
	tt := ui.NewTester(s.view, windowW, windowH)

	if err := tt.Click("Basic Cannon"); err != nil {
		t.Fatal(err)
	}
	if s.inAir() != 100 || !tt.HasText("100 in the air") {
		t.Fatalf("%d pieces after Basic Cannon, want 100", s.inAir())
	}
	tt.Advance(tick)
	if s.bursts[0].launched.IsZero() {
		t.Fatal("the burst did not head off")
	}
	tt.Advance(200 * tick)
	if len(s.bursts) != 0 {
		t.Errorf("%d bursts after they faded, want 0", len(s.bursts))
	}

	r, ok := tt.Find("Click anywhere here")
	if !ok {
		t.Fatal("no stage")
	}
	tt.ClickAt(r.X, r.Y+100)
	if s.inAir() != 150 {
		t.Fatalf("%d pieces after clicking the stage, want 150", s.inAir())
	}
	if b := s.bursts[0]; math.Abs(float64(b.x-r.X)) > 1 || math.Abs(float64(b.y-r.Y-100)) > 1 {
		t.Errorf("the burst starts at %.0f, %.0f, want the pointer at %.0f, %.0f", b.x, b.y, r.X, r.Y+100)
	}
}

// Pressing a demo again and again lets the oldest pieces go first.
func TestMaxPieces(t *testing.T) {
	s := newApp()
	tt := ui.NewTester(s.view, windowW, windowH)
	for range 8 {
		if err := tt.Click("Realistic Look"); err != nil {
			t.Fatal(err)
		}
	}
	// Realistic Look throws five bursts, of 50, 40, 70, 20 and 20 pieces:
	// the first two go, and 10 pieces of the third.
	if s.inAir() != maxPieces || len(s.bursts[0].pieces) != 60 || s.bursts[0].pieces[0].id != 10 {
		t.Errorf("%d pieces after 1600 thrown, want the last %d", s.inAir(), maxPieces)
	}
}

// Fireworks fire two bursts now and two more every 250 ms for 5 seconds.
func TestFireworks(t *testing.T) {
	s := newApp()
	tt := ui.NewTester(s.view, windowW, windowH)

	if err := tt.Click("Fireworks"); err != nil {
		t.Fatal(err)
	}
	if s.inAir() != 100 || len(s.emitters) != 1 {
		t.Fatalf("%d pieces and %d emitters after Fireworks, want 100 and 1", s.inAir(), len(s.emitters))
	}
	tt.Advance(250 * time.Millisecond)
	if s.inAir() != 100+2*(50*19/20) {
		t.Errorf("%d pieces 250 ms in, want %d", s.inAir(), 100+2*(50*19/20))
	}
	tt.Advance(5 * time.Second)
	if len(s.emitters) != 0 {
		t.Errorf("%d emitters after 5 s, want 0", len(s.emitters))
	}
}

// While the desktop asks for less motion, nothing flies.
func TestReduceMotion(t *testing.T) {
	s := newApp()
	tt := ui.NewTester(s.view, windowW, windowH)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if err := tt.Click("Realistic Look"); err != nil {
		t.Fatal(err)
	}
	if s.inAir() != 0 || !tt.HasText("Confetti rests while the desktop asks for less motion.") {
		t.Errorf("%d pieces with less motion, want 0", s.inAir())
	}
}

// A burst's easing goes from 0 to 1, fast at first as its speed decays.
func TestDecayEase(t *testing.T) {
	ease := decayEase(0.9, 200)
	if ease(0) != 0 || math.Abs(float64(ease(1)-1)) > 1e-6 {
		t.Errorf("ease(0) = %v, ease(1) = %v", ease(0), ease(1))
	}
	// After 10 ticks, 1 - 0.9^10 of the way.
	if got, want := ease(10.0/200), 1-math.Pow(0.9, 10); math.Abs(float64(got)-want) > 1e-4 {
		t.Errorf("ease after 10 ticks = %v, want %v", got, want)
	}
}
