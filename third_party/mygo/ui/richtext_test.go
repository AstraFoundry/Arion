package ui

import (
	"image"
	"math"
	"testing"
)

func TestRichText(t *testing.T) {
	red, blue := RGB(255, 0, 0), RGB(0, 0, 255)
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).Gap(10).AlignItems(Start).Children(func() {
			coreRichText(c, Span{Text: "plain "}, Span{Text: "RED", Color: red, Size: 30, Weight: 700}).Label("rich")
			coreRichText(c, Span{Text: "under", Underline: true, Color: blue}, Span{Text: " over"}).Label("lined")
			coreText(c, "plain RED").Label("text")
		})
	}, 400, 200)
	rich, _ := tt.Find("rich")
	lined, _ := tt.Find("lined")
	plain, _ := tt.Find("text")
	if rich.H < plain.H*1.5 {
		t.Errorf("a span of size 30 leaves the text %v high, plain text %v", rich.H, plain.H)
	}
	img := tt.Image()
	count := func(r Rect, match func(r, g, b uint8) bool) (n, widest int) {
		box := image.Rect(int(r.X), int(r.Y), int(r.X+r.W), int(r.Y+r.H))
		for y := box.Min.Y; y < box.Max.Y; y++ {
			row := 0
			for x := box.Min.X; x < box.Max.X; x++ {
				if c := img.RGBAAt(x, y); match(c.R, c.G, c.B) {
					n++
					row++
				}
			}
			widest = max(widest, row)
		}
		return n, widest
	}
	isRed := func(r, g, b uint8) bool { return r > 200 && g < 80 && b < 80 }
	isBlue := func(r, g, b uint8) bool { return b > 200 && r < 80 && g < 80 }
	if n, _ := count(rich, isRed); n < 30 {
		t.Errorf("%d red pixels in the red span", n)
	}
	if n, _ := count(plain, isRed); n != 0 {
		t.Errorf("%d red pixels in plain text", n)
	}
	// The underline is a row of blue under the first word alone.
	_, widest := count(lined, isBlue)
	if float32(widest) < lined.W*0.4 || float32(widest) > lined.W*0.8 {
		t.Errorf("the widest row of blue is %d pixels of %v", widest, lined.W)
	}
}

// TestMeasureTextAgain checks that measuring spans measured before
// allocates nothing, and that other spans measure anew.
func TestMeasureTextAgain(t *testing.T) {
	tt := coreNewTester(func(c *context) {}, 100, 100)
	c := &tt.rt.c
	short, _ := c.MeasureText(0, Span{Text: "000"})
	long, _ := c.MeasureText(0, Span{Text: "000000"})
	bold, _ := c.MeasureText(0, Span{Text: "000", Weight: 800, Size: 30})
	if again, _ := c.MeasureText(0, Span{Text: "000"}); again != short || long <= short || bold <= short {
		t.Errorf("widths %v, %v, %v, then %v", short, long, bold, again)
	}
	if n := testing.AllocsPerRun(20, func() { c.MeasureText(0, Span{Text: "000"}, Span{Text: "1", Weight: 700}) }); n != 0 {
		t.Errorf("measuring the same spans allocates %v times", n)
	}
}

// TestSpanNoLetterSpacing checks that a span, or an inline element, can
// take no letter spacing within tracked text, as CSS's letter-spacing: 0
// on an inline box: the number of a tracked label.
func TestSpanNoLetterSpacing(t *testing.T) {
	var para *node
	b := boxes(t, func(c *context) {
		coreColumn(c).AlignItems(Start).Children(func() {
			coreRichText(c, Span{Text: "LABEL "}, Span{Text: "4200"}).Label("plain")
			coreRichText(c, Span{Text: "LABEL "}, Span{Text: "4200"}).LetterSpacing(3).Label("tracked")
			coreRichText(c, Span{Text: "LABEL "}, Span{Text: "4200", NoLetterSpacing: true}).LetterSpacing(3).Label("span")
			para = coreRichText(c, Span{Text: "LABEL "}).LetterSpacing(3).Label("inline").Children(func() {
				coreText(c, "4200").LetterSpacing(0)
			})
			coreRichText(c, Span{Text: "LABEL "}).LetterSpacing(3).Label("nested").Children(func() {
				coreRichText(c).LetterSpacing(0).Children(func() { coreText(c, "4200") })
			})
			// A span's own spacing wins over that of the element around it.
			coreRichText(c, Span{Text: "LABEL "}).Label("over").Children(func() {
				coreRichText(c, Span{Text: "4200", LetterSpacing: 3}).LetterSpacing(0)
			})
		})
	}, 600, 400, "plain", "tracked", "span", "inline", "nested", "over")
	plain, tracked, span := b["plain"].W, b["tracked"].W, b["span"].W
	// Three DIPs after each of the four digits, or of three of them.
	if d := tracked - span; d < 3*3-0.5 || d > 3*4+0.5 {
		t.Errorf("tracked text is %v wide, %v with the number untracked", tracked, span)
	}
	if span <= plain {
		t.Errorf("the label untracked but its number is %v wide, untracked %v", span, plain)
	}
	for _, name := range []string{"inline", "nested"} {
		if b[name].W != span {
			t.Errorf("%s: %v wide, the span untracked %v", name, b[name].W, span)
		}
	}
	if b["over"].W <= plain {
		t.Errorf("a span's own spacing within no spacing is %v wide, plain %v", b["over"].W, plain)
	}
	if s := para.spans[1]; !s.NoLetterSpacing || s.LetterSpacing != 0 {
		t.Errorf("the inline element's span %+v", s)
	}
}

// TestSpanPadding checks the padding of a span's background, as CSS
// pads an inline box: PaddingX takes room in the line, PaddingY only
// paints, and each line's piece of a span gets both, as
// box-decoration-break: clone.
func TestSpanPadding(t *testing.T) {
	red := RGB(255, 0, 0)
	const px, py = 6, 3
	chip := Span{Text: "Ctrl K", Background: red, PaddingX: px, PaddingY: py}
	var padded, wrapped *node
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Padding(10).Gap(20).AlignItems(Start).Children(func() {
			coreRichText(c, Span{Text: "See "}, Span{Text: "Ctrl K"}, Span{Text: " now"}).Label("plain")
			padded = coreRichText(c, Span{Text: "See "}, chip, Span{Text: " now"}).Label("padded")
			wrapped = coreRichText(c, Span{Text: "See "}, Span{Text: "Ctrl K now", Background: red, PaddingX: px, PaddingY: py}).Width(70)
		})
	}, 300, 200)
	plain, _ := tt.Find("plain")
	chipped, _ := tt.Find("padded")
	if d := chipped.W - plain.W; d < 2*px-0.01 || d > 2*px+0.01 {
		t.Errorf("padding of %v widens the text by %v", px, d)
	}
	if chipped.H != plain.H {
		t.Errorf("vertical padding makes the text %v high, %v without", chipped.H, plain.H)
	}
	img := tt.Image()
	isRed := func(x, y int) bool { c := img.RGBAAt(x, y); return c.R > 200 && c.G < 60 && c.B < 60 }
	// The background of each line's piece of the span, runes a to b: the
	// glyphs' extent, PaddingX to the sides, on the row of their middle,
	// and their height, PaddingY above and below, for a span on one line.
	check := func(name string, e *node, a, b int) {
		t.Helper()
		ox, oy := e.x+e.contentX(), e.y+e.contentY()
		pieces := 0
		for li, line := range e.tl.Lines {
			x0, x1, size := float32(math.MaxFloat32), float32(-math.MaxFloat32), float32(0)
			for _, g := range line.Glyphs {
				if g.Cluster >= a && g.Cluster < b {
					x0, x1, size = min(x0, g.X), max(x1, g.X+g.Advance), g.Size
				}
			}
			if size == 0 {
				continue
			}
			pieces++
			row := int(oy + line.Baseline - size*0.3)
			left, right := -1, -1
			for x := 0; x < img.Bounds().Dx(); x++ {
				if isRed(x, row) {
					if left < 0 {
						left = x
					}
					right = x + 1
				}
			}
			if want := int(math.Round(float64(ox + x0 - px))); left < want-1 || left > want+1 {
				t.Errorf("%s, line %d: red from x %d, want %d", name, li, left, want)
			}
			if want := int(math.Round(float64(ox + x1 + px))); right < want-1 || right > want+1 {
				t.Errorf("%s, line %d: red to x %d, want %d", name, li, right, want)
			}
			if len(e.tl.Lines) > 1 {
				continue
			}
			col := int(ox + (x0+x1)/2)
			top, bottom := -1, -1
			for y := int(oy+line.Y) - 2*py; y < int(oy+line.Y+line.Height)+2*py; y++ {
				if isRed(col, y) {
					if top < 0 {
						top = y
					}
					bottom = y + 1
				}
			}
			if want := int(math.Round(float64(oy + line.Baseline - size*0.98 - py))); top < want-1 || top > want+1 {
				t.Errorf("%s: red from y %d, want %d", name, top, want)
			}
			if want := int(math.Round(float64(oy + line.Baseline + size*0.34 + py))); bottom < want-1 || bottom > want+1 {
				t.Errorf("%s: red to y %d, want %d", name, bottom, want)
			}
		}
		if want := len(e.tl.Lines); pieces != want {
			t.Errorf("%s: %d pieces of the span on %d lines", name, pieces, want)
		}
	}
	check("padded", padded, 4, 10)
	if len(wrapped.tl.Lines) < 2 {
		t.Fatalf("the wrapped span takes %d lines", len(wrapped.tl.Lines))
	}
	check("wrapped", wrapped, 4, 14)
	for i, line := range wrapped.tl.Lines {
		if line.Width > 70+0.01 {
			t.Errorf("line %d of the wrapped span is %v wide, past 70", i, line.Width)
		}
	}
}
