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
