package ui

import (
	"math"
	"testing"
	"time"
)

func TestSpringDurationAndBounce(t *testing.T) {
	m, k, c, _ := SpringOptions{}.physics()
	w := 2 * math.Pi / 0.5
	if m != 1 || math.Abs(k-w*w) > 1e-9 || math.Abs(c-2*w) > 1e-9 {
		t.Fatalf("the default spring: %v %v %v", m, k, c)
	}
	_, k, c, _ = SpringOptions{Duration: 250 * time.Millisecond, Bounce: 0.3}.physics()
	w = 2 * math.Pi / 0.25
	if math.Abs(k-w*w) > 1e-9 || math.Abs(c-2*0.7*w) > 1e-9 {
		t.Fatalf("a bouncy spring: %v %v", k, c)
	}
	_, _, c, _ = SpringOptions{Bounce: -0.5}.physics()
	if w := 2 * math.Pi / 0.5; math.Abs(c-2*2*w) > 1e-9 {
		t.Fatalf("an overdamped spring: %v", c)
	}
	m, k, c, _ = SpringOptions{Stiffness: 300}.physics()
	if m != 1 || k != 300 || c != 26 {
		t.Fatalf("a physical spring: %v %v %v", m, k, c)
	}
	for _, o := range []SpringOptions{
		{Duration: time.Second, Stiffness: 100},
		{Bounce: 0.2, Damping: 10},
		{Bounce: 1},
		{Bounce: -1},
		{Duration: -time.Second},
		{Precision: math.NaN()},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%+v accepted", o)
				}
			}()
			o.physics()
		}()
	}
	// A bouncy spring overshoots its target; a smooth one does not.
	for _, a := range []struct {
		Animation
		over bool
	}{{Smooth, false}, {Bouncy, true}} {
		ch := channel{at: time.Unix(0, 0)}
		ch.retarget(100, a.Animation)
		most := 0.0
		for i := 1; i <= 120; i++ {
			ch.advance(time.Unix(0, 0).Add(time.Duration(i) * time.Second / 60))
			most = max(most, ch.value)
		}
		if (most > 100.01) != a.over || ch.moving {
			t.Errorf("a spring of %v went as far as %v, moving %v", a.Spring.Bounce, most, ch.moving)
		}
	}
}

func TestSpringTakesAVelocity(t *testing.T) {
	target := float32(0)
	o := SpringOptions{}
	var value float32
	var e *node
	tt, now := motionTester(func(c *context) { e = coreBox(c); value = e.Spring("x", target, o) })
	target, o.Velocity = 100, -2000
	tt.Frame()
	if v := e.Velocity("x"); v != -2000 {
		t.Fatalf("the spring starts at %v/s", v)
	}
	*now = now.Add(30 * time.Millisecond)
	tt.Frame()
	if value >= 0 {
		t.Fatalf("thrown away from its target, the value is %v", value)
	}
	// The velocity counts only as the target changes.
	*now = now.Add(time.Second)
	tt.Frame()
	if value <= 50 {
		t.Fatalf("the spring did not head back: %v", value)
	}
}

func TestAnimationDelaySettledAndArrived(t *testing.T) {
	target := float32(0)
	var value float32
	var settled, arrived bool
	arrivals := 0
	tt, now := motionTester(func(c *context) {
		e := coreBox(c)
		value = e.AnimateTo("x", target, Animation{Duration: 100 * time.Millisecond, Ease: Linear, Delay: 50 * time.Millisecond})
		settled, arrived = e.Settled("x"), e.Arrived("x")
		if arrived {
			arrivals++
		}
	})
	if !settled || arrived {
		t.Fatal("a value at its first target is not settled")
	}
	target = 10
	tt.Frame()
	if value != 0 || settled {
		t.Fatalf("delayed, the value is %v, settled %v", value, settled)
	}
	*now = now.Add(40 * time.Millisecond)
	tt.Frame()
	closeFloat(t, value, 0)
	*now = now.Add(60 * time.Millisecond)
	tt.Frame()
	closeFloat(t, value, 5)
	*now = now.Add(60 * time.Millisecond)
	tt.Frame()
	if value != 10 || !settled || !arrived || arrivals != 1 {
		t.Fatalf("over, the value is %v, settled %v, arrived %v (%d)", value, settled, arrived, arrivals)
	}
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	if arrived || arrivals != 1 {
		t.Fatal("arrived again in a later frame")
	}
	if tt.rt.animating {
		t.Fatal("a settled value asks for frames")
	}
}

func TestSpringTakesOverACurve(t *testing.T) {
	target := float32(0)
	spring := false
	var value float32
	var e *node
	tt, now := motionTester(func(c *context) {
		e = coreBox(c)
		a := Animation{Duration: time.Second, Ease: Linear}
		if spring {
			a = Snappy
		}
		value = e.AnimateTo("x", target, a)
	})
	target = 100
	tt.Frame()
	*now = now.Add(500 * time.Millisecond)
	tt.Frame()
	closeFloat(t, value, 50)
	closeFloat(t, e.Velocity("x"), 100)
	spring, target = true, 0
	tt.Frame()
	closeFloat(t, value, 50)
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	if value <= 50 {
		t.Fatalf("the spring dropped the curve's velocity: %v", value)
	}
}

func TestJumpHandsItsVelocityOn(t *testing.T) {
	dragging := true
	x := float32(0)
	var value float32
	var e *node
	tt, now := motionTester(func(c *context) {
		e = coreBox(c)
		if dragging {
			value = e.Jump("x", x)
		} else {
			value = e.AnimateTo("x", 0, Smooth)
		}
	})
	for range 5 {
		*now = now.Add(10 * time.Millisecond)
		x += 10
		tt.Frame()
	}
	if value != 50 || e.Velocity("x") != 1000 {
		t.Fatalf("dragged, the value is %v at %v/s", value, e.Velocity("x"))
	}
	dragging = false
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	if value <= 50 {
		t.Fatalf("let go, the value stopped dead: %v", value)
	}
	*now = now.Add(5 * time.Second)
	tt.Frame()
	if value != 0 || !e.Settled("x") {
		t.Fatalf("the value did not come back: %v", value)
	}
}

func TestInterruptedKeyframesRepeatTheirRange(t *testing.T) {
	frames := []Keyframe{{Value: 0}, {At: 100 * time.Millisecond, Value: 100}}
	s := sequence{frames: frames, options: KeyframeOptions{Repeat: RepeatForever, Alternate: true}, start: time.Unix(0, 0), from: 50}
	at := func(d time.Duration) float32 { v, _ := s.sample(s.start.Add(d)); return v }
	closeFloat(t, at(0), 50)
	closeFloat(t, at(50*time.Millisecond), 75)
	// Back to the first frame's value, not to where it was interrupted.
	closeFloat(t, at(199*time.Millisecond), 1)
	closeFloat(t, at(250*time.Millisecond), 50)
	s = sequence{frames: frames, options: KeyframeOptions{Repeat: 1}, start: time.Unix(0, 0), from: 50}
	closeFloat(t, at(150*time.Millisecond), 50)
	if _, done := s.sample(s.start.Add(199 * time.Millisecond)); done {
		t.Fatal("Repeat 1 played once")
	}
	if v, done := s.sample(s.start.Add(200 * time.Millisecond)); !done || v != 100 {
		t.Fatalf("Repeat 1 played on: %v, %v", v, done)
	}
	s = sequence{frames: frames, start: time.Unix(0, 0)}
	if _, done := s.sample(s.start.Add(100 * time.Millisecond)); !done {
		t.Fatal("no Repeat played twice")
	}

	var value float32
	var arrived bool
	tt, now := motionTester(func(c *context) {
		e := coreBox(c)
		value = e.Keyframes("k", frames, KeyframeOptions{Delay: 50 * time.Millisecond})
		arrived = e.Arrived("k")
	})
	closeFloat(t, value, 0)
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	closeFloat(t, value, 50)
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if value != 100 || !arrived {
		t.Fatalf("over, the sequence is at %v, arrived %v", value, arrived)
	}
}

func TestCubicBezier(t *testing.T) {
	ease := CubicBezier(0.25, 0.1, 0.25, 1) // CSS's ease
	for _, c := range []struct{ x, y float32 }{{0, 0}, {1, 1}, {0.25, 0.4085}, {0.5, 0.8024}, {0.75, 0.9604}} {
		if y := ease(c.x); math.Abs(float64(y-c.y)) > 1e-3 {
			t.Errorf("ease(%v) = %v, not %v", c.x, y, c.y)
		}
	}
	linear := CubicBezier(0, 0, 1, 1)
	for x := float32(0); x <= 1; x += 0.1 {
		if y := linear(x); math.Abs(float64(y-x)) > 1e-4 {
			t.Errorf("linear(%v) = %v", x, y)
		}
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("x2 of 2 accepted")
			}
		}()
		CubicBezier(0, 0, 2, 1)
	}()
}

func TestDragVelocity(t *testing.T) {
	var vx, vy, ex, ey float32
	var ended bool
	tt, now := motionTester(func(c *context) {
		e := coreBox(c).Size(200, 100)
		vx, vy = e.DragVelocity()
		if x, y, ok := e.DragEnded(); ok {
			ex, ey, ended = x, y, true
		}
	})
	tt.Move(10, 10)
	tt.Press(10, 10)
	for i := 1; i <= 5; i++ {
		*now = now.Add(10 * time.Millisecond)
		tt.Move(10+float32(i)*5, 10+float32(i)*2)
	}
	tt.Frame()
	closeFloat(t, vx, 500)
	closeFloat(t, vy, 200)
	*now = now.Add(5 * time.Millisecond)
	tt.Release(35, 20)
	if !ended {
		t.Fatal("the drag did not end")
	}
	closeFloat(t, ex, 500)
	closeFloat(t, ey, 200)
	if vx != 0 || vy != 0 {
		t.Fatal("let go, the drag still has a velocity")
	}
	ended = false
	tt.Frame()
	if ended {
		t.Fatal("the drag ended twice")
	}
	// A pointer resting before it lets go flings nothing.
	tt.Move(10, 10)
	tt.Press(10, 10)
	*now = now.Add(10 * time.Millisecond)
	tt.Move(30, 10)
	*now = now.Add(200 * time.Millisecond)
	tt.Release(30, 10)
	if !ended || ex != 0 || ey != 0 {
		t.Fatalf("resting, it let go at %v, %v", ex, ey)
	}
	// A click ends no drag.
	ended = false
	tt.Press(30, 10)
	tt.Release(30, 10)
	if ended {
		t.Fatal("a click ended a drag")
	}
}

func TestInView(t *testing.T) {
	var top, bottom *node
	var topIn, bottomIn bool
	tt := coreNewTester(func(c *context) {
		coreScroll(c).Size(100, 100).Children(func() {
			top = coreBox(c).Key("top").Size(100, 60)
			coreBox(c).Size(100, 200)
			bottom = coreBox(c).Key("bottom").Size(100, 60)
			topIn, bottomIn = top.InView(), bottom.InView()
		})
	}, 200, 200)
	tt.Frame()
	if !topIn || bottomIn {
		t.Fatalf("at the top, the top is in view %v, the bottom %v", topIn, bottomIn)
	}
	tt.Scroll(50, 50, 0, 1000)
	tt.Frame()
	if topIn || !bottomIn {
		t.Fatalf("at the bottom, the top is in view %v, the bottom %v", topIn, bottomIn)
	}
}

func TestDisableAnimations(t *testing.T) {
	target, instant := float32(0), false
	show := true
	var value float32
	tt, now := clockTester(func(c *context) {
		if instant {
			c.DisableAnimations()
		}
		e := coreBox(c)
		value = e.AnimateTo("x", target, Smooth)
		coreBox(c).Size(100, 100).Children(func() {
			coreBox(c).Key("moving").Size(10, 10).Absolute().Left(target).Transition(ElementTransition{})
			if show {
				coreBox(c).Key("going").Size(10, 10).Transition(ElementTransition{Exit: &Motion{}})
			}
		})
	}, 200, 200)
	target, instant, show = 50, true, false
	tt.Frame()
	if value != 50 || tt.rt.animating {
		t.Fatalf("without animations, the value is %v", value)
	}
	for g := tt.rt.c.root.first; g != nil; g = g.next {
		for e := g.first; e != nil; e = e.next {
			if e.leaving != 0 {
				t.Error("without animations, a copy leaves")
			} else if e.x != 50 {
				t.Errorf("without animations, the box is at %v", e.x)
			}
		}
	}
	instant, target = false, 0
	*now = now.Add(10 * time.Millisecond)
	tt.Frame()
	if value == 0 {
		t.Error("with animations again, the value jumped")
	}
}

func TestAnimateColor(t *testing.T) {
	target := RGBA(255, 0, 0, 0)
	var shown Color
	var settled bool
	tt, now := motionTester(func(c *context) {
		e := coreBox(c)
		shown, settled = e.AnimateColor("bg", target, Animation{Duration: 100 * time.Millisecond, Ease: Linear}), e.Settled("bg")
	})
	target = RGB(0, 0, 255)
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	// From transparent red, the blue does not go through red.
	if shown.R != 0 || shown.B != 255 || shown.A < 120 || shown.A > 135 || settled {
		t.Errorf("halfway, the color is %+v", shown)
	}
	tt.SetPreferences(Preferences{ReduceMotion: true})
	target = RGB(0, 255, 0)
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if shown.G == 255 || shown.G == 0 {
		t.Errorf("without motion, colors still change smoothly: %+v", shown)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if shown != target || !settled {
		t.Errorf("over, the color is %+v", shown)
	}
}

func TestRubberbandAndProject(t *testing.T) {
	if v := Rubberband(50, 0, 100, 200); v != 50 {
		t.Errorf("within, %v", v)
	}
	over, under := Rubberband(150, 0, 100, 200), Rubberband(-50, 0, 100, 200)
	if over <= 100 || over >= 150 || over-100 != -under {
		t.Errorf("past either end, %v and %v", over, under)
	}
	if far := Rubberband(1e6, 0, 100, 200); far >= 300 {
		t.Errorf("however far it is dragged, it stays within its dimension: %v", far)
	}
	if v := Project(0, 1000); v < 498 || v > 500 {
		t.Errorf("a fling of 1000/s goes %v", v)
	}
}

func TestKeyframeRepeatDelay(t *testing.T) {
	frames := []Keyframe{{Value: 0}, {At: 100 * time.Millisecond, Value: 100}}
	s := sequence{frames: frames, options: KeyframeOptions{Repeat: 1, RepeatDelay: 50 * time.Millisecond}, start: time.Unix(0, 0)}
	at := func(d time.Duration) (float32, bool) { return s.sample(s.start.Add(d)) }
	if v, _ := at(120 * time.Millisecond); v != 100 {
		t.Errorf("between plays, it holds %v", v)
	}
	if v, _ := at(200 * time.Millisecond); v != 50 {
		t.Errorf("playing again, %v", v)
	}
	if v, done := at(250 * time.Millisecond); v != 100 || !done {
		t.Errorf("over, %v %v", v, done)
	}
}

func TestPathTrim(t *testing.T) {
	var f flatPath
	line := (&Path{}).MoveTo(0, 0).LineTo(100, 0)
	line.Trim(0.25, 0.75).flatten(&f, 2)
	if f.polys() != 1 || len(f.pts) != 2 || f.pts[0] != [2]float32{50, 0} || f.pts[1] != [2]float32{150, 0} {
		t.Fatalf("a line trimmed to its middle: %v", f.pts)
	}
	square := (&Path{}).MoveTo(0, 0).LineTo(10, 0).LineTo(10, 10).LineTo(0, 10).Close()
	square.Trim(0, 0.625).flatten(&f, 1)
	want := [][2]float32{{0, 0}, {10, 0}, {10, 10}, {5, 10}}
	if f.polys() != 1 || f.closed[0] || len(f.pts) != len(want) {
		t.Fatalf("a square trimmed: %v", f.pts)
	}
	for i := range want {
		if f.pts[i] != want[i] {
			t.Fatalf("a square trimmed: %v", f.pts)
		}
	}
	// Into the closing side, and across subpaths.
	two := (&Path{}).MoveTo(0, 0).LineTo(10, 0).MoveTo(0, 5).LineTo(10, 5)
	two.Trim(0.25, 0.75).flatten(&f, 1)
	if f.polys() != 2 || f.poly(0)[0] != [2]float32{5, 0} || f.poly(1)[1] != [2]float32{5, 5} {
		t.Fatalf("two lines trimmed: %v %v", f.pts, f.starts)
	}
	square.Trim(0.9, 1).flatten(&f, 1)
	if f.polys() != 1 || math.Abs(float64(f.pts[0][1]-4)) > 1e-4 || f.pts[0][0] != 0 || f.pts[1] != [2]float32{0, 0} {
		t.Fatalf("the closing side trimmed: %v", f.pts)
	}
	square.Trim(0.5, 0.5).flatten(&f, 1)
	if f.polys() != 0 {
		t.Fatalf("nothing kept, %v", f.pts)
	}
	// Painted, a trimmed stroke shows only its part.
	tt := coreNewTester(func(c *context) {
		coreBox(c).Size(100, 20).Draw(func(p *Painter, r Rect) {
			p.StrokePath((&Path{}).MoveTo(r.X, r.Y+10).LineTo(r.X+100, r.Y+10).Trim(0, 0.5), 4, RGB(255, 0, 0))
		})
	}, 100, 20)
	if p := tt.Image().RGBAAt(25, 10); p.R < 200 || p.G > 50 {
		t.Errorf("the half kept is %v", p)
	}
	if p := tt.Image().RGBAAt(75, 10); p.R > 200 && p.G < 50 {
		t.Errorf("the half trimmed away is %v", p)
	}
	// Trimmed to almost nothing, the stroke's round caps fade rather than
	// showing a dot that goes at once.
	kept := float32(0.004) // 0.4 of a 4 wide stroke
	tt = coreNewTester(func(c *context) {
		coreBox(c).Size(100, 20).Draw(func(p *Painter, r Rect) {
			p.StrokePath((&Path{}).MoveTo(r.X+10, r.Y+10).LineTo(r.X+110, r.Y+10).Trim(0, kept), 4, RGB(255, 0, 0))
		})
	}, 120, 20)
	if p := tt.Image().RGBAAt(10, 10); p.G < 120 {
		t.Errorf("trimmed to almost nothing, the stroke shows as %v", p)
	}
	kept = 0.04
	tt.Frame()
	if p := tt.Image().RGBAAt(10, 10); p.G > 30 {
		t.Errorf("as long as it is wide, the stroke shows as %v", p)
	}
}

func TestScrollToGlides(t *testing.T) {
	var feed ScrollState
	tt, now := clockTester(func(c *context) {
		coreScroll(c).Size(100, 100).TrackScroll(&feed).Children(func() {
			coreBox(c).Size(100, 1000)
		})
	}, 200, 200)
	feed.ScrollTo(0, 500, Animation{Duration: 100 * time.Millisecond, Ease: Linear})
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if feed.Y < 240 || feed.Y > 260 || !feed.Scrolling() {
		t.Fatalf("halfway, scrolled to %v", feed.Y)
	}
	*now = now.Add(60 * time.Millisecond)
	tt.Frame()
	if feed.Y != 500 || feed.Scrolling() || tt.rt.animating {
		t.Fatalf("there, scrolled to %v", feed.Y)
	}
	// To the end, and the user scrolling stops it.
	feed.ScrollTo(0, math.MaxFloat32, Smooth)
	tt.Frame()
	*now = now.Add(30 * time.Millisecond)
	tt.Frame()
	tt.Scroll(50, 50, 0, -10)
	if feed.Scrolling() {
		t.Error("the user scrolling did not stop it")
	}
	y := feed.Y
	*now = now.Add(time.Second)
	tt.Frame()
	if feed.Y != y {
		t.Errorf("stopped, it went on from %v to %v", y, feed.Y)
	}
}
