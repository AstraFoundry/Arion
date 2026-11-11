package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"math"
	"testing"
	"time"
)

func motionTester(view func(*context)) (*Tester, *time.Time) {
	now := time.Unix(1000, 0)
	h := &headless{w: 300, h: 150, scale: 1}
	tt := &Tester{h: h, rt: newRuntime(view, h)}
	tt.rt.clock = func() time.Time { return now }
	tt.settle()
	return tt, &now
}

func TestSpringRetargetKeepsVelocityBetweenFrames(t *testing.T) {
	target := float32(0)
	var value, velocity float32
	var e *node
	tt, now := motionTester(func(c *context) { e = coreBox(c); value = e.Spring("x", target); velocity = e.Velocity("x") })
	target = 100
	tt.Frame()
	*now = now.Add(120 * time.Millisecond)
	tt.Frame()
	if value <= 0 || velocity <= 0 {
		t.Fatalf("spring did not advance: %v at %v/s", value, velocity)
	}
	// Retarget 30 ms after the last drawn frame. The old trajectory must
	// be sampled at the interruption time before its new target is adopted.
	expected := *e.st.motions["x"]
	*now = now.Add(30 * time.Millisecond)
	expected.advance(*now)
	target = -50
	tt.Frame()
	closeFloat(t, value, float32(expected.value))
	closeFloat(t, velocity, float32(expected.velocity))
	*now = now.Add(5 * time.Millisecond)
	tt.Frame()
	if value < float32(expected.value) {
		t.Fatal("retarget discarded forward momentum")
	}
	*now = now.Add(5 * time.Second)
	tt.Frame()
	closeFloat(t, value, -50)
	closeFloat(t, velocity, 0)
	if tt.rt.animating || tt.h.requested.Load() {
		t.Fatal("resting spring kept requesting frames")
	}
}

func TestSpringFrameRateIndependenceAndDamping(t *testing.T) {
	for _, damping := range []float64{5, 2 * math.Sqrt(170), 60} {
		t.Run(time.Duration(damping).String(), func(t *testing.T) {
			m, k, c, _ := SpringOptions{Damping: damping}.physics()
			ax, av := solveSpring(-100, 0, m, k, c, 1)
			bx, bv := -100.0, 0.0
			for range 120 {
				bx, bv = solveSpring(bx, bv, m, k, c, 1.0/120)
			}
			if math.Abs(ax-bx) > 1e-6 || math.Abs(av-bv) > 1e-6 {
				t.Fatalf("frame rate changed spring: %v, %v / %v, %v", ax, av, bx, bv)
			}
		})
	}
}

func TestKeyframeTimingRestartAndAlternate(t *testing.T) {
	var value float32
	run := uint64(0)
	frames := []Keyframe{{At: 0, Value: 0}, {At: 100 * time.Millisecond, Value: 10}, {At: 300 * time.Millisecond, Value: 30, Ease: EaseIn}}
	tt, now := motionTester(func(c *context) { value = coreBox(c).Keyframes("sequence", frames, KeyframeOptions{Run: run}) })
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	closeFloat(t, value, 10)
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	closeFloat(t, value, 12.5)
	// Restart without a jump, from the value on the old trajectory now.
	*now = now.Add(50 * time.Millisecond)
	run++
	tt.Frame()
	closeFloat(t, value, 18.4375)
	*now = now.Add(400 * time.Millisecond)
	tt.Frame()
	closeFloat(t, value, 30)
	if tt.rt.animating {
		t.Fatal("completed sequence requests frames")
	}
	s := sequence{frames: frames, options: KeyframeOptions{Repeat: 1, Alternate: true}, start: time.Unix(0, 0)}
	v, done := s.sample(s.start.Add(500 * time.Millisecond))
	closeFloat(t, v, 10)
	if done {
		t.Fatal("alternate finished early")
	}
	v, done = s.sample(s.start.Add(600 * time.Millisecond))
	closeFloat(t, v, 0)
	if !done {
		t.Fatal("alternate did not finish at its first value")
	}
}

func TestMotionReducedMidFlightAndInfiniteSequences(t *testing.T) {
	target := float32(0)
	run := uint64(0)
	var spring, eased, keyframed float32
	tt, now := motionTester(func(c *context) {
		e := coreBox(c)
		spring = e.Spring("spring", target)
		eased = e.Animate("ease", target, time.Second)
		keyframed = e.Keyframes("keys", []Keyframe{{Value: 0}, {At: time.Second, Value: 20}}, KeyframeOptions{Repeat: -1, Run: run})
	})
	target = 100
	tt.Frame()
	*now = now.Add(200 * time.Millisecond)
	tt.Frame()
	if spring == 100 || eased == 100 || keyframed == 20 {
		t.Fatal("motion completed before reduced motion")
	}
	tt.SetPreferences(Preferences{ReduceMotion: true})
	closeFloat(t, spring, 100)
	closeFloat(t, eased, 100)
	closeFloat(t, keyframed, 20)
	if tt.rt.animating || tt.h.requested.Load() {
		t.Fatal("reduced motion left a frame loop running")
	}
	tt.SetPreferences(Preferences{})
	if keyframed != 20 {
		t.Fatal("preference change replayed a completed sequence")
	}
	run++
	tt.Frame()
	if !tt.rt.animating {
		t.Fatal("explicit replay did not restart sequence")
	}
}

func TestMotionNonMotionAnimatesWithLessMotion(t *testing.T) {
	// A crossfade the view drives says it moves nothing: with less motion,
	// it still fades, and still settles, where a move jumps at once.
	target := float32(0)
	var fade, move, pulse, still float32
	var settled, instant bool
	tt, now := motionTester(func(c *context) {
		if instant {
			c.DisableAnimations()
		}
		e := coreBox(c)
		a := Animation{Duration: 100 * time.Millisecond, Ease: Linear}
		move = e.AnimateTo("move", target, a)
		a.NonMotion = true
		fade, settled = e.AnimateTo("fade", target, a), e.Settled("fade")
		frames := []Keyframe{{Value: 0}, {At: 100 * time.Millisecond, Value: 1}}
		pulse = e.Keyframes("pulse", frames, KeyframeOptions{NonMotion: true})
		still = e.Keyframes("still", frames)
	})
	tt.SetPreferences(Preferences{ReduceMotion: true})
	closeFloat(t, still, 1)
	closeFloat(t, pulse, 0)
	target = 1
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	closeFloat(t, move, 1)
	closeFloat(t, fade, 0.5)
	if pulse <= 0 || pulse >= 1 || settled {
		t.Fatalf("halfway with less motion, the sequence is at %v and the fade settled: %v", pulse, settled)
	}
	if !tt.rt.animating && !tt.h.requested.Load() {
		t.Fatal("a fade with less motion asks for no frames")
	}
	*now = now.Add(time.Second)
	tt.Frame()
	closeFloat(t, fade, 1)
	closeFloat(t, pulse, 1)
	if !settled {
		t.Fatal("the fade did not settle")
	}
	tt.Frame()
	if tt.rt.animating || tt.h.requested.Load() {
		t.Fatal("a settled fade kept asking for frames")
	}
	// DisableAnimations still shows it at once.
	instant, target = true, 0
	tt.Frame()
	closeFloat(t, fade, 0)
}

func TestMotionRejectsInvalidParameters(t *testing.T) {
	for _, view := range []func(*context){
		func(c *context) { coreBox(c).Spring("x", 0, SpringOptions{Mass: -1}) },
		func(c *context) { coreBox(c).Keyframes("x", []Keyframe{{At: time.Second}, {At: 2 * time.Second}}) },
		func(c *context) { coreBox(c).Keyframes("x", []Keyframe{{}, {At: time.Second}, {At: time.Second}}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid motion accepted")
				}
			}()
			coreNewTester(view, 100, 100)
		}()
	}
}

func TestSpringSchedulingWhileOccluded(t *testing.T) {
	target := float32(0)
	var value float32
	tt := newRepaintTester(func(c *context) { e := coreBox(c).Size(30, 20); value = e.Spring("x", target); e.Translate(value, 0) })
	tt.h.hidden = true
	target = 100
	tt.rt.requestFrame()
	_, more := tt.frame(50 * time.Millisecond)
	if more || !tt.rt.held {
		t.Fatal("occluded spring kept drawing")
	}
	tt.h.hidden = false
	tt.rt.event(platform.SurfaceEvent{Kind: platform.SurfaceShown})
	if !tt.h.requested.Load() {
		t.Fatal("shown spring did not resume")
	}
	// The frame out of sight did not draw: the spring sets off as the
	// window shows, and moves on in the frames after.
	built, more := tt.frame(200 * time.Millisecond)
	if !built || !more {
		t.Fatal("spring did not set off after becoming visible")
	}
	if built, more = tt.frame(100 * time.Millisecond); !built || !more || value <= 0 {
		t.Fatal("spring did not advance after becoming visible")
	}
	_, more = tt.frame(5 * time.Second)
	if more || value != 100 {
		t.Fatal("completed spring kept scheduling frames")
	}
}
