package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Line backgrounds fill the rows of their lines across the text, not the
// gutter: those a fold hides take no rows, and a wrapped line's every row.
func TestLineBackgrounds(t *testing.T) {
	white := &Theme{Background: ui.RGB(255, 255, 255)}
	blue := ui.RGB(0, 0, 255)
	ct := newEditorTest(t, "go", "func a() {\n\tx()\n\ty()\n}\nfunc b() {}\nz|")
	ct.st.Theme, ct.st.NoActiveLine = white, true
	ct.st.LineBackgrounds = []LineBackground{{First: 0, Last: 4, Color: blue}}
	ct.tt.Frame()
	// filled reports whether x, in DIPs of the 600-wide window, is blue on
	// the row.
	filled := func(x float32, row int) bool {
		img := ct.tt.Image()
		s := float32(img.Bounds().Dx()) / 600
		c := img.RGBAAt(int(x*s), int((float32(row)*ct.st.ed.view.lineHeight+2)*s))
		return c.B > 200 && c.R < 60
	}
	for row, want := range []bool{true, true, true, true, true, false} {
		if filled(500, row) != want {
			t.Fatalf("row %d filled %v, want %v", row, !want, want)
		}
	}
	if filled(2, 0) {
		t.Fatal("the line background filled the gutter")
	}
	// Folded, line 0 hides lines 1 and 2.
	ct.st.Fold(0)
	ct.tt.Frame()
	for row, want := range []bool{true, true, true, false} {
		if filled(500, row) != want {
			t.Fatalf("folded, row %d filled %v, want %v", row, !want, want)
		}
	}
	// Wrapped, the line fills its rows.
	ct = newEditorTest(t, "", strings.Repeat("x", 150)+"\nz|")
	ct.st.Theme, ct.st.NoActiveLine, ct.st.SoftWrap = white, true, true
	ct.st.LineBackgrounds = []LineBackground{{First: 0, Last: 0, Color: blue}}
	ct.tt.Frame()
	ct.tt.Frame()
	if rows := len(ct.st.ed.view.line(0).rows); rows != 3 {
		t.Fatalf("the line wrapped into %d rows, want 3", rows)
	}
	for row, want := range []bool{true, true, true, false} {
		if filled(300, row) != want {
			t.Fatalf("wrapped, row %d filled %v, want %v", row, !want, want)
		}
	}
}

// tint returns the color under the top of the character at offset off,
// as the last frame painted it, above its glyph.
func (ct *editorTest) tint(off int) (r, g, b uint8) {
	ct.t.Helper()
	ed := &ct.st.ed
	cr, ok := ed.caretRect(off)
	if !ok {
		ct.t.Fatalf("offset %d doesn't show", off)
	}
	next, ok := ed.caretRect(off + 1)
	w := ed.view.charWidth
	if ok && next.Y == cr.Y && next.X > cr.X {
		w = next.X - cr.X
	}
	img := ct.tt.Image()
	s := float32(img.Bounds().Dx()) / 600
	x := ed.textX + cr.X - ed.scrollX + w/2
	y := float32(float64(cr.Y)-ed.scrollY) + 2
	c := img.RGBAAt(int(x*s), int(y*s))
	return c.R, c.G, c.B
}

// blueAt reports whether the character at off is tinted blue.
func (ct *editorTest) blueAt(off int) bool {
	r, g, b := ct.tint(off)
	return b > 200 && r < 60 && g < 60
}

// anyBlue reports whether the text's area has blue pixels.
func (ct *editorTest) anyBlue() bool {
	img := ct.tt.Image()
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if c := img.RGBAAt(x, y); c.B > 200 && c.R < 60 && c.G < 60 {
				return true
			}
		}
	}
	return false
}

// Text backgrounds fill their bytes where the editor lays them out: in
// the rows a line wraps into, over tabs, scrolled sideways, under the
// selections, and not in lines folds hide.
func TestTextBackgrounds(t *testing.T) {
	theme := &Theme{Background: ui.RGB(255, 255, 255), Selection: ui.RGB(0, 200, 0), InactiveSelection: ui.RGB(0, 200, 0)}
	blue := ui.RGB(0, 0, 255)
	ct := newEditorTest(t, "", "abc def ghi\nz|")
	ct.st.Theme, ct.st.NoActiveLine = theme, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 4, End: 7, Color: blue}}
	ct.tt.Frame()
	for off, want := range []bool{false, false, false, false, true, true, true, false, false} {
		if ct.blueAt(off) != want {
			t.Fatalf("offset %d tinted %v, want %v", off, !want, want)
		}
	}

	// Across the rows of a wrapped line.
	ct = newEditorTest(t, "", strings.Repeat("x", 150)+"\nz|")
	ct.st.Theme, ct.st.NoActiveLine, ct.st.SoftWrap = theme, true, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 40, End: 130, Color: blue}}
	ct.tt.Frame()
	ct.tt.Frame()
	ed := &ct.st.ed
	if rows := len(ed.view.line(0).rows); rows != 3 {
		t.Fatalf("the line wrapped into %d rows, want 3", rows)
	}
	rowOf := func(off int) float32 { r, _ := ed.caretRect(off); return r.Y }
	if rowOf(40) == rowOf(129) {
		t.Fatal("the range doesn't cross rows")
	}
	for _, off := range []int{40, 60, 75, 100, 129} {
		if !ct.blueAt(off) {
			t.Fatalf("wrapped, offset %d isn't tinted", off)
		}
	}
	for _, off := range []int{0, 39, 130, 149} {
		if ct.blueAt(off) {
			t.Fatalf("wrapped, offset %d is tinted", off)
		}
	}

	// Over tabs, at their width.
	ct = newEditorTest(t, "", "\tab\tcd\nz|")
	ct.st.Theme, ct.st.NoActiveLine = theme, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 3, End: 5, Color: blue}}
	ct.tt.Frame()
	if !ct.blueAt(3) || !ct.blueAt(4) || ct.blueAt(1) || ct.blueAt(5) {
		t.Fatalf("over a tab: tab %v c %v a %v d %v", ct.blueAt(3), ct.blueAt(4), ct.blueAt(1), ct.blueAt(5))
	}
	if r0, _ := ct.st.ed.caretRect(3); func() bool { r1, _ := ct.st.ed.caretRect(4); return r1.X-r0.X <= ct.st.ed.view.charWidth }() {
		t.Fatal("the tab isn't wider than a character")
	}

	// Scrolled sideways.
	long := strings.Repeat("y", 300)
	ct = newEditorTest(t, "", long+"|\nz")
	ct.st.Theme, ct.st.NoActiveLine = theme, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 280, End: 290, Color: blue}}
	ct.tt.Frame()
	if ct.st.ed.scrollX == 0 {
		t.Fatal("the editor didn't scroll sideways")
	}
	if !ct.blueAt(285) || ct.blueAt(295) || ct.blueAt(275) {
		t.Fatal("scrolled sideways, the range isn't where its characters are")
	}

	// Under the selection, which paints over it.
	ct = newEditorTest(t, "", "ab[cdef]gh")
	ct.st.Theme, ct.st.NoActiveLine = theme, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 0, End: 4, Color: blue}}
	ct.tt.Frame()
	for off, want := range []string{"blue", "blue", "green", "green", "green", "green", "white"} {
		r, g, b := ct.tint(off)
		got := "white"
		switch {
		case b > 200 && r < 60 && g < 60:
			got = "blue"
		case g > 150 && r < 60 && b < 60:
			got = "green"
		case r < 200 || g < 200 || b < 200:
			got = "other"
		}
		if got != want {
			t.Fatalf("with a selection, offset %d is %s, want %s", off, got, want)
		}
	}

	// Not in the lines a fold hides.
	ct = newEditorTest(t, "go", "func a() {\n\tx()\n\ty()\n}\nz|")
	ct.st.Theme, ct.st.NoActiveLine = theme, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 12, End: 15, Color: blue}}
	ct.tt.Frame()
	if !ct.blueAt(13) {
		t.Fatal("unfolded, x() isn't tinted")
	}
	ct.st.Fold(0)
	ct.tt.Frame()
	if ct.anyBlue() {
		t.Fatal("a folded line's range is tinted")
	}

	// Empty, reversed, transparent and out of the text, they fill
	// nothing; partly out, they are cut to the text.
	ct = newEditorTest(t, "", "abc def ghi\nz|")
	ct.st.Theme, ct.st.NoActiveLine = theme, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 5, End: 5, Color: blue}, {Start: 6, End: 2, Color: blue},
		{Start: -10, End: -1, Color: blue}, {Start: 1000, End: 2000, Color: blue}, {Start: 0, End: 5}}
	ct.tt.Frame()
	if ct.anyBlue() {
		t.Fatal("an empty or invalid range is tinted")
	}
	ct.st.TextBackgrounds = []TextBackground{{Start: -5, End: 2, Color: blue}, {Start: 12, End: 1000, Color: blue}}
	ct.tt.Frame()
	if !ct.blueAt(0) || !ct.blueAt(1) || ct.blueAt(2) || !ct.blueAt(12) {
		t.Fatal("ranges partly out of the text aren't cut to it")
	}

	// After an edit, before the app computes them again, they stay in
	// the shorter text without panicking, even inside a character.
	ct = newEditorTest(t, "", "héllo wörld\nsecond line|")
	ct.st.Theme, ct.st.NoActiveLine = theme, true
	ct.st.TextBackgrounds = []TextBackground{{Start: 2, End: 9, Color: blue}, {Start: 13, End: 30, Color: blue}}
	ct.tt.Frame()
	if !ct.blueAt(1) {
		t.Fatal("a range starting inside a character doesn't tint it")
	}
	ct.st.Select(0, ct.st.Buffer.Len())
	ct.st.Insert("x")
	ct.tt.Frame()
	ct.st.Select(1, 1)
	ct.st.Insert("é")
	ct.tt.Frame()
}

// ScrollY, SetScrollY and LineTop place lines in DIPs, as the editor lays
// them out, wrapped lines taking their rows.
func TestScrollYAndLineTop(t *testing.T) {
	var b strings.Builder
	for i := range 100 {
		if i == 3 {
			b.WriteString(strings.Repeat("w", 150) + "\n")
			continue
		}
		b.WriteString("func f() {\n")
	}
	ct := newEditorTest(t, "", b.String()+"|")
	ct.st.SoftWrap = true
	ct.tt.Frame()
	ct.tt.Frame()
	lh := float64(ct.st.ed.view.lineHeight)
	if got := ct.st.LineTop(2); got != 2*lh {
		t.Fatalf("line 2's top is %v, want %v", got, 2*lh)
	}
	// The wrapped line takes 3 rows.
	if got := ct.st.LineTop(4) - ct.st.LineTop(3); got != 3*lh {
		t.Fatalf("the wrapped line is %v high, want %v", got, 3*lh)
	}
	if got := ct.st.LineTop(101); got != 103*lh {
		t.Fatalf("the lines are %v high, want %v", got, 103*lh)
	}
	ct.st.SetScrollY(10 * lh)
	ct.tt.Frame()
	if got := ct.st.ScrollY(); got != 10*lh {
		t.Fatalf("scrolled to %v, want %v", got, 10*lh)
	}
	if first, _ := ct.st.VisibleLines(); first != 8 {
		t.Fatalf("the first line shown is %d, want 8", first)
	}
	ct.st.SetScrollY(-5)
	if ct.st.ScrollY() != 0 {
		t.Fatal("scrolled above the top")
	}
	ct.st.SetScrollY(1e9)
	if got, want := ct.st.ScrollY(), ct.st.LineTop(101)-lh; got != want {
		t.Fatalf("scrolled to %v past the end, want %v", got, want)
	}
}
