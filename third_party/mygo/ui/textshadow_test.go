package ui

import "testing"

// TestTextShadow casts the shadows of two glyphs under them and their
// strokes, offset and blurred as asked.
func TestTextShadow(t *testing.T) {
	black, red, white := RGB(0, 0, 0), RGB(255, 0, 0), RGB(255, 255, 255)
	tt := coreNewTester(func(c *context) {
		coreText(c, "HH").FontSize(32).TextColor(black).TextStroke(1, white).TextShadow(5, 7, 4, red)
	}, 300, 100)
	shadows, fills, under := strokesAndFills(tt, red, black)
	strokes, _, strokesUnder := strokesAndFills(tt, red, white)
	if len(shadows) != 2 || len(fills) != 2 || !under {
		t.Fatalf("%d shadows, %d fills, shadows under: %v; want 2, 2, true", len(shadows), len(fills), under)
	}
	if len(strokes) != 2 || !strokesUnder {
		t.Errorf("shadows under strokes: %v", strokesUnder)
	}
	for i, s := range shadows {
		f := fills[i]
		dx, dy := s.X+s.W/2-(f.X+f.W/2), s.Y+s.H/2-(f.Y+f.H/2)
		// The blur widens it by three sigmas, of half the blur, each side.
		if dx < 4 || dx > 6 || dy < 6 || dy > 8 || s.W < f.W+10 || s.H < f.H+10 {
			t.Errorf("shadow %d at %v,%v %v×%v, its glyph at %v,%v %v×%v", i, s.X, s.Y, s.W, s.H, f.X, f.Y, f.W, f.H)
		}
	}
}

// TestTextShadowSpans casts the shadows of a paragraph's text, inherited
// but where an inline element casts none, and of a span's own.
func TestTextShadowSpans(t *testing.T) {
	black, red, blue := RGB(0, 0, 0), RGB(255, 0, 0), RGB(0, 0, 255)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).TextShadow(1, 1, 0, red).Children(func() {
			coreRichText(c).TextColor(black).Children(func() {
				coreText(c, "ab")
				coreText(c, "cd").TextShadow(0, 0, 0, Color{})
			})
			coreRichText(c, Span{Text: "ef"}, Span{Text: "g", Shadow: NewTextShadow(0, 2, 0, blue)}).TextColor(black)
		})
	}, 300, 100)
	reds, fills, _ := strokesAndFills(tt, red, black)
	blues, _, _ := strokesAndFills(tt, blue, black)
	if len(reds) != 4 || len(blues) != 1 || len(fills) != 7 {
		t.Errorf("%d red shadows, %d blue, %d fills; want 4, 1, 7", len(reds), len(blues), len(fills))
	}
}

// TestTextShadowRenders paints a shadow offset beside its white glyph,
// red, and blurred over more pixels.
func TestTextShadowRenders(t *testing.T) {
	view := func(blur float32) func(c *context) {
		return func(c *context) {
			coreBox(c).Fill().Background(RGB(255, 255, 255)).Children(func() {
				coreText(c, "H").FontSize(40).TextColor(RGB(255, 255, 255)).TextShadow(40, 0, blur, RGB(255, 0, 0))
			})
		}
	}
	sharp, soft := coreRender(view(0), 140, 60, 1), coreRender(view(12), 140, 60, 1)
	// The pixels the shadow reddens along the middle row.
	var sharpEdge, softEdge int
	for x := 0; x < 140; x++ {
		if c := sharp.RGBAAt(x, 30); c.G < 128 {
			sharpEdge++
		}
		if c := soft.RGBAAt(x, 30); c.G < 250 {
			softEdge++
		}
	}
	if sharpEdge == 0 || softEdge <= sharpEdge {
		t.Errorf("red pixels across the middle: %d sharp, %d blurred", sharpEdge, softEdge)
	}
}
