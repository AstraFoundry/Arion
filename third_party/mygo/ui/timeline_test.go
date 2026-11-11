package ui

import (
	"math"
	"testing"
	"time"
)

// timelineTester returns a tester whose frames happen at the time the test
// sets, from a clock it starts, as a Timeline measures its playhead by.
func timelineTester(view func(*context)) (*Tester, *time.Time) {
	now := time.Unix(2000, 0)
	h := &headless{w: 400, h: 300, scale: 1}
	tt := &Tester{h: h, rt: newRuntime(view, h)}
	tt.rt.clock = func() time.Time { return now }
	tt.settle()
	return tt, &now
}

// at moves the tester's clock on by d and draws the frame that follows.
func step(tt *Tester, now *time.Time, d time.Duration) {
	*now = now.Add(d)
	tt.Frame()
}

// resting reports that nothing is asking for another frame.
func resting(t *testing.T, tt *Tester) {
	t.Helper()
	if tt.rt.animating || tt.h.requested.Load() {
		t.Fatal("kept asking for frames")
	}
}

// timelineClickTester is a timelineTester that also collects what it takes
// to find an element and click it.
func timelineClickTester(view func(*context)) (*Tester, *time.Time) {
	now := time.Unix(2000, 0)
	h := &headless{w: 400, h: 300, scale: 1}
	tt := &Tester{h: h, rt: newRuntime(view, h)}
	tt.rt.collect, tt.rt.strict = true, true
	tt.rt.clock = func() time.Time { return now }
	tt.settle()
	return tt, &now
}

func TestTimelinePlaysTweensOneAfterTheOther(t *testing.T) {
	var x, y float32
	var dur time.Duration
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("intro", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		y = tl.Tween(Tween{From: -50, To: 50, Duration: 300 * time.Millisecond, Ease: Linear})
		dur = tl.Duration()
	})
	if dur != 700*time.Millisecond {
		t.Fatalf("Duration = %v, want 700ms", dur)
	}
	closeFloat(t, x, 0)
	closeFloat(t, y, -50)
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 50)
	closeFloat(t, y, -50)
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 100)
	closeFloat(t, y, -50) // the second starts where the first ends
	step(tt, now, 150*time.Millisecond)
	closeFloat(t, x, 100)
	closeFloat(t, y, 0)
}

func TestTimelinePlacesTweens(t *testing.T) {
	var after, overlap, with, abs float32
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("places", TimelineOptions{})
		after = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		// 250ms to 450ms: 150ms before the one before it ends.
		with = tl.Tween(Tween{From: 0, To: 100, Duration: 200 * time.Millisecond, Ease: Linear},
			AfterPrevious(-150*time.Millisecond))
		// 550ms to 750ms: 100ms after the one before it ends.
		overlap = tl.Tween(Tween{From: 0, To: 100, Duration: 200 * time.Millisecond, Ease: Linear},
			AfterPrevious(100*time.Millisecond))
		// 550ms to 950ms: where the one before it starts.
		abs = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear}, WithPrevious(0))
	})
	closeFloat(t, after, 0)
	closeFloat(t, overlap, 0)
	closeFloat(t, with, 0)
	closeFloat(t, abs, 0)
	step(tt, now, 250*time.Millisecond)
	closeFloat(t, after, 62.5)
	closeFloat(t, overlap, 0)
	closeFloat(t, with, 0)
	closeFloat(t, abs, 0)
	step(tt, now, 100*time.Millisecond) // 350ms
	closeFloat(t, overlap, 0)
	closeFloat(t, with, 50)
	closeFloat(t, abs, 0)
	step(tt, now, 100*time.Millisecond) // 450ms
	closeFloat(t, overlap, 0)
	closeFloat(t, with, 100)
	closeFloat(t, abs, 0)
	step(tt, now, 100*time.Millisecond) // 550ms
	closeFloat(t, overlap, 0)
	closeFloat(t, abs, 0)
	step(tt, now, 100*time.Millisecond) // 650ms
	closeFloat(t, overlap, 50)
	closeFloat(t, abs, 25)
	step(tt, now, 100*time.Millisecond) // 750ms
	closeFloat(t, overlap, 100)
	closeFloat(t, abs, 50)
}

func TestTimelineAbsolutePlacement(t *testing.T) {
	var abs, delayed float32
	var dur time.Duration
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("abs", TimelineOptions{})
		abs = tl.Tween(Tween{From: 0, To: 100, Duration: 200 * time.Millisecond, Ease: Linear}, AtTime(200*time.Millisecond))
		// Placed at 500ms, after the 200ms to 400ms of the one before it.
		delayed = tl.Tween(Tween{From: 10, To: 20, Duration: 200 * time.Millisecond, Ease: Linear},
			AtTime(500*time.Millisecond))
		dur = tl.Duration()
	})
	if dur != 700*time.Millisecond {
		t.Fatalf("Duration = %v, want 700ms", dur)
	}
	closeFloat(t, abs, 0)
	closeFloat(t, delayed, 10)
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, abs, 0)               // it has not started yet
	step(tt, now, 200*time.Millisecond) // 300ms
	closeFloat(t, abs, 50)
	closeFloat(t, delayed, 10)
	step(tt, now, 200*time.Millisecond) // 500ms
	closeFloat(t, abs, 100)
	closeFloat(t, delayed, 10)          // the delay is over at 500ms
	step(tt, now, 100*time.Millisecond) // 600ms
	closeFloat(t, delayed, 15)
}

func TestTimelineLabels(t *testing.T) {
	var a, b, cv float32
	var dur time.Duration
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("labels", TimelineOptions{})
		a = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		tl.Label("intro") // where a ends, at 400ms
		b = tl.Tween(Tween{From: 0, To: 100, Duration: 200 * time.Millisecond, Ease: Linear}, AtLabel("intro", 0))
		tl.Label("far", AtTime(800*time.Millisecond))
		cv = tl.Tween(Tween{From: 0, To: 100, Duration: 200 * time.Millisecond, Ease: Linear},
			AtLabel("far", 100*time.Millisecond))
		dur = tl.Duration()
	})
	if dur != 1100*time.Millisecond {
		t.Fatalf("Duration = %v, want 1.1s", dur)
	}
	closeFloat(t, b, 0)
	closeFloat(t, cv, 0)
	step(tt, now, 400*time.Millisecond)
	closeFloat(t, a, 100)
	closeFloat(t, b, 0)
	closeFloat(t, cv, 0)
	step(tt, now, 200*time.Millisecond) // 600ms
	closeFloat(t, b, 100)
	closeFloat(t, cv, 0)
	step(tt, now, 400*time.Millisecond) // 1s, 100ms into the cv tween
	closeFloat(t, cv, 50)
}

func TestTimelineEase(t *testing.T) {
	var linear, squared, plain float32
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("ease", TimelineOptions{})
		linear = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		squared = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: func(t float32) float32 { return t * t }},
			AtTime(0))
		plain = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond}, AtTime(0))
	})
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, linear, 50)
	closeFloat(t, squared, 25)             // the ease, not the time, shapes it
	closeFloat(t, plain, 100*EaseOut(0.5)) // no Ease eases out, as an Animation
}

func TestTimelinePlacesRelativeToThePrevious(t *testing.T) {
	var first, overlap, with float32
	var dur time.Duration
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("relative", TimelineOptions{})
		first = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear}, AtTime(100*time.Millisecond))
		// 300ms to 500ms: a negative offset overlaps the one before it.
		overlap = tl.Tween(Tween{From: 0, To: 100, Duration: 200 * time.Millisecond, Ease: Linear},
			AfterPrevious(-200*time.Millisecond))
		// 400ms to 600ms: 100ms after the one before it starts.
		with = tl.Tween(Tween{From: 0, To: 100, Duration: 200 * time.Millisecond, Ease: Linear},
			WithPrevious(100*time.Millisecond))
		dur = tl.Duration()
	})
	if dur != 600*time.Millisecond {
		t.Fatalf("Duration = %v, want 600ms", dur)
	}
	step(tt, now, 400*time.Millisecond)
	closeFloat(t, first, 75)
	closeFloat(t, overlap, 50)
	closeFloat(t, with, 0)
	step(tt, now, 100*time.Millisecond) // 500ms
	closeFloat(t, overlap, 100)
	closeFloat(t, with, 50)
}

func TestTimelineTimeAndProgress(t *testing.T) {
	var tm time.Duration
	var p float32
	var done bool
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("progress", TimelineOptions{})
		tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		tm, p, done = tl.Time(), tl.Progress(), tl.Settled()
	})
	if tm != 0 || p != 0 || done {
		t.Fatalf("a new timeline is at %v, %v, done %v", tm, p, done)
	}
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, float32(tm), float32(100*time.Millisecond))
	closeFloat(t, p, 0.25)
	step(tt, now, 500*time.Millisecond) // past the end
	closeFloat(t, float32(tm), float32(400*time.Millisecond))
	closeFloat(t, p, 1)
	if !done {
		t.Fatal("a timeline past its end is not done")
	}
	resting(t, tt)
}

func TestTimelinePauseAndResume(t *testing.T) {
	var x float32
	paused := false
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("pause", TimelineOptions{Paused: paused})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 25)
	paused = true
	tt.Frame()
	closeFloat(t, x, 25)
	resting(t, tt) // a paused timeline asks for no frames
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 25) // time passes and nothing moves
	paused = false
	tt.Frame() // the frame that lets the pause go plays no time
	closeFloat(t, x, 25)
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 50) // it goes on where it was held
}

func TestTimelineSeekAndProgressHoldThePlayhead(t *testing.T) {
	var x float32
	var seek *time.Duration
	var progress *float32
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("seek", TimelineOptions{Seek: seek, Progress: progress})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 25)
	// A seek holds the playhead where the scrubber asks, and asks for no
	// frames while it does.
	where := 300 * time.Millisecond
	seek = &where
	tt.Frame()
	closeFloat(t, x, 75)
	resting(t, tt)
	step(tt, now, 200*time.Millisecond) // time passes and it stays there
	closeFloat(t, x, 75)
	// It goes on from where it was seeked to: the frame that lets it go
	// plays no time, as one that lets a pause go.
	seek = nil
	step(tt, now, 50*time.Millisecond)
	closeFloat(t, x, 75)
	step(tt, now, 50*time.Millisecond)
	closeFloat(t, x, 87.5)
	// Progress does the same, by part of the duration.
	seek = nil
	part := float32(0.25)
	progress = &part
	tt.Frame()
	closeFloat(t, x, 25)
	resting(t, tt)
	progress = nil
	tt.Frame()
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 50)
}

func TestTimelineReverse(t *testing.T) {
	var x float32
	var done bool
	reverse := false
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("reverse", TimelineOptions{Reverse: reverse})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		done = tl.Settled()
	})
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 25)
	reverse = true
	tt.Frame()
	closeFloat(t, x, 25) // it turns where the playhead is, without a jump
	step(tt, now, 50*time.Millisecond)
	closeFloat(t, x, 12.5)
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 0) // it comes to rest at its start
	if !done {
		t.Fatal("a timeline played back to its start is not done")
	}
	resting(t, tt)
	reverse = false
	tt.Frame()
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 25) // and plays forward again from there
}

func TestTimelineReversesFromItsEnd(t *testing.T) {
	// A timeline that came to rest at its end plays back from it, as a
	// panel that opened closes, rather than jump to its start.
	var x float32
	reverse := false
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("close", TimelineOptions{Reverse: reverse})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	step(tt, now, 400*time.Millisecond)
	closeFloat(t, x, 100)
	// Frames something else asked for go by: they are not time it plays.
	step(tt, now, time.Second)
	reverse = true
	tt.Frame()
	closeFloat(t, x, 100)
	if !tt.rt.animating {
		t.Fatal("a timeline playing back asks for no frames")
	}
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 75)
}

func TestTimelineStartsReversedAtItsStart(t *testing.T) {
	// A timeline that plays backward from the first frame has played
	// nothing to play back: it rests at its start, until it plays forward,
	// and a Run that plays it over while reversed plays it from its end.
	var x float32
	reverse, run := true, uint64(0)
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("open", TimelineOptions{Reverse: reverse, Run: run})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	closeFloat(t, x, 0)
	resting(t, tt)
	reverse = false
	tt.Frame()
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 50)
	reverse, run = true, 1
	tt.Frame()
	closeFloat(t, x, 100)
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 75)
}

func TestTimelineRepeatAndAlternate(t *testing.T) {
	var x float32
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("repeat", TimelineOptions{Repeat: 1})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 50)
	step(tt, now, 200*time.Millisecond) // it plays again from the start
	closeFloat(t, x, 0)
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 50)
	step(tt, now, 200*time.Millisecond) // two plays, and it rests at the end
	closeFloat(t, x, 100)
	resting(t, tt)

	var y float32
	var tm time.Duration
	tt, now = timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("alternate", TimelineOptions{Repeat: 1, Alternate: true})
		y = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		tm = tl.Time()
	})
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, y, 50)
	step(tt, now, 200*time.Millisecond) // the second play goes backward
	closeFloat(t, y, 100)
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, y, 50)
	step(tt, now, 200*time.Millisecond) // two plays, and it rests where it began
	closeFloat(t, y, 0)
	if tm != 0 {
		t.Fatalf("Time = %v, want 0 at the end of an alternated timeline", tm)
	}
	resting(t, tt)
}

func TestTimelineRepeatForever(t *testing.T) {
	var x float32
	var done bool
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("forever", TimelineOptions{Repeat: RepeatForever})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		done = tl.Settled()
	})
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 25)
	for range 20 {
		step(tt, now, 200*time.Millisecond)
	}
	if done {
		t.Fatal("a timeline that repeats for ever is done")
	}
	if !tt.rt.animating {
		t.Fatal("a timeline that repeats for ever stopped asking for frames")
	}
}

func TestTimelineStagger(t *testing.T) {
	var v [3]float32
	var dur time.Duration
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("stagger", TimelineOptions{})
		tl.Label("rows", AtTime(0))
		for i := range v {
			v[i] = tl.Tween(Tween{From: 0, To: 1, Duration: 100 * time.Millisecond, Ease: Linear},
				AtLabel("rows", time.Duration(i)*50*time.Millisecond))
		}
		dur = tl.Duration()
	})
	if dur != 200*time.Millisecond {
		t.Fatalf("Duration = %v, want 200ms", dur)
	}
	closeFloat(t, v[0], 0)
	closeFloat(t, v[1], 0)
	closeFloat(t, v[2], 0)
	step(tt, now, 75*time.Millisecond)
	closeFloat(t, v[0], 0.75)
	closeFloat(t, v[1], 0.25)
	closeFloat(t, v[2], 0)
	step(tt, now, 50*time.Millisecond) // 125ms
	closeFloat(t, v[0], 1)
	closeFloat(t, v[1], 0.75)
	closeFloat(t, v[2], 0.25)
	step(tt, now, 100*time.Millisecond) // 225ms, past the end
	closeFloat(t, v[0], 1)
	closeFloat(t, v[1], 1)
	closeFloat(t, v[2], 1)
	resting(t, tt)
}

func TestTimelineCompletion(t *testing.T) {
	var x float32
	var done, arrived bool
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("done", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 1, Duration: 100 * time.Millisecond, Ease: Linear})
		done, arrived = tl.Settled(), tl.Arrived()
	})
	if done || arrived {
		t.Fatal("a new timeline is done or arrived")
	}
	step(tt, now, 50*time.Millisecond)
	if done || arrived {
		t.Fatal("a timeline halfway through is done or arrived")
	}
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 1)
	if !done || !arrived {
		t.Fatalf("the frame it came to rest in: done %v, arrived %v", done, arrived)
	}
	resting(t, tt) // and it asks for no more frames
	step(tt, now, 100*time.Millisecond)
	if !done {
		t.Fatal("a timeline that came to rest is not done")
	}
	if arrived {
		t.Fatal("Arrived is true in a frame after the one it came to rest in")
	}
}

func TestTimelineRestartsWhenRunChanges(t *testing.T) {
	var x float32
	var run uint64
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("run", TimelineOptions{Run: run})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	step(tt, now, 300*time.Millisecond)
	closeFloat(t, x, 75)
	run = 1
	tt.Frame()
	closeFloat(t, x, 0) // it plays again from the start
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 50)
	// A run that stays the same does not restart it.
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 75)
}

func TestTimelineFollowsAChangedDefinition(t *testing.T) {
	var x, y float32
	two := false
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("definition", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		y = 0
		if two {
			y = tl.Tween(Tween{From: 0, To: 10, Duration: 200 * time.Millisecond, Ease: Linear})
		}
	})
	step(tt, now, 300*time.Millisecond)
	closeFloat(t, x, 75)
	// The view adds a tween: the playhead goes on, and the new tween shows
	// what its own time says, without jumping.
	two = true
	tt.Frame()
	closeFloat(t, x, 75)
	closeFloat(t, y, 0)
	step(tt, now, 150*time.Millisecond)
	closeFloat(t, x, 100)
	closeFloat(t, y, 2.5)
}

func TestTimelineLessMotion(t *testing.T) {
	var x float32
	var done bool
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("less", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		done = tl.Settled()
	})
	closeFloat(t, x, 0)
	tt.SetPreferences(Preferences{ReduceMotion: true})
	closeFloat(t, x, 100) // where it would have ended, at once
	if !done {
		t.Fatal("a timeline with less motion is not done")
	}
	resting(t, tt)
	// It stays at its end while the view goes on building it.
	step(tt, now, time.Second)
	closeFloat(t, x, 100)
	resting(t, tt)
}

func TestTimelineNonMotionPlaysWithLessMotion(t *testing.T) {
	// A timeline of fades plays with less motion; one that moves is at
	// once where it rests.
	var fade, x float32
	var done bool
	tt, now := timelineTester(func(c *context) {
		e := coreBox(c)
		tl := e.Timeline("fade", TimelineOptions{NonMotion: true})
		fade = tl.Tween(Tween{From: 0, To: 1, Duration: 400 * time.Millisecond, Ease: Linear})
		done = tl.Settled()
		x = e.Timeline("move", TimelineOptions{}).Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	tt.SetPreferences(Preferences{ReduceMotion: true})
	closeFloat(t, x, 100)
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, fade, 0.5)
	if done {
		t.Fatal("a timeline of fades is done halfway")
	}
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, fade, 1)
	if !done {
		t.Fatal("a timeline of fades did not finish")
	}
	resting(t, tt)
}

func TestTimelineLessMotionKeepsDirectManipulation(t *testing.T) {
	// A scrubber and a pause still move and hold the playhead with less
	// motion: only the timeline playing on its own goes to rest at once.
	var x float32
	var o TimelineOptions
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("less-held", o)
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	tt.SetPreferences(Preferences{ReduceMotion: true})
	closeFloat(t, x, 100)
	at := 100 * time.Millisecond
	o.Seek = &at
	step(tt, now, 16*time.Millisecond)
	closeFloat(t, x, 25) // the scrubber moves it, even from where it rested
	part := float32(0.5)
	o.Seek, o.Progress = nil, &part
	step(tt, now, 16*time.Millisecond)
	closeFloat(t, x, 50)
	o.Progress, o.Paused = nil, true
	step(tt, now, time.Second)
	closeFloat(t, x, 50) // paused, it holds where the scrubber left it
	resting(t, tt)
	o.Paused = false
	step(tt, now, 16*time.Millisecond)
	closeFloat(t, x, 100) // let go, it is at once where it comes to rest
	resting(t, tt)
}

func TestTimelineLessMotionRestarts(t *testing.T) {
	var x float32
	var run uint64
	tt, _ := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("less-run", TimelineOptions{Run: run})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	tt.SetPreferences(Preferences{ReduceMotion: true})
	closeFloat(t, x, 100)
	run = 1
	tt.Frame()
	closeFloat(t, x, 100)
	resting(t, tt)
}

func TestTimelineWithoutTweensStops(t *testing.T) {
	var done bool
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("empty", TimelineOptions{})
		done = tl.Settled()
	})
	// Nothing is known to play yet: the next frame is the one that shows
	// the timeline has nothing to play.
	step(tt, now, time.Millisecond)
	if !done {
		t.Fatal("a timeline with no tweens never came to rest")
	}
	resting(t, tt)
}

func TestTimelineGoesWithTheView(t *testing.T) {
	var x float32
	build := true
	var id uint64
	tt, now := timelineTester(func(c *context) {
		if !build {
			return
		}
		e := coreBox(c)
		id = e.id
		x = e.Timeline("gone", TimelineOptions{}).Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 50)
	build = false
	tt.Frame()
	if s := tt.rt.states[id]; s != nil && s.timelines != nil {
		t.Fatal("a timeline the view stopped building stayed")
	}
	resting(t, tt)
}

func TestTimelinesOfTwoElements(t *testing.T) {
	// Two cards built by the same code keep timelines of their own under
	// the same key: playing one over leaves the other as it was.
	var v [2]float32
	var runs [2]uint64
	tt, now := timelineTester(func(c *context) {
		for i := range v {
			tl := coreBox(c).Timeline("card", TimelineOptions{Run: runs[i]})
			v[i] = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		}
	})
	step(tt, now, 200*time.Millisecond)
	runs[0] = 1
	tt.Frame()
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, v[0], 25)
	closeFloat(t, v[1], 75)
}

func TestTimelineLetsASeekGoWhereItHeld(t *testing.T) {
	// A held timeline asks for no frames: letting it go after a while
	// with none plays on from where it held, as a pause does.
	var x float32
	var seek *time.Duration
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("held", TimelineOptions{Seek: seek})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	where := 100 * time.Millisecond
	seek = &where
	tt.Frame()
	closeFloat(t, x, 25)
	resting(t, tt)
	*now = now.Add(5 * time.Second)
	seek = nil
	tt.Frame()
	closeFloat(t, x, 25)
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 50)
}

func TestTimelineLessMotionRestsWhereItWould(t *testing.T) {
	// With less motion a timeline is at once where it would have come to
	// rest: its start when it plays back to it, or alternates an even
	// number of times, as Keyframes are.
	for _, tc := range []struct {
		name string
		o    TimelineOptions
		want float32
	}{
		{"forward", TimelineOptions{}, 100},
		{"reversed", TimelineOptions{Reverse: true}, 0},
		{"alternating twice", TimelineOptions{Repeat: 1, Alternate: true}, 0},
		{"alternating three times", TimelineOptions{Repeat: 2, Alternate: true}, 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var x, p float32
			tt, now := timelineTester(func(c *context) {
				tl := coreBox(c).Timeline("less", tc.o)
				x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
				p = tl.Progress()
			})
			tt.SetPreferences(Preferences{ReduceMotion: true})
			step(tt, now, time.Millisecond)
			closeFloat(t, x, tc.want)
			closeFloat(t, p, tc.want/100)
			resting(t, tt)
		})
	}
}

func TestTimelineGoesOnAfterFramesAtRest(t *testing.T) {
	// Frames something else asks for while a timeline rests at its end are
	// not time it played: a longer timeline goes on from that end.
	var x float32
	span := 400 * time.Millisecond
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("rest", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: span, Ease: Linear})
	})
	step(tt, now, 400*time.Millisecond)
	step(tt, now, time.Second)
	span = 800 * time.Millisecond
	tt.Frame()
	closeFloat(t, x, 50)
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 75)
}

func TestTimelineNilIsSafe(t *testing.T) {
	var tl *Timeline
	if got := tl.Tween(Tween{From: 3, To: 9, Duration: time.Second, Ease: Linear}); got != 3 {
		t.Fatalf("a nil Timeline showed %v, want its From", got)
	}
	tl.Label("nowhere")
	if tl.Duration() != 0 || tl.Time() != 0 || tl.Progress() != 0 || tl.Settled() || tl.Arrived() {
		t.Fatal("a nil Timeline answered")
	}
}

func TestTimelinePanics(t *testing.T) {
	cases := []struct {
		name string
		view func(*context)
	}{
		{"a negative duration", func(c *context) {
			coreBox(c).Timeline("t", TimelineOptions{}).Tween(Tween{From: 0, To: 1, Duration: -time.Millisecond, Ease: Linear})
		}},
		{"Seek and Progress", func(c *context) {
			at, part := time.Duration(0), float32(0)
			coreBox(c).Timeline("t", TimelineOptions{Seek: &at, Progress: &part})
		}},
		{"one key twice in a build", func(c *context) {
			b := coreBox(c)
			b.Timeline("t", TimelineOptions{})
			b.Timeline("t", TimelineOptions{})
		}},
		{"a value that is not finite", func(c *context) {
			coreBox(c).Timeline("t", TimelineOptions{}).Tween(Tween{From: 0, To: float32(math.Inf(1)), Duration: time.Second})
		}},
		{"two placements", func(c *context) {
			coreBox(c).Timeline("t", TimelineOptions{}).Tween(Tween{From: 0, To: 1, Duration: time.Second, Ease: Linear}, AtTime(0), AtTime(time.Second))
		}},
		{"a label that was not marked", func(c *context) {
			coreBox(c).Timeline("t", TimelineOptions{}).Tween(Tween{From: 0, To: 1, Duration: time.Second, Ease: Linear}, AtLabel("nowhere", 0))
		}},
		{"a Repeat below RepeatForever", func(c *context) {
			coreBox(c).Timeline("t", TimelineOptions{Repeat: -2}).Tween(Tween{From: 0, To: 1, Duration: time.Second, Ease: Linear})
		}},
		{"a negative timeline delay", func(c *context) {
			coreBox(c).Timeline("t", TimelineOptions{Delay: -time.Millisecond}).Tween(Tween{From: 0, To: 1, Duration: time.Second, Ease: Linear})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("no panic")
				}
			}()
			clockTester(tc.view, 100, 100)
		})
	}
}

func TestTimelinePlaysOnceAtOnce(t *testing.T) {
	// A tween that takes no time is at its To from the frame it is placed
	// in, and the timeline rests there.
	var x float32
	var done bool
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("instant", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 100, Ease: Linear})
		done = tl.Settled()
	})
	step(tt, now, time.Millisecond)
	closeFloat(t, x, 100)
	if !done {
		t.Fatal("a timeline of one instant tween is not done")
	}
	resting(t, tt)
}

func TestTimelineEndsATweenThatTakesNoTime(t *testing.T) {
	// An instant tween at the very end of a timeline is where it ends:
	// the playhead comes to rest on the time it is placed at.
	var x, last float32
	var done bool
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("instant-end", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		last = tl.Tween(Tween{From: 0, To: 1, Ease: Linear})
		done = tl.Settled()
	})
	closeFloat(t, last, 0) // its time has not come yet
	step(tt, now, 400*time.Millisecond)
	closeFloat(t, x, 100)
	closeFloat(t, last, 1)
	if !done {
		t.Fatal("the timeline is not done")
	}
	resting(t, tt)
}

func TestTimelinePausedTimeIsNotPlayedTime(t *testing.T) {
	var x float32
	paused := false
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("pause-idle", TimelineOptions{Paused: paused})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: time.Second, Ease: Linear})
	})
	step(tt, now, 250*time.Millisecond)
	closeFloat(t, x, 25)
	paused = true
	tt.Frame()
	closeFloat(t, x, 25)
	resting(t, tt)
	// Five seconds go by with no frames at all, as they do for a paused
	// timeline in a window that needs no drawing.
	*now = now.Add(5 * time.Second)
	paused = false
	tt.Frame() // the frame that lets the pause go plays no time
	closeFloat(t, x, 25)
	step(tt, now, 100*time.Millisecond)
	closeFloat(t, x, 35)
}

func TestTimelineStartsWhereItIsAsked(t *testing.T) {
	// A part of the duration, or a time, given in the frame the timeline is
	// first built: its duration is not known yet, so the timeline asks for
	// the frame that shows where the part lands.
	var x float32
	part := float32(0.25)
	tt, _ := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("part-at-once", TimelineOptions{Progress: &part})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	closeFloat(t, x, 25)
	resting(t, tt)

	where := 100 * time.Millisecond
	tt, _ = timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("seek-at-once", TimelineOptions{Seek: &where})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
	})
	closeFloat(t, x, 25)
	resting(t, tt)
}

func TestTimelinePlaysOnADefinitionThatGoesOn(t *testing.T) {
	var x float32
	var done bool
	span := 400 * time.Millisecond
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("grow", TimelineOptions{})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: span, Ease: Linear})
		done = tl.Settled()
	})
	step(tt, now, 400*time.Millisecond)
	closeFloat(t, x, 100)
	resting(t, tt)
	// The same Run, a longer timeline: it goes on from where the playhead
	// is instead of resting where the old one ended.
	span = 800 * time.Millisecond
	tt.Frame()
	if done {
		t.Fatal("a timeline that goes on is done")
	}
	closeFloat(t, x, 50)
	if !tt.rt.animating {
		t.Fatal("a timeline that goes on stopped asking for frames")
	}
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 75)
}

func TestTimelineArrivesAgainAfterSeekingBack(t *testing.T) {
	var x float32
	var arrived bool
	var where *time.Duration
	tt, now := timelineTester(func(c *context) {
		tl := coreBox(c).Timeline("arrive-again", TimelineOptions{Seek: where})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: 400 * time.Millisecond, Ease: Linear})
		arrived = tl.Arrived()
	})
	step(tt, now, 400*time.Millisecond)
	closeFloat(t, x, 100)
	if !arrived {
		t.Fatal("the timeline did not arrive")
	}
	// Seek back into it: it moves again, so coming to rest again is an
	// arrival of its own.
	back := 200 * time.Millisecond
	where = &back
	tt.Frame()
	if arrived {
		t.Fatal("Arrived while the timeline is seeked back")
	}
	where = nil
	tt.Frame()
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 100)
	if !arrived {
		t.Fatal("Arrived is false when the timeline comes to rest again")
	}
}

func TestTimelineAppliesOptionsInTheFrameTheyChange(t *testing.T) {
	// A control that plays the timeline over runs while the frame builds,
	// after the view built the timeline: the frame the window shows is the
	// one that plays it again, and it goes on asking for frames.
	var run uint64
	var x float32
	tt, now := timelineClickTester(func(c *context) {
		tl := coreBox(c).Timeline("restart", TimelineOptions{Run: run})
		x = tl.Tween(Tween{From: 0, To: 100, Duration: time.Second, Ease: Linear})
		if coreButton(c, "Restart").Clicked() {
			run++
		}
	})
	step(tt, now, time.Second)
	closeFloat(t, x, 100)
	resting(t, tt)
	if err := tt.Click("Restart"); err != nil {
		t.Fatal(err)
	}
	closeFloat(t, x, 0)
	step(tt, now, 200*time.Millisecond)
	closeFloat(t, x, 20)
}
