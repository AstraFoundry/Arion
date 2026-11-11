package editor

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestCodeEditorParts builds the find bar, the go to line bar, the
// completions and the hover with the app's render functions, which
// replace the editor's own and use their handles.
func TestCodeEditorParts(t *testing.T) {
	ct := newEditorTest(t, "", "|cat dog cat\ncat")
	st := ct.st
	st.RenderFindBar = func(c *ui.Context, bar ui.Element, f *FindBar) {
		bar.Row().Children(func() {
			f.QueryInput(ui.TextInputBase(c, f.Query()).Label("Search").Width(100))
			n, _ := f.Matches()
			ui.Text(c, "found "+strconv.Itoa(n))
			if ui.Button(c, "Onward").Clicked() {
				f.Next()
			}
			if ui.Button(c, "Every").Clicked() {
				f.SelectAll()
			}
		})
	}
	st.Find("cat")
	ct.tt.Frame()
	if !ct.tt.HasText("found 3") || ct.tt.HasText("Match case") {
		t.Fatalf("custom find bar not built in place of the default: %q", ct.tt.Texts())
	}
	if err := ct.tt.Click("Onward"); err != nil {
		t.Fatal(err)
	}
	if _, cur := st.FindMatches(); cur != 1 {
		t.Fatalf("current match %d after Onward", cur)
	}
	if err := ct.tt.Click("Every"); err != nil {
		t.Fatal(err)
	}
	ct.want("[cat] dog [cat]\n[cat]")

	// Going to a line with a bar of the app's.
	ct = newEditorTest(t, "", "|a\nb\nc\nd")
	ct.st.RenderGoToLine = func(c *ui.Context, bar ui.Element, g *GoToLineBar) {
		bar.Children(func() {
			g.Input(ui.TextInputBase(c, g.Text()).Label("Line").Width(80))
			ui.Text(c, "of "+strconv.Itoa(g.LineCount()))
		})
	}
	ct.run(GoToLine)
	if !ct.tt.HasText("of 4") {
		t.Fatalf("custom go to line bar not built: %q", ct.tt.Texts())
	}
	ct.tt.Type("3")
	ct.key(0, ui.KeyEnter)
	ct.want("a\nb\n|c\nd")

	// Completions in a list of the app's, chosen by a click.
	ct = newEditorTest(t, "go", "|")
	ct.st.OnCompletion = func(r CompletionRequest) {
		ct.st.ShowCompletions(r.Offset-len(r.Prefix), []Completion{{Label: "alpha"}, {Label: "alpine"}, {Label: "beta"}})
	}
	ct.st.RenderCompletions = func(c *ui.Context, panel ui.Element, m *CompletionMenu) {
		panel.Children(func() {
			for i, it := range m.Items() {
				m.Item(i).Children(func() { ui.Text(c, "→ "+it.Label) })
			}
		})
	}
	ct.typ("al")
	if !ct.tt.HasText("→ alpine") || ct.tt.HasText("→ beta") {
		t.Fatalf("custom completions: %q", ct.tt.Texts())
	}
	if err := ct.tt.Click("→ alpine"); err != nil {
		t.Fatal(err)
	}
	ct.want("alpine|")

	// A hover of the app's, wrapping the default.
	ct = newEditorTest(t, "go", "|x := 1")
	ct.st.RenderHover = func(c *ui.Context, panel ui.Element, h *HoverCard) {
		DefaultHover(c, panel, h)
		panel.Background(ui.RGB(255, 255, 200))
	}
	ct.st.ShowHover(0, 1, "x is an int", nil)
	ct.tt.Frame()
	if !ct.tt.HasText("x is an int") {
		t.Fatalf("hover not built: %q", ct.tt.Texts())
	}
}

// TestCodeDiffParts builds the separators, the actions on changes and
// the gutter's button with the app's render functions, and checks the
// default separators expand from their buttons.
func TestCodeDiffParts(t *testing.T) {
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = "line " + strconv.Itoa(i)
	}
	old := strings.Join(lines, "\n")
	lines[30] = "changed"
	nw := strings.Join(lines, "\n")

	// The default separators: before the change, and after it.
	d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw)}
	tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 800, 600)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Expand 27 unmodified lines") || !tt.HasText("Expand 26 unmodified lines") {
		t.Fatalf("default separators: %q", tt.Texts())
	}
	if err := tt.Click("Expand 27 unmodified lines"); err != nil {
		t.Fatal(err)
	}
	// Gone with the frames the click brings, not a frame later.
	if tt.HasText("Expand 27 unmodified lines") {
		t.Fatal("the separator stayed after expanding its lines")
	}

	// The app's.
	var commented []DiffLineRange
	d = &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), HunkActions: true, Editable: true,
		OnGutterButton: func(r DiffLineRange) { commented = append(commented, r) }}
	d.RenderSeparator = func(c *ui.Context, sep ui.Element, s *DiffSeparator) {
		sep.Children(func() {
			if ui.Button(c, "Show "+strconv.Itoa(s.Lines)).Clicked() {
				s.ExpandAll()
			}
		})
	}
	d.RenderHunkActions = func(c *ui.Context, bar ui.Element, a *DiffHunkActions) {
		bar.Children(func() {
			if ui.Button(c, "Undo change "+strconv.Itoa(a.Index)).Clicked() {
				a.Revert()
			}
		})
	}
	d.RenderGutterButton = func(c *ui.Context, btn ui.Element, line DiffLineRange) {
		btn.Label("Comment")
	}
	tt = ui.NewTester(func(c *ui.Context) { DiffView(c, d).Fill() }, 800, 600)
	tt.Frame()
	tt.Frame()
	if !tt.HasText("Show 26") || tt.HasText("Expand 26 unmodified lines") {
		t.Fatalf("custom separators: %q", tt.Texts())
	}
	// The actions show on the change under the pointer.
	ed := &d.eds[1]
	tt.Move(600, float32(ed.view.heights.geometry(30).rows-ed.scrollY)+ed.view.lineHeight/2)
	tt.Frame()
	if err := tt.Click("Undo change 0"); err != nil {
		t.Fatalf("%v: %q", err, tt.Texts())
	}
	tt.Frame()
	if d.New.String() != old || len(d.Hunks()) != 0 {
		t.Fatal("the custom action did not revert the change")
	}
	// The gutter's button, on the line under the pointer, of the old side
	// on the left.
	y := float32(ed.view.heights.geometry(28).rows-ed.scrollY) + ed.view.lineHeight/2
	tt.Move(ed.gutterW/2, y)
	tt.Frame()
	if err := tt.Click("Comment"); err != nil {
		t.Fatalf("%v: %q", err, tt.Texts())
	}
	if len(commented) != 1 || commented[0] != (DiffLineRange{Side: DiffOld, Start: 28, End: 28}) {
		t.Fatalf("gutter button reported %+v", commented)
	}
}

// TestCodeEditorHoverLeave checks that the pointer passing over words on
// its way out of the editor asks for no hover, and that leaving closes
// the hover it asked for, as in examples/code-editor with its sidebar.
func TestCodeEditorHoverLeave(t *testing.T) {
	st := &State{Buffer: NewBuffer("// Package server serves greetings.\nfunc main() {}\n")}
	var asked []int
	st.OnHover = func(r HoverRequest) {
		asked = append(asked, r.Offset)
		st.ShowHover(r.Offset, r.Offset+1, "word", nil)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Box(c).Width(200).Label("Sidebar")
			View(c, st).Grow(1)
		})
	}, 800, 400)
	tt.Frame()
	rest := func() {
		tt.Advance(time.Hour)
	}
	// Over a word, then out to the sidebar before resting: no hover.
	tt.Move(400, 10)
	tt.Move(100, 10)
	rest()
	if len(asked) != 0 || st.ed.hover.shown {
		t.Fatalf("hover asked for %v after the pointer left", asked)
	}
	// Resting on a word asks; leaving closes the hover.
	tt.Move(400, 10)
	rest()
	if len(asked) != 1 || !st.ed.hover.shown {
		t.Fatalf("no hover after resting: asked %v", asked)
	}
	tt.Move(100, 10)
	tt.Frame()
	if !st.ed.hover.shown {
		t.Fatal("the hover went at once as the pointer left the editor, not after a grace")
	}
	graceOver := func() {
		tt.Advance(hoverHideDelay)
	}
	graceOver()
	if st.ed.hover.shown {
		t.Fatal("the hover stayed after the pointer left the editor")
	}
	// A hover asked for from the keyboard stays.
	st.Run(ShowHover)
	tt.Frame()
	if !st.ed.hover.shown {
		t.Fatal("the keyboard's hover closed with the pointer outside")
	}
}

// TestCodeEditorHoverInteractive checks that the pointer can go from the
// text into its hover, which stays, and select and copy its text, and
// that Escape closes it and gives the keyboard back to the editor.
func TestCodeEditorHoverInteractive(t *testing.T) {
	st := &State{Buffer: NewBuffer("first line\nsecond line\nfoo := bar()\n")}
	st.OnHover = func(r HoverRequest) {
		st.ShowHover(r.Offset, r.Offset+3, "func bar() int", nil)
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 600, 400)
	tt.Frame()
	ed := &st.ed
	y := float32(ed.view.heights.geometry(2).rows-ed.scrollY) + ed.view.lineHeight/2
	tt.Move(ed.textX+80, y)
	tt.Advance(time.Hour)
	r, ok := tt.Find("func bar() int")
	if !ok {
		t.Fatalf("no hover: %q", tt.Texts())
	}
	// On the way up, the pointer crosses the line above, then rests in
	// the hover past the grace.
	tt.Move(ed.textX+80, y-ed.view.lineHeight)
	tt.Frame()
	tt.Move(r.X+4, r.Y+r.H/2)
	tt.Frame()
	tt.Advance(hoverHideDelay)
	if !ed.hover.shown {
		t.Fatal("the hover went as the pointer went into it")
	}
	// Its text selects and copies.
	tt.Press(r.X+1, r.Y+r.H/2)
	tt.Move(r.X+r.W-1, r.Y+r.H/2)
	tt.Release(r.X+r.W-1, r.Y+r.H/2)
	tt.Key(ui.Cmd, ui.KeyC)
	if got := tt.Clipboard(); got != "func bar() int" {
		t.Fatalf("copied %q from the hover", got)
	}
	// Escape closes it, and the editor has the keyboard again.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if ed.hover.shown || !st.Focused() {
		t.Fatalf("after Escape: hover shown %v, editor focused %v", ed.hover.shown, st.Focused())
	}
}

// TestDiffSeparatorsKeepRoomOnlyForScrollbars: a diff that fits its
// content shows no scroll bar, so its separators run to the viewer's right
// edge; one that scrolls keeps the room of the scroll bar's track.
func TestDiffSeparatorsKeepRoomOnlyForScrollbars(t *testing.T) {
	lines := make([]string, 60)
	for i := range lines {
		lines[i] = "line " + strconv.Itoa(i)
	}
	old := strings.Join(lines, "\n")
	lines[30] = "changed"
	nw := strings.Join(lines, "\n")

	for _, tc := range []struct {
		name string
		fit  bool
		want float32 // where the separators end, from the viewer's left edge
	}{
		{"fit content", true, 800},
		{"scrolling", false, 800 - scrollbarWidth},
	} {
		d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), FitContent: tc.fit}
		var right float32
		d.RenderSeparator = func(c *ui.Context, sep ui.Element, s *DiffSeparator) {
			// A frame's bounds are the last frame's, which the second Frame reads.
			b := sep.Bounds()
			right = b.X + b.W
			sep.Children(func() { ui.Text(c, "separator") })
		}
		tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).FillWidth() }, 800, 600)
		tt.Frame()
		tt.Frame()
		if right != tc.want {
			t.Errorf("%s: separators end at %v, want %v", tc.name, right, tc.want)
		}
	}
}

// The sides of a split diff wrap their lines at the same width, whether
// or not it fits its content, though only the new side draws the scroll
// bar of a viewer that scrolls.
func TestDiffSplitSidesWrapAlike(t *testing.T) {
	long := strings.Repeat("word ", 80)
	old := "a\n" + long + "\nb\nc"
	nw := "a\n" + long + "\nB\nc"
	for _, fit := range []bool{false, true} {
		d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), Layout: DiffSplit, Wrap: true, FitContent: fit}
		tt := ui.NewTester(func(c *ui.Context) { DiffView(c, d).FillWidth() }, 800, 600)
		tt.Frame()
		tt.Frame()
		if o, n := d.eds[0].view.wrapWidth, d.eds[1].view.wrapWidth; o != n || o <= 0 {
			t.Errorf("FitContent %v: the old side wraps at %v, the new one at %v", fit, o, n)
		}
	}
}

// A diff that fits its content is as high as its lines with the unchanged
// ones folded before it is first drawn: below the view of a scroll
// container, it would else stand as high as all its lines until scrolled
// to, then shrink, and the container's end with it, under the user.
func TestDiffFitContentFoldsBeforeDrawn(t *testing.T) {
	var lines []string
	for i := range 400 {
		lines = append(lines, "line "+strconv.Itoa(i))
	}
	old := strings.Join(lines, "\n")
	lines[200] = "changed"
	nw := strings.Join(lines, "\n")
	for _, layout := range []DiffLayout{DiffUnified, DiffSplit} {
		d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), Layout: layout, FitContent: true}
		var h float32
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Scroll(c).Fill().Children(func() {
				ui.Box(c).Height(5000)
				// A frame's bounds are the last frame's.
				h = DiffView(c, d).FillWidth().Bounds().H
			})
		}, 800, 600)
		tt.Frame()
		tt.Frame()
		tt.Frame()
		if max := float32(40 * d.eds[1].view.lineHeight); h <= 0 || h > max {
			t.Errorf("layout %v: the diff not yet drawn is %v high, want at most %v", layout, h, max)
		}
	}
}

// A long diff that fits its content builds the separators of the folded
// lines in view of the scroll container it is in, not all of its own.
func TestDiffFitContentBuildsSeparatorsInView(t *testing.T) {
	var lines []string
	for i := range 20000 {
		lines = append(lines, "line "+strconv.Itoa(i))
	}
	old := strings.Join(lines, "\n")
	for i := 0; i < len(lines); i += 50 {
		lines[i] = "changed " + strconv.Itoa(i)
	}
	nw := strings.Join(lines, "\n")
	for _, layout := range []DiffLayout{DiffUnified, DiffSplit} {
		built := 0
		d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), Layout: layout, FitContent: true}
		d.RenderSeparator = func(c *ui.Context, sep ui.Element, s *DiffSeparator) {
			built++
			DefaultSeparator(c, sep, s)
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Scroll(c).Fill().Children(func() { DiffView(c, d).FillWidth() })
		}, 800, 600)
		tt.Frame()
		tt.Frame()
		built = 0
		// After a pause, as the first frame of a scroll.
		tt.Advance(time.Second)
		tt.Scroll(400, 300, 0, 200)
		tt.Frame()
		if built == 0 || built > 100 {
			t.Errorf("layout %v: a frame built %d of the diff's 400 separators", layout, built)
		}
	}
}

// A long diff that fits its content builds the separators that come into
// view as its scroll container jumps further than it built around what it
// showed last, in one wheel's turn or a scroll bar's drag.
func TestDiffFitContentBuildsSeparatorsScrolledTo(t *testing.T) {
	var lines []string
	for i := range 20000 {
		lines = append(lines, "line "+strconv.Itoa(i))
	}
	old := strings.Join(lines, "\n")
	for i := 0; i < len(lines); i += 50 {
		lines[i] = "changed " + strconv.Itoa(i)
	}
	nw := strings.Join(lines, "\n")
	for _, layout := range []DiffLayout{DiffUnified, DiffSplit} {
		d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), Layout: layout, FitContent: true}
		var labels []string
		d.RenderSeparator = func(c *ui.Context, sep ui.Element, s *DiffSeparator) {
			l := "separator " + strconv.Itoa(s.a)
			labels = append(labels, l)
			sep.Children(func() { ui.Text(c, l) })
		}
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Scroll(c).Fill().Children(func() { DiffView(c, d).FillWidth() })
		}, 800, 600)
		tt.Frame()
		tt.Frame()
		// No frame but those the scroll asks for, as in a window.
		tt.Scroll(400, 300, 0, 3000)
		shown := 0
		for _, l := range labels {
			if r, ok := tt.Find(l); ok && r.Y+r.H > 0 && r.Y < 600 {
				shown++
			}
		}
		if shown == 0 {
			t.Errorf("layout %v: no separator shown in view after scrolling 3000", layout)
		}
	}
}

// A long diff that fits its content measures the wraps of the lines it
// shows, not of those it folds away, which it measures once they show:
// a window resized through widths would else measure every line at each.
func TestDiffFitContentMeasuresLinesShown(t *testing.T) {
	var lines []string
	for i := range 20000 {
		lines = append(lines, "line "+strconv.Itoa(i)+" "+strings.Repeat("word ", i%40))
	}
	old := strings.Join(lines, "\n")
	lines[10000] = "changed"
	nw := strings.Join(lines, "\n")
	d := &DiffState{Old: NewBuffer(old), New: NewBuffer(nw), Wrap: true, FitContent: true}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Children(func() { DiffView(c, d).FillWidth() })
	}, 800, 600)
	tt.Frame()
	tt.Frame()
	v := &d.eds[1].view
	if known := v.heights.wraps.known; known == 0 || known > 1000 {
		t.Errorf("%d of the %d lines' wraps measured, those shown being a few", known, v.text.LineCount())
	}
	d.ExpandUnchanged = true
	tt.Frame()
	tt.Frame()
	if known := v.heights.wraps.known; known < v.text.LineCount() {
		t.Errorf("all lines shown, %d of their %d wraps measured", known, v.text.LineCount())
	}
}

// HoverDelay sets how long the pointer rests before the hover is asked.
func TestCodeEditorHoverDelay(t *testing.T) {
	st := &State{Buffer: NewBuffer("SELECT 1\n"), HoverDelay: 2 * time.Second}
	asked := 0
	st.OnHover = func(r HoverRequest) { asked++ }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill() }, 600, 300)
	tt.Frame()
	ed := &st.ed
	tt.Move(ed.textX+ed.view.charWidth*2, float32(-ed.scrollY)+ed.view.lineHeight/2)
	tt.Advance(time.Second)
	tt.Frame()
	if asked != 0 {
		t.Fatal("asked before the delay")
	}
	tt.Advance(1500 * time.Millisecond)
	tt.Frame()
	if asked != 1 {
		t.Fatalf("asked %d times after the delay", asked)
	}
}

// With NoPointerHover the resting pointer asks for no hover, nor shows a
// diagnostic's; ShowHover, from the keys, still asks.
func TestCodeEditorNoPointerHover(t *testing.T) {
	st := &State{Buffer: NewBuffer("SELECT 1\n"), NoPointerHover: true}
	asked := 0
	st.OnHover = func(r HoverRequest) { asked++ }
	tt := ui.NewTester(func(c *ui.Context) { View(c, st).Fill().AutoFocus() }, 600, 300)
	tt.Frame()
	st.SetDiagnostics(st.Buffer.Version(), []Diagnostic{{Start: 0, End: 6, Message: "a problem"}})
	ed := &st.ed
	tt.Move(ed.textX+ed.view.charWidth*2, float32(-ed.scrollY)+ed.view.lineHeight/2)
	tt.Advance(time.Hour)
	tt.Frame()
	if asked != 0 || tt.HasText("a problem") {
		t.Fatalf("the resting pointer asked %d times, problem shown %v", asked, tt.HasText("a problem"))
	}
	st.Run(ShowHover)
	tt.Frame()
	if asked != 1 {
		t.Fatalf("ShowHover asked %d times", asked)
	}
}
