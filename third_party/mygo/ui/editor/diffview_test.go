package editor

import (
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/diff"
	"github.com/egoist/mygo/ui"
)

// checkHunks checks that hunks turn old's lines into new's.
func checkHunks(t *testing.T, old, nw string, hunks []diff.Hunk) {
	t.Helper()
	a, b := strings.Split(old, "\n"), strings.Split(nw, "\n")
	var out []string
	i, j := 0, 0
	for _, h := range hunks {
		if h.A0 < i || h.B0 < j || h.A0-i != h.B0-j {
			t.Fatalf("hunk %+v out of order after %d, %d: %+v", h, i, j, hunks)
		}
		for ; i < h.A0; i, j = i+1, j+1 {
			if a[i] != b[j] {
				t.Fatalf("unchanged old line %d %q is %q in new: %+v", i, a[i], b[j], hunks)
			}
			out = append(out, a[i])
		}
		out = append(out, b[h.B0:h.B1]...)
		i, j = h.A1, h.B1
	}
	if len(a)-i != len(b)-j {
		t.Fatalf("tails differ: %d, %d: %+v", len(a)-i, len(b)-j, hunks)
	}
	out = append(out, a[i:]...)
	if got := strings.Join(out, "\n"); got != nw {
		t.Fatalf("hunks make %q, want %q", got, nw)
	}
}

// Edits of the new text compare again only around them, and the hunks
// stay those of the texts.
func TestCodeDiffIncremental(t *testing.T) {
	r := rand.New(rand.NewPCG(9, 10))
	pieces := []string{"x", "\n", "", "line\n", "a\nb\n", "{", "}\n", "\n\n"}
	for round := range 300 {
		var sb strings.Builder
		for i := range 30 + r.IntN(30) {
			sb.WriteString([]string{"func a() {", "\treturn", "}", "", "x := 1", "y"}[i%6])
			sb.WriteByte('\n')
		}
		old := NewBuffer(sb.String())
		nw := NewBuffer(sb.String())
		var d diffModel
		d.set(old, nw)
		d.update()
		for range 20 {
			text := nw.String()
			at := r.IntN(len(text) + 1)
			end := min(len(text), at+r.IntN(8))
			nw.Edit(Edit{Start: at, End: end, Text: pieces[r.IntN(len(pieces))]})
			if r.IntN(3) == 0 {
				d.update()
				checkHunks(t, old.String(), nw.String(), d.hunks)
			}
		}
		d.update()
		checkHunks(t, old.String(), nw.String(), d.hunks)
		if round%10 == 0 {
			// A change of the old text compares the whole.
			old.Edit(Edit{Start: 0, End: 0, Text: "// new first line\n"})
			d.update()
			checkHunks(t, old.String(), nw.String(), d.hunks)
		}
	}
}

func TestCodeDiffRevertAccept(t *testing.T) {
	for _, c := range []struct{ old, new string }{
		{"a\nb\nc", "a\nx\nc"},
		{"a\nb\nc", "a\nc"},
		{"a\nc", "a\nb\nc"},
		{"a\nb", "a"},
		{"a", "a\nb"},
		{"b\na", "a"},
		{"", "a\nb"},
		{"a\nb", ""},
		{"a\nb\n", "a\nb\nc\n"},
	} {
		old, nw := NewBuffer(c.old), NewBuffer(c.new)
		var d diffModel
		d.set(old, nw)
		d.update()
		for i := 0; len(d.hunks) > 0 && i < 10; i++ {
			d.revert(0)
			d.update()
		}
		if nw.String() != c.old {
			t.Errorf("reverting %q to %q gave %q", c.new, c.old, nw.String())
		}
		old, nw = NewBuffer(c.old), NewBuffer(c.new)
		d = diffModel{}
		d.set(old, nw)
		d.update()
		for i := 0; len(d.hunks) > 0 && i < 10; i++ {
			d.accept(0)
			d.update()
		}
		if old.String() != c.new {
			t.Errorf("accepting %q over %q gave %q", c.new, c.old, old.String())
		}
	}
}

func TestDiffViewer(t *testing.T) {
	var oldText, newText strings.Builder
	for i := range 200 {
		line := "line " + strconv.Itoa(i)
		oldText.WriteString(line + "\n")
		if i == 100 {
			newText.WriteString("changed\n")
			continue
		}
		newText.WriteString(line + "\n")
	}
	for _, layout := range []DiffLayout{DiffSplit, DiffUnified} {
		d := &DiffState{Old: NewBuffer(oldText.String()), New: NewBuffer(newText.String()), Layout: layout, Editable: true}
		tt := ui.NewTester(func(c *ui.Context) {
			DiffView(c, d).Fill()
		}, 900, 500)
		tt.Frame()
		if hs := d.Hunks(); len(hs) != 1 || hs[0] != (DiffHunk{100, 101, 100, 101}) {
			t.Fatalf("hunks %+v", hs)
		}
		if a, del := d.Stats(); a != 1 || del != 1 {
			t.Fatalf("stats %d, %d", a, del)
		}
		// Unchanged lines collapse but 3 around the change: 97 before, 97
		// after, with the empty line after the last newline.
		gaps := d.gapList
		if len(gaps) != 2 || gaps[0].hidden != 97 || gaps[1].hidden != 97 {
			t.Fatalf("gaps %+v", gaps)
		}
		ed := &d.eds[1]
		total := ed.view.heights.total()
		lh := float64(ed.view.lineHeight)
		want := float64(3+1+3)*lh + 2*diffSeparatorHeight
		if layout == DiffUnified {
			want += lh
		}
		if total != want {
			t.Fatalf("%v: height %v, want %v", layout, total, want)
		}
		// Expanding shows the lines.
		d.expand(gaps[0].a, 1, true)
		tt.Frame()
		if d.gapList[0].hidden != 0 {
			t.Fatalf("expanded gap %+v", d.gapList[0])
		}
		// Editing the new side follows: reverting the change by hand.
		ed.sync()
		r := &ed.view.text
		st := &State{}
		_ = st
		a, z := r.LineStart(100), r.LineEnd(100)
		d.New.Edit(Edit{Start: a, End: z, Text: "line 100"})
		tt.Frame()
		if hs := d.Hunks(); len(hs) != 0 {
			t.Fatalf("hunks after reverting by hand %+v", hs)
		}
		d.New.Undo()
		tt.Frame()
		d.Revert(0)
		tt.Frame()
		if d.New.String() != d.Old.String() {
			t.Fatal("Revert left the texts different")
		}
	}
}

// A split view's sides are as high as each other, line for line.
func TestDiffViewerAlign(t *testing.T) {
	old := "a\nb\nc\nd\ne\nf\n"
	nw := "a\nB\nB2\nB3\nc\nf\ng\n"
	for _, wrap := range []bool{false, true} {
		d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), ExpandUnchanged: true, Wrap: wrap,
			Annotations:      []DiffAnnotation{{Side: DiffNew, Line: 2, Height: 40}, {Side: DiffOld, Line: 0, Height: 25}},
			RenderAnnotation: func(c *ui.Context, a DiffAnnotation) { ui.Text(c, "note") }}
		tt := ui.NewTester(func(c *ui.Context) {
			DiffView(c, d).Fill()
		}, 600, 400)
		tt.Frame()
		ha, hb := &d.eds[0].view.heights, &d.eds[1].view.heights
		if ha.total() != hb.total() {
			t.Fatalf("wrap %v: heights %v and %v", wrap, ha.total(), hb.total())
		}
		// Unchanged lines line up.
		for _, p := range [][2]int{{0, 0}, {2, 4}, {5, 5}} {
			if ga, gb := ha.geometry(p[0]).rows, hb.geometry(p[1]).rows; ga != gb {
				t.Fatalf("wrap %v: old line %d at %v, new line %d at %v", wrap, p[0], ga, p[1], gb)
			}
		}
		if !tt.HasText("note") {
			t.Fatal("no annotation")
		}
	}
}

func TestDiffViewerLineSelection(t *testing.T) {
	var got DiffLineRange
	d := &DiffState{Old: NewBuffer("a\nb\nc\n"), New: NewBuffer("a\nx\nc\n"), Layout: DiffUnified, ExpandUnchanged: true,
		LineSelection: true, OnLineSelection: func(r DiffLineRange) { got = r }}
	tt := ui.NewTester(func(c *ui.Context) {
		DiffView(c, d).Fill()
	}, 600, 400)
	tt.Frame()
	ed := &d.eds[1]
	lh := ed.view.lineHeight
	// The number of new line 2, below the deleted line.
	y := float32(ed.view.heights.geometry(1).rows) + lh/2
	tt.ClickAt(ed.gutterW-8, y)
	if r, ok := d.SelectedLines(); !ok || r != (DiffLineRange{DiffNew, 1, 1}) || got != r {
		t.Fatalf("selected %+v %v, reported %+v", r, ok, got)
	}
	tt.ClickAtWith(ui.Shift, ed.gutterW-8, float32(ed.view.heights.geometry(2).rows)+lh/2)
	if r, _ := d.SelectedLines(); r != (DiffLineRange{DiffNew, 1, 2}) {
		t.Fatalf("extended %+v", r)
	}
}

// The app runs commands and reads the caret in the side that has the
// keyboard, or had it last: the new side's until the old one has it.
func TestDiffViewerActiveSide(t *testing.T) {
	var oldText, newText strings.Builder
	for i := range 40 {
		line := "line " + strconv.Itoa(i)
		oldText.WriteString(line + "\n")
		switch i {
		case 5, 30:
			newText.WriteString("changed " + strconv.Itoa(i) + "\n")
		default:
			newText.WriteString(line + "\n")
		}
	}
	d := &DiffState{Old: NewBuffer(oldText.String()), New: NewBuffer(newText.String()), ExpandUnchanged: true}
	type caret struct {
		side   DiffSide
		line   int
		hunk   int
		inside bool
	}
	at := func() caret {
		h, in := d.CurrentHunk()
		return caret{d.ActiveSide(), d.CursorPos().Line, h, in}
	}
	// Before the viewer is built, the new side's caret is at the start.
	if c := at(); c != (caret{DiffNew, 0, -1, false}) || d.Cursor() != 0 {
		t.Fatalf("before building: %+v, offset %d", c, d.Cursor())
	}
	tt := ui.NewTester(func(c *ui.Context) {
		DiffView(c, d).Fill()
	}, 900, 500)
	tt.Frame()
	d.GoToHunk(1)
	tt.Frame()
	if c := at(); c != (caret{DiffNew, 30, 1, true}) || d.Cursor() != d.New.Offset(Pos{Line: 30}) {
		t.Fatalf("after GoToHunk(1): %+v, offset %d", c, d.Cursor())
	}
	// Between changes, the current one is the one before the caret: the
	// view shows the second change in its middle, lines of the first
	// above it.
	lh := float32(d.eds[1].view.lineHeight)
	tt.ClickAt(200, 2.5*lh)
	tt.Frame()
	if c := at(); c.side != DiffOld || c.line <= 5 || c.line >= 30 || c.hunk != 0 || c.inside {
		t.Fatalf("clicked in the old side: %+v", c)
	}
	// The old side stays active as the window loses the keyboard.
	tt.SetFocused(false)
	tt.Frame()
	if side := d.ActiveSide(); side != DiffOld {
		t.Fatalf("unfocused: side %v", side)
	}
	// GoToHunk moves the caret CursorPos reads, so an app's Next Change
	// steps from it.
	h, _ := d.CurrentHunk()
	d.GoToHunk(h + 1)
	tt.Frame()
	if c := at(); c != (caret{DiffOld, 30, 1, true}) {
		t.Fatalf("after GoToHunk in the old side: %+v", c)
	}
	tt.SetFocused(true)
	tt.Frame()
	if !d.Run(Find) {
		t.Fatal("Run(Find) did nothing")
	}
	tt.Frame()
	if !d.eds[0].find.open || d.eds[1].find.open {
		t.Fatalf("find bars open: old %v, new %v", d.eds[0].find.open, d.eds[1].find.open)
	}
	// Back in the new side, between the changes, Find selects the match
	// after the caret.
	tt.ClickAt(650, 2.5*lh)
	tt.Frame()
	d.Find("changed")
	tt.Frame()
	if c := at(); c.side != DiffNew || c.line != 30 || !d.eds[1].find.open {
		t.Fatalf("Find in the new side: %+v, open %v", c, d.eds[1].find.open)
	}
	// A unified view has only the new side's editor.
	d.Layout = DiffUnified
	tt.Frame()
	if side := d.ActiveSide(); side != DiffNew {
		t.Fatalf("unified: side %v", side)
	}
}

// The caret is in a hunk on the lines it changes, and in a deletion on
// the line of the side without its lines where they were; after the last
// line of a hunk, it is after it.
func TestDiffViewerCurrentHunk(t *testing.T) {
	d := &DiffState{Old: NewBuffer("a\nb\nc\nd\ne\n"), New: NewBuffer("a\nB\nc\ne\n"), ExpandUnchanged: true}
	if n := len(d.Hunks()); n != 2 {
		t.Fatalf("%d hunks, want 2", n)
	}
	ed := d.editorOf(DiffNew)
	for _, c := range []struct {
		line, hunk int
		inside     bool
	}{{0, -1, false}, {1, 0, true}, {2, 0, false}, {3, 1, true}, {4, 1, false}} {
		off := d.New.Offset(Pos{Line: c.line})
		ed.setSels([]sel{ed.newSel(off, off)}, revealNone)
		if h, in := d.CurrentHunk(); h != c.hunk || in != c.inside {
			t.Errorf("new line %d: hunk %d inside %v, want %d %v", c.line, h, in, c.hunk, c.inside)
		}
	}
}

// The wheel over what the diff shows over its lines, as a comment under a
// line, scrolls the diff, as it does over the lines.
func TestDiffScrollsUnderAnnotation(t *testing.T) {
	var a []string
	for i := range 300 {
		a = append(a, "line "+strconv.Itoa(i))
	}
	b := append([]string(nil), a...)
	b[1] = "changed"
	d := &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), Layout: DiffUnified, ExpandUnchanged: true}
	d.Annotations = []DiffAnnotation{{Side: DiffNew, Line: 1, Height: 300, Key: "comment"}}
	d.RenderAnnotation = func(c *ui.Context, an DiffAnnotation) { ui.Column(c).Fill().Children(func() { ui.Text(c, "comment") }) }
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 600, 400)
	tt.Frame()
	r, ok := tt.Find("comment")
	if !ok {
		t.Fatal("no comment shown")
	}
	tt.Scroll(r.X+r.W/2, r.Y+r.H/2, 0, 40)
	tt.Frame()
	if y := d.eds[1].scrollY; y != 40 {
		t.Fatalf("scrolled to %v over the comment, want 40", y)
	}
}

// The wheel over a separator, a hunk's header or the bar expanding the
// unmodified lines, scrolls the diff, as it does over the lines, in both
// layouts.
func TestDiffScrollsOverSeparator(t *testing.T) {
	var patch strings.Builder
	patch.WriteString("--- a/x\n+++ b/x\n")
	for h := range 12 {
		n := strconv.Itoa(1 + h*100)
		patch.WriteString("@@ -" + n + ",6 +" + n + ",6 @@ hunk " + strconv.Itoa(h) + "\n")
		for i := range 6 {
			patch.WriteString("-old " + strconv.Itoa(i) + "\n+new " + strconv.Itoa(i) + "\n")
		}
	}
	files, err := ParsePatch(patch.String())
	if err != nil {
		t.Fatal(err)
	}
	var a []string
	for i := range 600 {
		a = append(a, "line "+strconv.Itoa(i))
	}
	b := append([]string(nil), a...)
	for i := 0; i < len(b); i += 50 {
		b[i] = "changed"
	}
	for _, tc := range []struct {
		name, find string
		diff       func() *DiffState
	}{
		{"header", "@@ -201,6 +201,6 @@ hunk 2", func() *DiffState {
			d := &DiffState{}
			d.SetPatch(files[0])
			return d
		}},
		{"separator", "unmodified lines", func() *DiffState {
			return &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n"))}
		}},
	} {
		for _, layout := range []DiffLayout{DiffSplit, DiffUnified} {
			d := tc.diff()
			d.Layout = layout
			tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 800, 600)
			tt.Frame()
			tt.Frame()
			var r ui.Rect
			found := false
			for _, n := range tt.Texts() {
				if strings.Contains(n, tc.find) {
					r, found = tt.Find(n)
					break
				}
			}
			if !found {
				t.Fatalf("%s, layout %v: no %q in %q", tc.name, layout, tc.find, tt.Texts())
			}
			tt.Scroll(r.X+r.W/2, r.Y+r.H/2, 0, 40)
			tt.Frame()
			if y := d.eds[1].scrollY; y != 40 {
				t.Errorf("%s, layout %v: scrolled to %v over the %s, want 40", tc.name, layout, y, tc.name)
			}
		}
	}
}

// A filler's stripes are those in sight, however high it is, at -45°
// every 8 DIPs from its bottom-left corner, which they keep to as it
// scrolls.
func TestFillerStripes(t *testing.T) {
	v := ui.Rect{X: 100, Y: 0, W: 600, H: 800}
	// pieces returns the stripes in v of a filler whose bottom-left corner
	// is at (100, bottom), where x+y is each at its bottom.
	pieces := func(v ui.Rect, bottom float64) []float64 {
		var out []float64
		fillerStripes(v, [2]float64{100, bottom}, func(a, b [2]float32) {
			if a[1] != v.Y+v.H || b[1] != v.Y || math.Abs(float64(b[0]-a[0]-v.H)) > 1e-3 {
				t.Fatalf("stripe from %v to %v, want from the bottom of %v to its top at -45°", a, b, v)
			}
			if a[0] >= v.X+v.W || b[0] <= v.X {
				t.Fatalf("stripe from %v to %v misses %v", a, b, v)
			}
			out = append(out, float64(a[0])+float64(a[1]))
		})
		return out
	}
	// 12M DIPs high, 700k lines, scrolled to its middle.
	bottom := 6e6 + 0.3
	got := pieces(v, bottom)
	if want := int(v.W+v.H) / 8; len(got) < want-1 || len(got) > want+1 {
		t.Fatalf("%d stripes, want about %d", len(got), want)
	}
	for i, s := range got {
		if d := math.Mod(s-(100+bottom), 8); math.Abs(d) > 1e-3 && math.Abs(math.Abs(d)-8) > 1e-3 {
			t.Fatalf("stripe %d at %v, off the corner's phase by %v", i, s, d)
		}
		if i > 0 && math.Abs(s-got[i-1]-8) > 1e-3 {
			t.Fatalf("stripes at %v and %v, want 8 DIPs apart", got[i-1], s)
		}
	}
	// Scrolled by 13.25 DIPs, the stripes move with the filler.
	moved := pieces(v, bottom-13.25)
	for _, s := range moved {
		if d := math.Mod(s+13.25-(100+bottom), 8); math.Abs(d) > 1e-3 && math.Abs(math.Abs(d)-8) > 1e-3 {
			t.Fatalf("scrolled, a stripe at %v, off the filler's phase by %v", s, d)
		}
	}
	if got := pieces(ui.Rect{X: 100, Y: 50}, bottom); len(got) != 0 {
		t.Fatalf("%d stripes in an empty rectangle", len(got))
	}
}

// A split view hatches the filler of rows a side lacks with diagonal
// stripes, as many as show, however many lines it stands for: not a
// line across its top.
func TestDiffViewerFillerHatched(t *testing.T) {
	var nw strings.Builder
	nw.WriteString("a\n")
	const added = 200_000
	for range added {
		nw.WriteString("0x4d, 0x5a, 0x90, 0x00,\n")
	}
	nw.WriteString("z\n")
	d := &DiffState{Old: NewBuffer("a\nz\n"), New: NewBuffer(nw.String()), Layout: DiffSplit}
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 900, 500)
	tt.SetScale(2)
	tt.Frame()
	ed := &d.eds[0]
	lh := ed.view.lineHeight
	if h := ed.view.heights.total(); h < added*float64(lh) {
		t.Fatalf("the old side is %v DIPs high, want a filler of %d lines", h, added)
	}
	bg := ed.theme.Background
	// stripes returns where, in DIPs across the old side's text, the row
	// at y in the window is darker than the background: the middle of
	// each run of such pixels.
	stripes := func(y float32) []float32 {
		img := tt.Image()
		s := float32(img.Bounds().Dx()) / 900
		var out []float32
		from := -1
		x0, x1 := int((ed.r.X+ed.gutterW+2)*s), int((ed.r.X+ed.r.W-scrollbarWidth-2)*s)
		for x := x0; x <= x1; x++ {
			c := img.RGBAAt(x, int(y*s))
			dark := x < x1 && math.Abs(float64(c.R)-float64(bg.R))+math.Abs(float64(c.G)-float64(bg.G))+math.Abs(float64(c.B)-float64(bg.B)) > 3
			switch {
			case dark && from < 0:
				from = x
			case !dark && from >= 0:
				out = append(out, float32(from+x-1)/2/s)
				from = -1
			}
		}
		return out
	}
	// In the filler's top row of pixels, a stripe every 8 DIPs, not a
	// line across.
	top := ed.r.Y + lh + 0.25
	n := (ed.r.W - ed.gutterW - scrollbarWidth - 4) / 8
	if got := stripes(top); float32(len(got)) < n-2 || float32(len(got)) > n+2 {
		t.Fatalf("at the filler's top, the row is dark in %d runs %v, want a stripe every 8 DIPs (%v)", len(got), got, n)
	}
	// Halfway down, the stripes go up to the right, at -45°.
	tt.Scroll(ed.r.X+ed.r.W/2, ed.r.Y+ed.r.H/2, 0, float32(added/2)*lh+0.5)
	tt.Frame()
	if ed.scrollY < float64(added/4)*float64(lh) {
		t.Fatalf("scrolled to %v", ed.scrollY)
	}
	y := ed.r.Y + ed.r.H/2
	a, b := stripes(y), stripes(y+3)
	if float32(len(a)) < n-2 || float32(len(a)) > n+2 {
		t.Fatalf("halfway down the filler, the row is dark in %d runs %v, want a stripe every 8 DIPs (%v)", len(a), a, n)
	}
	for i := 1; i < len(a)-1; i++ {
		// 3 DIPs lower, a stripe is 3 DIPs to the left.
		found := false
		for _, x := range b {
			if math.Abs(float64(x-(a[i]-3))) < 1 {
				found = true
			}
		}
		if !found {
			t.Fatalf("a stripe at %v, and 3 DIPs lower at %v, want one at %v", a[i], b, a[i]-3)
		}
	}
}

// A diff as high as its lines, in a scroll container, shows a horizontal
// scroll bar at the bottom of the part of it the window shows, on both
// sides of a split diff, and dragging it scrolls the lines sideways.
func TestFitContentDiffHorizontalScrollbar(t *testing.T) {
	var a []string
	for i := range 200 {
		a = append(a, "line "+strconv.Itoa(i)+" "+strings.Repeat("x", 200))
	}
	b := append([]string(nil), a...)
	b[1] = "changed " + strings.Repeat("y", 200)
	for _, layout := range []DiffLayout{DiffSplit, DiffUnified} {
		d := &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), Layout: layout, FitContent: true, ExpandUnchanged: true}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Scroll(c).Fill().Children(func() { DiffView(c, d).FillWidth() })
		}, 900, 400)
		tt.Frame()
		tt.Frame()
		eds := []*editor{&d.eds[1]}
		if layout == DiffSplit {
			eds = append(eds, &d.eds[0])
		}
		for _, ed := range eds {
			if ed.r.H <= 400 {
				t.Fatalf("%v: editor %v high, want its lines", layout, ed.r.H)
			}
			if _, h := ed.scrollbars(); !h {
				t.Fatalf("%v: no horizontal scroll bar over lines wider than the view", layout)
			}
			if want := 400 - ed.r.Y - scrollbarWidth; ed.hbarY != want {
				t.Fatalf("%v: horizontal bar at %v, want %v, at the window's bottom", layout, ed.hbarY, want)
			}
		}
		ed := eds[0]
		y := ed.r.Y + ed.hbarY + scrollbarWidth/2
		x := ed.r.X + ed.textX + 2
		tt.Move(x, y)
		tt.Press(x, y)
		tt.Move(x+200, y)
		tt.Release(x+200, y)
		tt.Frame()
		if ed.scrollX <= 0 {
			t.Fatalf("%v: dragging the bar scrolled to %v", layout, ed.scrollX)
		}
	}
}

// A diff's scroll position, the height it shows and where its lines are
// read as the last frame laid them out, however it scrolled: the wheel,
// the keys, the scroll bar's thumb, GoToHunk and SetScrollY; and the view
// built after a scroll reads it, as a minimap beside the diff does.
func TestDiffScrollPosition(t *testing.T) {
	var a []string
	for i := range 600 {
		a = append(a, "line "+strconv.Itoa(i))
	}
	b := append([]string(nil), a...)
	for i := 25; i < len(b); i += 50 {
		b[i] = "changed"
	}
	for _, layout := range []DiffLayout{DiffSplit, DiffUnified} {
		d := &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), Layout: layout, Editable: true}
		var built float64
		tt := ui.NewTester(func(c *ui.Context) {
			built = d.ScrollY()
			DiffView(c, d).Fill()
		}, 800, 600)
		tt.Frame()
		tt.Frame()
		check := func(how string) {
			t.Helper()
			tt.Frame()
			y := d.ScrollY()
			if y != d.eds[1].scrollY || layout == DiffSplit && y != d.eds[0].scrollY {
				t.Fatalf("layout %v, %s: ScrollY %v, the sides %v and %v", layout, how, y, d.eds[0].scrollY, d.eds[1].scrollY)
			}
			if built != y {
				t.Fatalf("layout %v, %s: the view was last built at %v, ScrollY %v", layout, how, built, y)
			}
		}
		if h := d.ViewportHeight(); h != 600 {
			t.Fatalf("layout %v: ViewportHeight %v, want 600", layout, h)
		}
		total := d.LineTop(DiffNew, d.New.LineCount())
		if total != d.eds[1].view.heights.total() || total <= 600 {
			t.Fatalf("layout %v: the diff is %v high, its rows %v", layout, total, d.eds[1].view.heights.total())
		}
		// The collapsed lines take no room: far less than 600 lines.
		if lh := float64(d.eds[1].view.lineHeight); total >= 600*lh {
			t.Fatalf("layout %v: the diff is %v high, as all its lines", layout, total)
		}
		if old := d.LineTop(DiffOld, d.Old.LineCount()); old != total {
			t.Errorf("layout %v: the old side is %v high, the new %v", layout, old, total)
		}
		// The wheel.
		tt.Scroll(600, 300, 0, 40)
		check("the wheel")
		if d.ScrollY() != 40 {
			t.Fatalf("layout %v: ScrollY %v after the wheel, want 40", layout, d.ScrollY())
		}
		// The keys, in the new side's editor.
		tt.ClickAt(700, 300)
		before := d.ScrollY()
		for range 3 {
			tt.Key(0, ui.KeyPageDown)
		}
		check("the keys")
		if d.ScrollY() <= before {
			t.Fatalf("layout %v: Page Down left ScrollY at %v", layout, d.ScrollY())
		}
		// The thumb.
		ed := &d.eds[1]
		t0, n := ed.bar(false).Thumb()
		x, y := ed.r.X+ed.r.W-scrollbarWidth/2, ed.r.Y+t0+n/2
		before = d.ScrollY()
		// Up, as Page Down went to the end.
		tt.Press(x, y)
		tt.Move(x, y-100)
		tt.Release(x, y-100)
		check("the thumb")
		if d.ScrollY() >= before {
			t.Fatalf("layout %v: dragging the thumb left ScrollY at %v", layout, d.ScrollY())
		}
		// GoToHunk, which scrolls as the frame lays the view out.
		last := len(d.Hunks()) - 1
		d.GoToHunk(last)
		check("GoToHunk")
		top := d.LineTop(DiffNew, d.Hunks()[last].NewStart)
		if y := d.ScrollY(); top < y || top >= y+d.ViewportHeight() {
			t.Fatalf("layout %v: GoToHunk showed %v to %v, the hunk at %v", layout, y, y+d.ViewportHeight(), top)
		}
		// SetScrollY, as far as the diff scrolls.
		// Line 323, the first after the lines collapsed under line 275's
		// change.
		d.SetScrollY(d.LineTop(DiffNew, 323))
		check("SetScrollY")
		if d.ScrollY() != d.LineTop(DiffNew, 323) {
			t.Fatalf("layout %v: SetScrollY scrolled to %v, want %v", layout, d.ScrollY(), d.LineTop(DiffNew, 323))
		}
		if first := d.eds[1].shown[0]; first != 323 {
			t.Errorf("layout %v: SetScrollY to line 323 shows line %d first", layout, first)
		}
		d.SetScrollY(-5)
		check("SetScrollY above the top")
		if d.ScrollY() != 0 {
			t.Fatalf("layout %v: SetScrollY(-5) scrolled to %v", layout, d.ScrollY())
		}
		d.SetScrollY(math.MaxFloat64)
		check("SetScrollY past the end")
		if y := d.ScrollY(); y != d.eds[1].scrollMax() || y <= 0 {
			t.Fatalf("layout %v: SetScrollY past the end scrolled to %v, want %v", layout, y, d.eds[1].scrollMax())
		}
	}
}

// LineTop places the lines of both sides where they show: side by side,
// a changed line across from its new version; one over the other, the
// deleted line above the new one, and an unchanged line of the old text
// where it is in the new.
func TestDiffLineTop(t *testing.T) {
	a := []string{"a", "b", "c", "d", "e", "f"}
	b := []string{"a", "B", "c", "d", "e", "f", "g"}
	for _, layout := range []DiffLayout{DiffSplit, DiffUnified} {
		d := &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), Layout: layout, ExpandUnchanged: true}
		tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 800, 600)
		tt.Frame()
		tt.Frame()
		lh := float64(d.eds[1].view.lineHeight)
		for line := range b {
			if got, want := d.LineTop(DiffNew, line), float64(line)*lh; layout == DiffSplit && got != want {
				t.Errorf("split: new line %d at %v, want %v", line, got, want)
			}
		}
		switch layout {
		case DiffSplit:
			if got, want := d.LineTop(DiffOld, 1), d.LineTop(DiffNew, 1); got != want {
				t.Errorf("split: old line 1 at %v, its new version at %v", got, want)
			}
		case DiffUnified:
			if got, want := d.LineTop(DiffOld, 1), 1*lh; got != want {
				t.Errorf("unified: deleted line 1 at %v, want %v", got, want)
			}
			if got, want := d.LineTop(DiffNew, 1), 2*lh; got != want {
				t.Errorf("unified: new line 1 at %v, want %v below the deleted one", got, want)
			}
			for _, line := range []int{0, 2, 5} {
				if got, want := d.LineTop(DiffOld, line), d.LineTop(DiffNew, line); got != want {
					t.Errorf("unified: unchanged old line %d at %v, the new at %v", line, got, want)
				}
			}
			if got, want := d.LineTop(DiffNew, len(b)), float64(len(b)+1)*lh; got != want {
				t.Errorf("unified: the diff is %v high, want %v", got, want)
			}
		}
		if got, want := d.LineTop(DiffOld, len(a)), d.LineTop(DiffNew, len(b)); got != want {
			t.Errorf("layout %v: the old side ends at %v, the new at %v", layout, got, want)
		}
	}
}

// KeepScrollbars shows the diff's thumb with the pointer away, and
// NoScrollbars takes it away, with its room, on both sides.
func TestDiffScrollbarsKeptAndHidden(t *testing.T) {
	var a []string
	for i := range 200 {
		a = append(a, "line "+strconv.Itoa(i))
	}
	b := append([]string(nil), a...)
	b[100] = "changed"
	for _, layout := range []DiffLayout{DiffSplit, DiffUnified} {
		d := &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), Layout: layout, ExpandUnchanged: true}
		tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 800, 600)
		tt.Move(-10, -10)
		tt.Frame()
		column := func() []uint32 {
			img := tt.Image()
			s := img.Bounds().Dx() / 800
			var px []uint32
			for y := 0; y < 600*s; y++ {
				for x := (800 - scrollbarWidth) * s; x < 800*s; x++ {
					c := img.RGBAAt(x, y)
					px = append(px, uint32(c.R)<<24|uint32(c.G)<<16|uint32(c.B)<<8|uint32(c.A))
				}
			}
			return px
		}
		none := column()
		d.KeepScrollbars = true
		tt.Frame()
		if slices.Equal(column(), none) {
			t.Errorf("layout %v: a kept thumb doesn't show with the pointer away", layout)
		}
		d.KeepScrollbars, d.NoScrollbars = false, true
		tt.Move(400, 300)
		tt.Frame()
		for i := range d.eds {
			if i == 0 && layout == DiffUnified {
				continue
			}
			ed := &d.eds[i]
			if v, h := ed.scrollbars(); v || h || ed.trackWidth() != 0 {
				t.Errorf("layout %v, side %d: NoScrollbars left bars %v %v, room %v", layout, i, v, h, ed.trackWidth())
			}
		}
		if layout == DiffSplit && d.eds[0].view.wrapWidth != d.eds[1].view.wrapWidth {
			t.Errorf("NoScrollbars: the sides wrap at %v and %v", d.eds[0].view.wrapWidth, d.eds[1].view.wrapWidth)
		}
	}
}

// A minimap beside a diff, as its guide builds one, shows where the view
// is and scrolls the diff where it is pressed and dragged, and follows
// the diff as it scrolls.
func TestDiffMinimap(t *testing.T) {
	var a []string
	for i := range 600 {
		a = append(a, "line "+strconv.Itoa(i))
	}
	b := append([]string(nil), a...)
	for i := 25; i < len(b); i += 50 {
		b[i] = "changed"
	}
	d := &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), KeepScrollbars: true}
	var box ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		var scale float64
		ui.Row(c).Fill().Children(func() {
			DiffView(c, d).Grow(1).FillHeight()
			mini := ui.Box(c).Width(18).FillHeight()
			mini.Draw(func(p *ui.Painter, r ui.Rect) {
				scale = float64(r.H) / max(d.LineTop(DiffNew, d.New.LineCount()), 1)
				at := func(y float64) float32 { return r.Y + float32(y*scale) }
				for _, h := range d.Hunks() {
					top := at(d.LineTop(DiffNew, h.NewStart))
					bottom := at(d.LineTop(DiffNew, max(h.NewEnd, h.NewStart+1)))
					p.Fill(ui.Rect{X: r.X, Y: top, W: r.W, H: max(bottom-top, 2)}, ui.RGB(46, 160, 67), 0)
				}
				box = ui.Rect{X: r.X, Y: at(d.ScrollY()), W: r.W, H: float32(d.ViewportHeight() * scale)}
				p.Stroke(box, c.Theme().Border, 0, 1)
			})
			mini.HandleInput(func(ev ui.InputEvent) bool {
				if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerMove && ev.Button == 0 {
					d.SetScrollY(float64(ev.Y)/max(scale, 1e-9) - d.ViewportHeight()/2)
					return true
				}
				return ev.Kind == ui.InputPointerUp
			})
		})
	}, 800, 600)
	tt.Frame()
	tt.Frame()
	total := d.LineTop(DiffNew, d.New.LineCount())
	if box.Y != 0 || math.Abs(float64(box.H)-600*600/total) > 0.01 {
		t.Fatalf("the view's box at %v, want at the top, %v high", box, 600*600/total)
	}
	tt.Press(791, 300)
	tt.Move(791, 400)
	tt.Release(791, 400)
	tt.Frame()
	want := 400*total/600 - 300
	if want <= 0 || want >= d.eds[1].scrollMax() {
		t.Fatalf("the fixture scrolls to %v, out of 0 to %v", want, d.eds[1].scrollMax())
	}
	if math.Abs(d.ScrollY()-want) > 0.01 {
		t.Fatalf("dragging the minimap to 400 scrolled to %v, want %v", d.ScrollY(), want)
	}
	if mid := float64(box.Y + box.H/2); math.Abs(mid-400) > 0.01 {
		t.Fatalf("the view's box is centered at %v, want 400", mid)
	}
	// The wheel over the diff moves the box.
	before := box.Y
	tt.Scroll(600, 300, 0, -100)
	tt.Frame()
	if box.Y >= before {
		t.Fatalf("the view's box stayed at %v as the diff scrolled up", box.Y)
	}
}

// Unchanged lines collapsed into a separator are at the separator's top,
// before the first change, between changes and after the last, on both
// sides of both layouts; past the last line is the height of the diff.
func TestDiffLineTopCollapsed(t *testing.T) {
	var a []string
	for i := range 100 {
		a = append(a, "line "+strconv.Itoa(i))
	}
	b := append([]string(nil), a...)
	b[20], b[60] = "changed", "changed"
	for _, layout := range []DiffLayout{DiffSplit, DiffUnified} {
		d := &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), Layout: layout}
		seps := map[int]float64{} // by the old line the gap starts at
		d.RenderSeparator = func(c *ui.Context, sep ui.Element, s *DiffSeparator) {
			seps[s.a] = float64(sep.Bounds().Y)
			DefaultSeparator(c, sep, s)
		}
		tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 800, 2000)
		tt.Frame()
		tt.Frame()
		// The gaps: lines 0 to 16, 24 to 56 and 64 to 99 collapse.
		for _, gap := range []struct{ start, line int }{{0, 5}, {21, 40}, {61, 80}} {
			top, ok := seps[gap.start]
			if !ok {
				t.Fatalf("layout %v: no separator of the gap at %d: %v", layout, gap.start, seps)
			}
			for _, side := range []DiffSide{DiffOld, DiffNew} {
				if got := d.LineTop(side, gap.line); got != top {
					t.Errorf("layout %v, side %v: collapsed line %d at %v, its separator's top %v", layout, side, gap.line, got, top)
				}
			}
		}
		total := d.eds[1].view.heights.total()
		for _, side := range []DiffSide{DiffOld, DiffNew} {
			if got := d.LineTop(side, 100); got != total {
				t.Errorf("layout %v, side %v: past the last line at %v, want the height %v", layout, side, got, total)
			}
		}
	}
}

// A view built after the diff or the editor was resized reads the height
// it shows now, though the scroll and the content stay.
func TestViewportHeightAfterResize(t *testing.T) {
	d := &DiffState{Old: NewBuffer("a\nb\nc"), New: NewBuffer("a\nB\nc")}
	st := &State{Buffer: NewBuffer("a\nb\nc")}
	var diffH, editorH float64
	tt := ui.NewTester(func(c *ui.Context) {
		diffH, editorH = d.ViewportHeight(), st.ViewportHeight()
		ui.Column(c).Fill().Children(func() {
			DiffView(c, d).Grow(1).FillWidth()
			View(c, st).Grow(1).FillWidth()
		})
	}, 800, 600)
	tt.Frame()
	if diffH != 300 || editorH != 300 {
		t.Fatalf("the view read heights %v and %v, want 300", diffH, editorH)
	}
	// The frame of the resize builds before it lays out: the view is
	// built again, with no other frame asked for.
	tt.SetSize(800, 400)
	if d.ViewportHeight() != 200 || diffH != 200 || editorH != 200 {
		t.Fatalf("after a resize the view read heights %v and %v, the diff's %v, want 200", diffH, editorH, d.ViewportHeight())
	}
}

// splitDiff returns a split diff of lines, long ones when long, whose
// line 20 changed.
func splitDiff(long bool) *DiffState {
	var a, b []string
	for i := range 40 {
		line := "line " + strconv.Itoa(i)
		if long {
			line += strings.Repeat(" word", 30)
		}
		a = append(a, line)
		if i == 20 {
			line = "changed " + line
		}
		b = append(b, line)
	}
	return &DiffState{Old: NewBuffer(strings.Join(a, "\n")), New: NewBuffer(strings.Join(b, "\n")), ExpandUnchanged: true}
}

// sideWidths returns the widths the sides of a split diff were drawn at,
// and the left of the new one.
func sideWidths(d *DiffState) (old, nw, newX float32) {
	return d.eds[0].r.W, d.eds[1].r.W, d.eds[1].r.X
}

// A split diff's sides share its width by SplitPercent, as two halves
// with none, the line between them taking a DIP.
func TestDiffSplitShare(t *testing.T) {
	d := splitDiff(false)
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 901, 400)
	tt.Frame()
	if a, b, x := sideWidths(d); a != 450 || b != 450 || x != 451 {
		t.Fatalf("with no share, the sides are %v and %v wide, the new one at %v, want halves", a, b, x)
	}
	if tt.FramePending() {
		t.Fatal("the halves ask for frames")
	}
	d.SplitPercent = 30
	tt.Frame()
	if a, b, x := sideWidths(d); a != 270 || b != 630 || x != 271 {
		t.Fatalf("at 30%%, the sides are %v and %v wide, the new one at %v", a, b, x)
	}
	// They keep the share as the view grows and shrinks.
	tt.SetSize(1201, 400)
	tt.Frame()
	if a, b, _ := sideWidths(d); a != 360 || b != 840 {
		t.Fatalf("at 30%% of a wider view, the sides are %v and %v wide", a, b)
	}
	if tt.FramePending() {
		t.Fatal("the share asks for frames")
	}
}

// The pointer drags the line between the sides, from a few DIPs either
// side of it, which focuses it; SplitPercent follows.
func TestDiffSplitDrag(t *testing.T) {
	d := splitDiff(false)
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 901, 400)
	tt.Frame()
	x, y := float32(450.5), float32(200)
	tt.Press(x, y)
	if !tt.Focused("Divider") {
		t.Fatal("a press on the line does not focus it")
	}
	if tt.Cursor() != ui.CursorResizeEW {
		t.Fatalf("the cursor over the line is %v", tt.Cursor())
	}
	tt.Move(x+90, y)
	tt.Release(x+90, y)
	if a, b, nx := sideWidths(d); a != 540 || b != 360 || nx != 541 {
		t.Fatalf("dragged 90 DIPs right, the sides are %v and %v wide, the new one at %v", a, b, nx)
	}
	if d.SplitPercent != 60 {
		t.Fatalf("dragged to 540 of 900 DIPs, SplitPercent is %v", d.SplitPercent)
	}
	// From over the old side as from over the new, and with the
	// release's own move.
	tt.Press(538, y)
	tt.Move(448, y)
	tt.Release(358, y)
	if a, _, _ := sideWidths(d); a != 360 || d.SplitPercent != 40 {
		t.Fatalf("dragged 180 DIPs left, the old side is %v wide, at %v%%", a, d.SplitPercent)
	}
	// A press without a move keeps the share, as the editors' text.
	tt.Press(360.5, y)
	tt.Release(360.5, y)
	if d.SplitPercent != 40 || d.Old.String() != splitDiff(false).Old.String() {
		t.Fatalf("a press moved the line to %v%%", d.SplitPercent)
	}
}

// Tab goes from the old side to the line between the sides and on to the
// new side, and the arrows move the line.
func TestDiffSplitKeys(t *testing.T) {
	d := splitDiff(false)
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 901, 400)
	tt.Frame()
	tt.ClickAt(200, 10)
	if d.ActiveSide() != DiffOld {
		t.Fatal("a click on the old side does not give it the keyboard")
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("Divider") {
		t.Fatal("Tab from the old side does not focus the line")
	}
	// By 10 DIPs, as ui.Split's; the share is a percentage, which the
	// sides' widths come back from as near as floats are.
	near := func(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-3 }
	tt.Key(0, ui.KeyRight)
	if a, b, _ := sideWidths(d); !near(a, 460) || !near(b, 440) {
		t.Fatalf("Right on the line: the sides are %v and %v wide", a, b)
	}
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyLeft)
	if a, _, _ := sideWidths(d); !near(a, 440) || !near(d.SplitPercent, 440*100/float32(900)) {
		t.Fatalf("Left twice: the old side is %v wide, at %v%%", a, d.SplitPercent)
	}
	tt.Key(0, ui.KeyTab)
	if !tt.Focused("New text") {
		t.Fatal("Tab from the line does not go to the new side")
	}
}

// Neither side is dragged narrower than its gutter and 12 columns, and a
// view too narrow for both shows halves, keeping the share for when it is
// wide enough again.
func TestDiffSplitMinimum(t *testing.T) {
	d := splitDiff(false)
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 901, 400)
	tt.Frame()
	ed := &d.eds[0]
	least := ed.textX + diffMinColumns*ed.view.charWidth + ed.trackWidth() + 4
	if least < 100 {
		t.Fatalf("a side keeps only %v DIPs", least)
	}
	tt.Press(450.5, 200)
	tt.Move(0, 200)
	tt.Release(0, 200)
	if a, b, _ := sideWidths(d); a != least || b != 900-least {
		t.Fatalf("dragged to the left edge, the sides are %v and %v wide, want the old one %v", a, b, least)
	}
	tt.Press(least+0.5, 200)
	tt.Move(900, 200)
	tt.Release(900, 200)
	if a, b, _ := sideWidths(d); b != least || a != 900-least {
		t.Fatalf("dragged to the right edge, the sides are %v and %v wide, want the new one %v", a, b, least)
	}
	// The arrows stop there too.
	tt.Key(0, ui.KeyRight)
	if _, b, _ := sideWidths(d); b != least {
		t.Fatalf("Right at the edge: the new side is %v wide", b)
	}
	// A share the app sets past them shows at them.
	d.SplitPercent = 1
	tt.Frame()
	if a, _, _ := sideWidths(d); a != least {
		t.Fatalf("at 1%%, the old side is %v wide", a)
	}
	// Too narrow for both: halves, the share kept.
	tt.SetSize(201, 400)
	tt.Frame()
	if a, b, x := sideWidths(d); a != 100 || b != 100 || x != 101 {
		t.Fatalf("in 201 DIPs, the sides are %v and %v wide, the new one at %v", a, b, x)
	}
	tt.Press(100.5, 200)
	tt.Move(150, 200)
	tt.Release(150, 200)
	if d.SplitPercent != 1 {
		t.Fatalf("a drag where the sides do not fit set the share to %v", d.SplitPercent)
	}
	tt.SetSize(901, 400)
	tt.Frame()
	if a, _, _ := sideWidths(d); a != least {
		t.Fatalf("wide again, the old side is %v wide", a)
	}
}

// A hunk's header in a split diff keeps to the old side, cut at the line
// between the sides wherever that is dragged, though its bar spans both;
// a unified diff shows it whole.
func TestDiffSplitHeaderKeepsToOldSide(t *testing.T) {
	const header = "@@ -21,6 +21,7 @@ import jakarta.servlet.AsyncContext;"
	files, err := ParsePatch("--- a/x\n+++ b/x\n" + header + "\n a\n b\n c\n+d\n e\n f\n g\n")
	if err != nil {
		t.Fatal(err)
	}
	// headerBox returns the box of the header's text, not that of the
	// separator labeled with it.
	headerBox := func(tt *ui.Tester) ui.Rect {
		t.Helper()
		a, ok := tt.AccessByRole(ui.RoleText, header)
		if !ok {
			t.Fatalf("no %q in %q", header, tt.Texts())
		}
		return a.Bounds
	}
	d := &DiffState{}
	d.SetPatch(files[0])
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 901, 400)
	tt.Frame()
	tt.Press(450.5, 200)
	tt.Move(0, 200)
	tt.Release(0, 200)
	tt.Frame()
	old, _, _ := sideWidths(d)
	if old > 200 {
		t.Fatalf("dragged to the left edge, the old side is %v wide", old)
	}
	if r := headerBox(tt); r.X+r.W > old {
		t.Fatalf("the header's text spans %v to %v, past the old side's right edge at %v", r.X, r.X+r.W, old)
	}

	// Unified, nothing cuts it: it is as wide as in a view twice as wide.
	d.Layout = DiffUnified
	tt.Frame()
	whole := headerBox(tt)
	tt.SetSize(1801, 400)
	tt.Frame()
	if r := headerBox(tt); whole.W != r.W || whole.W <= old {
		t.Fatalf("unified, the header's text is %v wide, %v in a wider view", whole.W, r.W)
	}
}

// Sides of different widths keep scrolling together and their lines
// aligned, wrapped too, each wrapping at its own width.
func TestDiffSplitAligned(t *testing.T) {
	for _, c := range []struct{ wrap, collapse bool }{{false, false}, {true, false}, {true, true}} {
		wrap := c.wrap
		d := splitDiff(true)
		d.Wrap, d.SplitPercent, d.ExpandUnchanged = wrap, 30, !c.collapse
		d.Annotations = []DiffAnnotation{{Side: DiffNew, Line: 22, Height: 40}}
		d.RenderAnnotation = func(c *ui.Context, a DiffAnnotation) { ui.Text(c, "note") }
		tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 901, 400)
		tt.Frame()
		tt.Press(270.5, 200)
		tt.Move(200, 200)
		tt.Release(200, 200)
		tt.Frame()
		if tt.FramePending() {
			t.Fatalf("%+v: the sides ask for frames", c)
		}
		ha, hb := &d.eds[0].view.heights, &d.eds[1].view.heights
		if ha.total() != hb.total() {
			t.Fatalf("%+v: heights %v and %v", c, ha.total(), hb.total())
		}
		lines := []int{0, 5, 19, 21, 22, 23, 39}
		if c.collapse {
			// The context around the change, and the separators' tops.
			lines = []int{17, 18, 19, 21, 22, 23, 24, 39}
		}
		for _, l := range lines {
			if ga, gb := rowsTop(ha, l), rowsTop(hb, l); ga != gb {
				t.Fatalf("%+v: old line %d at %v, new line at %v", c, l, ga, gb)
			}
		}
		if wrap && ha.wraps.rows(19) <= hb.wraps.rows(19) {
			t.Fatalf("the narrow old side wraps line 19 in %d rows, the new side in %d", ha.wraps.rows(19), hb.wraps.rows(19))
		}
		if c.collapse {
			continue
		}
		tt.Scroll(600, 200, 0, 120)
		tt.Frame()
		if d.eds[0].scrollY != 120 || d.eds[1].scrollY != 120 {
			t.Fatalf("%+v: scrolled the sides to %v and %v", c, d.eds[0].scrollY, d.eds[1].scrollY)
		}
		if !wrap {
			// Sideways the sides scroll together, as before, each as far
			// as its own lines run past it: the narrow old side further.
			old, nw := &d.eds[0], &d.eds[1]
			tt.Scroll(600, 200, 5000, 0)
			tt.Frame()
			if m := nw.maxScrollX(); m <= 0 || nw.scrollX != m || old.scrollX != m {
				t.Fatalf("over the new side, scrolled sideways to %v and %v, the new side's end %v", old.scrollX, nw.scrollX, m)
			}
			tt.Scroll(100, 200, 5000, 0)
			tt.Frame()
			if m := old.maxScrollX(); m <= nw.maxScrollX() || old.scrollX != m || nw.scrollX != nw.maxScrollX() {
				t.Fatalf("over the old side, scrolled sideways to %v of %v, the new side to %v of %v", old.scrollX, m, nw.scrollX, nw.maxScrollX())
			}
		}
	}
}
