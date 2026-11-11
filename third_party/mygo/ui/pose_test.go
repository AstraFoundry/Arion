package ui

import (
	"testing"
	"time"
)

func TestPoseMovesInFramesThatOnlyPaint(t *testing.T) {
	lifted := false
	var card *node
	clicks := 0
	tt := newRepaintTester(func(c *context) {
		p := Pose{}
		if lifted {
			p = Pose{X: 100, Fade: 0.5}
		}
		card = coreBox(c).Size(40, 40).Background(RGB(255, 0, 0)).AnimatePose(p, Animation{Duration: 100 * time.Millisecond, Ease: Linear})
		if card.Clicked() {
			clicks++
		}
	})
	lifted = true
	tt.rt.requestFrame()
	if built, more := tt.frame(0); !built || !more {
		t.Fatalf("the change: built %v, more %v", built, more)
	}
	builds := tt.builds
	built, more := tt.frame(50 * time.Millisecond)
	if built || !more || tt.builds != builds {
		t.Fatalf("moving: built %v, more %v", built, more)
	}
	if w := card.st.world; !w.Set || w.X < 49 || w.X > 51 {
		t.Fatalf("halfway, the card shows at %+v", w)
	}
	// The pointer finds it where it shows.
	tt.ClickAt(70, 20)
	if clicks != 1 {
		t.Errorf("a click where it shows: %d clicks", clicks)
	}
	// A quarter faded, red over white.
	if p := tt.Image().RGBAAt(70, 20); p.R != 255 || p.G < 54 || p.G > 74 {
		t.Errorf("halfway faded, the card is %v", p)
	}
	tt.frame(100 * time.Millisecond)
	if _, more := tt.frame(10 * time.Millisecond); more {
		t.Error("in place, it asks for frames")
	}
	if w := card.st.world; w.X != 100 {
		t.Errorf("in place, the card shows at %+v", w)
	}
}

func TestPoseWithoutMotion(t *testing.T) {
	lifted := false
	var card *node
	tt, now := clockTester(func(c *context) {
		p := Pose{}
		if lifted {
			p = Pose{Y: -20, Scale: 2, Fade: 1}
		}
		card = coreBox(c).Size(40, 40).AnimatePose(p, Animation{Duration: 100 * time.Millisecond, Ease: Linear})
	}, 200, 200)
	tt.SetPreferences(Preferences{ReduceMotion: true})
	lifted = true
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if w := card.st.world; w.A != 2 || card.poseFade < 0.4 || card.poseFade > 0.6 || !card.Transitioning() {
		t.Errorf("without motion, the card shows as %+v, faded %v", w, card.poseFade)
	}
}

// Poses nest, in frames that only paint as in those that build: what is
// inside moving poses follows them, whether its own pose moves or rests.
func TestNestedPosesInFramesThatOnlyPaint(t *testing.T) {
	moved := false
	var leaf, still *node
	tt := newRepaintTester(func(c *context) {
		outer, inner := Pose{}, Pose{}
		if moved {
			outer, inner = Pose{X: 100}, Pose{Y: 60}
		}
		a := Animation{Duration: 100 * time.Millisecond, Ease: Linear}
		coreBox(c).Size(40, 40).AnimatePose(outer, a).Children(func() {
			// A pose at rest inside a moving one, and a moving one inside it.
			coreBox(c).Size(40, 40).AnimatePose(Pose{X: 10}, a).Children(func() {
				coreBox(c).Size(40, 40).AnimatePose(inner, a).Children(func() {
					leaf = coreBox(c).Size(10, 10)
				})
				still = coreBox(c).Size(10, 10)
			})
		})
	})
	moved = true
	tt.rt.requestFrame()
	tt.frame(0)
	builds := tt.builds
	tt.frame(50 * time.Millisecond)
	if tt.builds != builds {
		t.Fatal("moving poses built the view")
	}
	if w := leaf.st.world; w.X < 59 || w.X > 61 || w.Y < 29 || w.Y > 31 {
		t.Errorf("halfway, the innermost element shows at %+v, want 60, 30", w)
	}
	if w := still.st.world; w.X < 59 || w.X > 61 || w.Y != 0 {
		t.Errorf("halfway, the element under the resting pose shows at %+v, want 60, 0", w)
	}
}

// An element posed for the first time shows at its pose at once, and
// stays there until a build poses it elsewhere; posed elsewhere in the
// next build, which the first asks for, it moves there, as AnimatePose's
// documentation says an entrance goes.
func TestPoseEntersAfterASecondBuild(t *testing.T) {
	for _, second := range []bool{false, true} {
		var card *node
		tt := coreNewTester(func(c *context) {
			card = coreBox(c).Size(40, 40)
			entered := coreLocal(card, "entered", func() bool { return false })
			p := Pose{}
			if !*entered {
				p, *entered = Pose{Y: 8, Fade: 1}, true
				if second {
					c.Invalidate()
				}
			}
			card.AnimatePose(p, Animation{Duration: 100 * time.Millisecond, Ease: Linear})
		}, 200, 200)
		tt.Advance(50 * time.Millisecond)
		y, fade := card.st.world.Y, card.poseFade
		if !second {
			if y != 8 || fade != 1 {
				t.Errorf("without a second build, the card shows at %v, faded %v: want 8, 1", y, fade)
			}
			continue
		}
		if y < 3.9 || y > 4.1 || fade < 0.49 || fade > 0.51 {
			t.Errorf("halfway in, the card shows at %v, faded %v: want 4, 0.5", y, fade)
		}
		tt.Advance(100 * time.Millisecond)
		if y, fade := card.st.world.Y, card.poseFade; y != 0 || fade != 0 {
			t.Errorf("in, the card shows at %v, faded %v", y, fade)
		}
	}
}

// A pose by percent of the element's size follows the size at once, in
// the frame it changes, and moves along its curve when the pose changes.
func TestPosePercent(t *testing.T) {
	w, pct := float32(40), float32(50)
	var card *node
	tt, now := clockTester(func(c *context) {
		card = coreBox(c).Size(w, 20).AnimatePose(Pose{X: 5, XPercent: pct, YPercent: -100}, Animation{Duration: 100 * time.Millisecond, Ease: Linear})
	}, 200, 200)
	if m := card.st.world; m.X != 25 || m.Y != -20 {
		t.Fatalf("posed by percent, the card shows at %v, %v", m.X, m.Y)
	}
	w = 80
	tt.Frame()
	if m := card.st.world; m.X != 45 || card.Transitioning() {
		t.Errorf("resized, the card shows at %v", m.X)
	}
	pct = 0
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if m := card.st.world; m.X < 24.9 || m.X > 25.1 {
		t.Errorf("halfway, the card shows at %v", m.X)
	}
}
