package ui

import (
	"testing"
	"time"

	"github.com/egoist/mygo/internal/scene"
)

// strokesAndFills returns the glyphs the last frame painted in stroke and
// in fill, and whether every stroke painted before every fill.
func strokesAndFills(tt *Tester, stroke, fill Color) (strokes, fills []scene.Glyph, under bool) {
	under = true
	for _, op := range tt.h.last.Ops {
		if op.Kind != scene.OpGlyphs {
			continue
		}
		for _, g := range tt.h.last.Glyphs[op.Start:op.End] {
			switch g.Color {
			case stroke.scene():
				strokes = append(strokes, g)
				under = under && len(fills) == 0
			case fill.scene():
				fills = append(fills, g)
			}
		}
	}
	return strokes, fills, under
}

func TestTextStroke(t *testing.T) {
	black, white := RGB(0, 0, 0), RGB(255, 255, 255)
	tt := coreNewTester(func(c *context) {
		coreText(c, "HH").FontSize(32).TextColor(black).TextStroke(4, white)
	}, 300, 100)
	strokes, fills, under := strokesAndFills(tt, white, black)
	if len(strokes) != 2 || len(fills) != 2 {
		t.Fatalf("%d strokes, %d fills, want 2 of each", len(strokes), len(fills))
	}
	if !under {
		t.Error("a stroke paints over a glyph's fill")
	}
	for i, s := range strokes {
		f := fills[i]
		// Two DIPs outside the glyph, a pixel of padding beyond. Where the
		// system antialiases the glyph by subpixels (ClearType), its box is
		// a pixel wider each side than the coverage the stroke follows.
		if s.X > f.X-2 || s.Y > f.Y-2 || s.X+s.W < f.X+f.W+2 || s.Y+s.H < f.Y+f.H+2 ||
			s.X < f.X-3 || s.Y < f.Y-3 || s.X+s.W > f.X+f.W+3 || s.Y+s.H > f.Y+f.H+3 {
			t.Errorf("stroke %d at %v,%v %v×%v, its glyph at %v,%v %v×%v", i, s.X, s.Y, s.W, s.H, f.X, f.Y, f.W, f.H)
		}
	}
}

func TestTextStrokeInherited(t *testing.T) {
	black, red := RGB(0, 0, 0), RGB(255, 0, 0)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).TextStroke(2, red).TextColor(black).Children(func() {
			coreText(c, "A")
			coreText(c, "B").TextStroke(0, red)
		})
	}, 300, 100)
	strokes, fills, _ := strokesAndFills(tt, red, black)
	if len(strokes) != 1 || len(fills) != 2 {
		t.Errorf("%d strokes, %d fills, want 1 and 2", len(strokes), len(fills))
	}
}

// TestInlineStroke strokes the text of inline elements with their own
// strokes, over their paragraph's.
func TestInlineStroke(t *testing.T) {
	black, blue, green := RGB(0, 0, 0), RGB(0, 0, 255), RGB(0, 255, 0)
	tt := coreNewTester(func(c *context) {
		coreRichText(c).TextColor(black).TextStroke(2, blue).Children(func() {
			coreText(c, "ab")
			coreText(c, "cd").TextStroke(2, green)
			coreText(c, "ef").TextStroke(0, blue)
		})
	}, 300, 100)
	blues, fills, under := strokesAndFills(tt, blue, black)
	greens, _, _ := strokesAndFills(tt, green, black)
	if len(blues) != 2 || len(greens) != 2 || len(fills) != 6 || !under {
		t.Errorf("%d blue strokes, %d green, %d fills, strokes under: %v; want 2, 2, 6, true", len(blues), len(greens), len(fills), under)
	}
}

// TestInlineStrokeHidden strokes none of the text of an inline element
// hidden or faded out, whether the stroke is its own or its paragraph's.
func TestInlineStrokeHidden(t *testing.T) {
	black, blue, green := RGB(0, 0, 0), RGB(0, 0, 255), RGB(0, 255, 0)
	tt := coreNewTester(func(c *context) {
		coreRichText(c).TextColor(black).TextStroke(2, blue).Children(func() {
			coreText(c, "ab")
			coreText(c, "cd").Invisible()
			coreText(c, "ef").TextStroke(2, green).Opacity(0)
		})
	}, 300, 100)
	blues, fills, _ := strokesAndFills(tt, blue, black)
	greens, _, _ := strokesAndFills(tt, green, black)
	if len(blues) != 2 || len(greens) != 0 || len(fills) != 2 {
		t.Errorf("%d blue strokes, %d green, %d fills; want 2, 0, 2", len(blues), len(greens), len(fills))
	}
}

func TestTabSize(t *testing.T) {
	var inherited, own float32
	tt := coreNewTester(func(c *context) {
		coreColumn(c).TabSize(4).Font("monospace").AlignItems(Start).Children(func() {
			coreText(c, "a b").Label("space")
			coreText(c, "ab").Label("none")
			coreText(c, "\tb").Label("four")
			coreText(c, "\tb").TabSize(2).Label("two")
			inherited = coreText(c, "x").textParams(0).Style.TabSize
			own = coreText(c, "x").TabSize(8).textParams(0).Style.TabSize
		})
	}, 400, 300)
	if inherited != 4 || own != 8 {
		t.Errorf("tab sizes %v and %v, want 4 and 8", inherited, own)
	}
	width := func(label string) float32 {
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("no %s", label)
		}
		return r.W
	}
	space := width("space") - width("none")
	letter := width("none") / 2
	if d := width("four") - (4*space + letter); d < -0.5 || d > 0.5 {
		t.Errorf("a tab of 4 spaces of %v with a letter of %v is %v wide", space, letter, width("four"))
	}
	if d := width("two") - (2*space + letter); d < -0.5 || d > 0.5 {
		t.Errorf("a tab of 2 spaces of %v with a letter of %v is %v wide", space, letter, width("two"))
	}
}

// TestMorphStroke strokes the glyphs of text morphing, under them all.
func TestMorphStroke(t *testing.T) {
	s := "ab"
	a := Animation{Duration: 100 * time.Millisecond, Ease: Linear}
	tt, now := clockTester(func(c *context) {
		keep("text", coreText(c, s)).TextStroke(2, RGB(255, 255, 255)).ContentTransition(ContentTransition{Morph: true, Animation: &a})
	}, 200, 100)
	s = "ba"
	tt.Frame()
	*now = now.Add(50 * time.Millisecond)
	tt.Frame()
	// The two move, each with its stroke.
	if n := glyphsPainted(tt); n != 4 {
		t.Errorf("halfway from ab to ba, %d glyphs paint, want 2 and their strokes", n)
	}
	strokes := tt.rt.scene.Glyphs[:2]
	for _, g := range tt.rt.scene.Glyphs[2:] {
		for _, s := range strokes {
			if g.Color == s.Color {
				t.Errorf("a glyph in its stroke's color %v", g.Color)
			}
		}
	}
	if strokes[0].Color != RGB(255, 255, 255).scene() || strokes[1].Color != strokes[0].Color {
		t.Errorf("strokes in %v and %v", strokes[0].Color, strokes[1].Color)
	}
}
