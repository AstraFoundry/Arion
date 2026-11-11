package text

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// TestStrokeBitmap strokes a square: around its edge, centered on it,
// neither inside its middle nor far outside.
func TestStrokeBitmap(t *testing.T) {
	const n = 20
	pix := make([]byte, n*n)
	for i := range pix {
		pix[i] = 255
	}
	b := strokeBitmap(bitmap{left: 3, top: -n, w: n, h: n, pix: pix}, 2)
	pad := 3 // ⌈2⌉ + 1
	if b.w != n+2*pad || b.h != n+2*pad || b.left != 3-pad || b.top != -n-pad {
		t.Fatalf("stroked box %d×%d at (%d, %d)", b.w, b.h, b.left, b.top)
	}
	at := func(x, y int) byte { return b.pix[(y+pad)*b.w+x+pad] }
	// The square's edge is at 0, between pixels -1 and 0: the stroke
	// covers [-2, 2) of it, pixels -2 and -1 outside and 0 and 1 inside.
	for _, x := range []int{-2, -1, 0, 1} {
		if v := at(x, n/2); v < 250 {
			t.Errorf("pixel %d from the edge has %d of coverage, want full", x, v)
		}
	}
	for _, x := range []int{-3, 3, n / 2} {
		if v := at(x, n/2); v > 5 {
			t.Errorf("pixel %d from the edge has %d of coverage, want none", x, v)
		}
	}
	// Round at the corner: the pixel diagonally out past the stroke's
	// reach is clear, the one at its edge partly covered.
	if v := at(-3, -3); v != 0 {
		t.Errorf("past the corner: %d", v)
	}
	if v := at(-2, -2); v == 0 || v == 255 {
		t.Errorf("at the corner's curve: %d, want partial", v)
	}
}

// TestStrokeBitmapHalfCovered keeps the place of an antialiased edge: a
// column half covered moves the stroke by half a pixel.
func TestStrokeBitmapHalfCovered(t *testing.T) {
	const n = 10
	pix := make([]byte, n*n)
	for y := range n {
		for x := range n {
			switch {
			case x < 5:
				pix[y*n+x] = 255
			case x == 5:
				pix[y*n+x] = 128
			}
		}
	}
	b := strokeBitmap(bitmap{w: n, h: n, pix: pix}, 1)
	pad := 2
	row := b.pix[(n/2+pad)*b.w:]
	// The edge is at 5.5: the stroke covers [4.5, 6.5), half of pixels 4
	// and 6, all of 5.
	for x, want := range map[int]float64{3: 0, 4: 0.5, 5: 1, 6: 0.5, 7: 0} {
		if v := float64(row[x+pad]) / 255; math.Abs(v-want) > 0.1 {
			t.Errorf("pixel %d: %.2f of coverage, want %.2f", x, v, want)
		}
	}
}

func TestStrokeGlyph(t *testing.T) {
	s := newSystem()
	l := s.Layout(Params{Text: "O", Style: Style{Size: 40}})
	g := l.Lines[0].Glyphs[0]
	fill := s.Glyph(g.Font, g.ID, 1, 0, 0, false)
	stroke := s.StrokeGlyph(g.Font, g.ID, 1, 0, 0, false, 4)
	if !fill.OK || !stroke.OK {
		t.Fatalf("glyph %+v, its stroke %+v", fill, stroke)
	}
	// Two pixels out on each side, and one of padding.
	if stroke.W != fill.W+6 || stroke.H != fill.H+6 || stroke.Left != fill.Left-3 || stroke.Top != fill.Top-3 {
		t.Errorf("glyph %+v, its stroke %+v", fill, stroke)
	}
	if again := s.StrokeGlyph(g.Font, g.ID, 1, 0, 0, false, 4); again != stroke {
		t.Errorf("stroked again: %+v, then %+v", stroke, again)
	}
	if wider := s.StrokeGlyph(g.Font, g.ID, 1, 0, 0, false, 8); wider.W != fill.W+10 {
		t.Errorf("a stroke 8 pixels wide is %d wide, the glyph %d", wider.W, fill.W)
	}
	// Ink where the glyph has none, around it and in its hole.
	px := func(gi GlyphImage, x, y int) byte {
		if x < 0 || y < 0 || x >= int(gi.W) || y >= int(gi.H) {
			return 0
		}
		return s.MaskAtlas.Pix[(int(gi.Y)+y)*s.MaskAtlas.W+int(gi.X)+x]
	}
	var outside int
	for y := range int(stroke.H) {
		for x := range int(stroke.W) {
			f := px(fill, x-3, y-3)
			if f == 0 && px(stroke, x, y) > 128 {
				outside++
			}
		}
	}
	if outside < 20 {
		t.Errorf("the stroke covers %d pixels outside the glyph", outside)
	}
}

// TestTabSize puts tab stops every TabSize spaces from the start of each
// line.
func TestTabSize(t *testing.T) {
	s := newSystem()
	s.UseTabSize()
	style := Style{Family: "monospace", Size: 16}
	space := s.Layout(Params{Text: "a a", Style: style}).Width - s.Layout(Params{Text: "aa", Style: style}).Width
	if space <= 0 {
		t.Fatal("a space has no advance")
	}
	// The X of the first glyph of rune i of a layout's line li.
	x := func(l *Layout, li, i int) float32 {
		for _, g := range l.Lines[li].Glyphs {
			if g.Cluster == i {
				return g.X - l.Lines[li].X
			}
		}
		t.Fatalf("no glyph of rune %d in line %d of %q", i, li, l.Params.Text)
		return 0
	}
	for _, c := range []struct {
		text string
		size float32
		at   int
		want float32 // in spaces
	}{
		{"\tb", 4, 1, 4},
		{"\tb", 2, 1, 2},
		{"a\tb", 4, 2, 4},
		{"abcde\tb", 4, 6, 8},
		{"\t\tb", 3, 2, 6},
	} {
		st := style
		st.TabSize = c.size
		l := s.Layout(Params{Text: c.text, Style: st})
		if got := x(l, 0, c.at); math.Abs(float64(got-c.want*space)) > 0.5 {
			t.Errorf("%q with tabs of %v: rune %d at %v, want %v (%v spaces); glyphs %s", c.text, c.size, c.at, got, c.want*space, c.want, glyphList(l.Lines[0]))
		}
	}
	// Each line counts from its start, after a newline and where it wraps.
	st := style
	st.TabSize = 4
	l := s.Layout(Params{Text: "abcdef\n\tb", Style: st})
	if got := x(l, 1, 8); math.Abs(float64(got-4*space)) > 0.5 {
		t.Errorf("after a newline: at %v, want %v; glyphs %s", got, 4*space, glyphList(l.Lines[1]))
	}
	l = s.Layout(Params{Text: "abcdefgh abcdefgh\tb", Style: st, Width: 14 * space})
	if len(l.Lines) != 2 {
		t.Fatalf("%d lines", len(l.Lines))
	}
	if got := x(l, 1, 18); math.Abs(float64(got-12*space)) > 0.5 {
		t.Errorf("on a wrapped line: at %v, want %v", got, 12*space)
	}
}

// glyphList lists the clusters, places and advances of a line's glyphs.
func glyphList(line Line) string {
	var b strings.Builder
	for _, g := range line.Glyphs {
		fmt.Fprintf(&b, "[%d at %v, %v wide] ", g.Cluster, g.X, g.Advance)
	}
	return fmt.Sprintf("%sfrom %v", b.String(), line.X)
}
