package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
)

// paintedGlyphs returns how many glyphs the last frame painted, and how
// many of them were partly transparent.
func paintedGlyphs(tt *Tester) (all, faded int) {
	for _, op := range tt.h.last.Ops {
		if op.Kind != scene.OpGlyphs {
			continue
		}
		for _, g := range tt.h.last.Glyphs[op.Start:op.End] {
			all++
			if a := g.Color.A; a > 20 && a < 235 || op.Opacity > 0.05 && op.Opacity < 0.95 {
				faded++
			}
		}
	}
	return all, faded
}

func clips(tt *Tester) int {
	n := 0
	for _, op := range tt.h.last.Ops {
		if op.Kind == scene.OpPushClip {
			n++
		}
	}
	return n
}

func TestNumericContentTransition(t *testing.T) {
	value := "120"
	numeric := true
	tt, now := clockTester(func(c *context) {
		coreText(c, value).FontSize(30).ContentTransition(ContentTransition{Numeric: numeric, Animation: &Animation{Duration: 100 * time.Millisecond, Ease: Linear}})
	}, 200, 60)
	base := clips(tt)
	if all, faded := paintedGlyphs(tt); all != 3 || faded != 0 {
		t.Fatalf("at rest, %d glyphs, %d faded", all, faded)
	}
	value = "130"
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	// The 1 and the 0 stay; the 2 goes as the 3 comes.
	if n := clips(tt) - base; n != 4 {
		t.Errorf("rolling, %d bands", n)
	}
	all, faded := paintedGlyphs(tt)
	if faded < 2 {
		t.Errorf("rolling, %d of %d glyphs faded", faded, all)
	}
	if !tt.rt.animating && !tt.h.requested.Load() {
		t.Error("rolling, it asks for no frames")
	}
	*now = now.Add(100 * time.Millisecond)
	tt.Frame()
	if all, faded := paintedGlyphs(tt); clips(tt) != base || faded != 0 {
		t.Errorf("over, %d glyphs, %d faded, %d bands", all, faded, clips(tt)-base)
	}
	// Without Numeric, the text fades across.
	numeric = false
	value = "hello"
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if all, faded := paintedGlyphs(tt); clips(tt) != base || faded != all || all != 8 {
		t.Errorf("fading, %d glyphs, %d faded, %d bands", all, faded, clips(tt)-base)
	}
	// Not painted in the frame before, a change shows at once.
	tt.SetPreferences(Preferences{ReduceMotion: true})
	numeric = true
	value = "131"
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if n := clips(tt) - base; n != 0 {
		t.Errorf("without motion, %d bands", n)
	}
}

func TestIconContentTransition(t *testing.T) {
	a := MustParseSVG([]byte(`<svg viewBox="0 0 10 10"><rect width="10" height="10"/></svg>`))
	b := MustParseSVG([]byte(`<svg viewBox="0 0 10 10"><circle cx="5" cy="5" r="5"/></svg>`))
	icon := a
	tt, now := clockTester(func(c *context) {
		coreIcon(c, icon).FontSize(40).ContentTransition(ContentTransition{Animation: &Animation{Duration: 100 * time.Millisecond, Ease: Linear}})
	}, 100, 100)
	images := func() (n, scaled int) {
		for _, op := range tt.h.last.Ops {
			if op.Kind == scene.OpGlyphs {
				n++
				if op.Transform.Set {
					scaled++
				}
			}
		}
		return
	}
	n0, _ := images()
	if n0 == 0 {
		t.Fatal("the icon paints no picture")
	}
	icon = b
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	if n, scaled := images(); n != 2*n0 || scaled != n {
		t.Errorf("replacing, %d pictures (%d at rest), %d scaled", n, n0, scaled)
	}
	*now = now.Add(time.Second)
	tt.Frame()
	if n, scaled := images(); n != n0 || scaled != 0 {
		t.Errorf("replaced, %d pictures, %d scaled", n, scaled)
	}
}

func TestContentTransitionOnlyPaints(t *testing.T) {
	value := "1"
	tt := newRepaintTester(func(c *context) {
		coreText(c, value).ContentTransition(ContentTransition{Numeric: true, Animation: &Animation{Duration: 100 * time.Millisecond, Ease: Linear}})
	})
	value = "2"
	tt.rt.requestFrame()
	if built, more := tt.frame(0); !built || !more {
		t.Fatalf("the change: built %v, more %v", built, more)
	}
	before := tt.h.last.Glyphs[0].Color.A
	built, more := tt.frame(50 * time.Millisecond)
	if built || !more {
		t.Fatalf("rolling: built %v, more %v", built, more)
	}
	if after := tt.h.last.Glyphs[0].Color.A; after == before {
		t.Error("repainted, the text did not move on")
	}
	if _, more := tt.frame(100 * time.Millisecond); more {
		t.Error("over, it asks for more frames")
	}
}
