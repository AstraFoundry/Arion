package richtext

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
)

// proseTester builds an EditorBase of doc, 10 DIPs an em in theme th,
// filling a window 300 wide and h high, focused. Its view reads the
// paragraphs in view as it builds, as an app's gutter does, into the
// boxes returned.
func proseTester(t *testing.T, doc *Document, th *Theme, h int) (*ui.Tester, *[]ParagraphBox) {
	t.Helper()
	boxes := new([]ParagraphBox)
	tt := ui.NewTester(func(c *ui.Context) {
		*boxes = doc.VisibleParagraphs()
		EditorBase(c, doc, EditorOptions{Font: ui.Font{Size: 10}, Theme: th}).Fill()
	}, 300, h)
	tt.Press(150, float32(h)-2)
	tt.Release(150, float32(h)-2)
	return tt, boxes
}

// caretAt puts the caret at i and returns where the editor shows it.
func caretAt(t *testing.T, tt *ui.Tester, doc *Document, i int) ui.Rect {
	t.Helper()
	doc.SetSelection(i, i)
	tt.Frame()
	r, ok := tt.TextCaret()
	if !ok {
		t.Fatal("no caret")
	}
	return r
}

// para returns a body paragraph of text s.
func para(s string) Paragraph { return Paragraph{Runs: []Run{{Text: s}}} }

// starts returns the rune each paragraph of t starts at.
func starts(t Text) []int {
	var out []int
	for start := range t.Paragraphs() {
		out = append(out, start)
	}
	return out
}

// TestFirstLineIndent checks which paragraphs Theme.FirstLineIndent
// indents: a body paragraph after another, not the first after a
// heading, a centered paragraph, an empty one, a list or the start of a
// quotation, and that the caret shows where the text is painted.
func TestFirstLineIndent(t *testing.T) {
	text := TextOf(
		Paragraph{Style: ParagraphStyle{Heading: 2}, Runs: []Run{{Text: "Title"}}},
		para("After the heading"), // 1
		para("Indented"),          // 2
		Paragraph{Style: ParagraphStyle{Align: ui.Center}, Runs: []Run{{Text: "* * *"}}},
		para("After the break"),     // 4
		para(""),                    // 5
		para("After an empty line"), // 6
		Paragraph{Style: ParagraphStyle{Quote: true}, Runs: []Run{{Text: "Quoted"}}},       // 7
		Paragraph{Style: ParagraphStyle{Quote: true}, Runs: []Run{{Text: "Quoted again"}}}, // 8
		Paragraph{Style: ParagraphStyle{List: Bullets}, Runs: []Run{{Text: "An item"}}},    // 9
		para("After the list"), // 10
		Paragraph{Style: ParagraphStyle{Align: ui.End}, Runs: []Run{{Text: "Signed"}}}, // 11
		para("Indented too?"),  // 12
		para("Indented again"), // 13
	)
	indented := map[int]bool{2: true, 5: true, 8: true, 13: true}
	doc := NewDocument(text)
	tt, _ := proseTester(t, doc, &Theme{FirstLineIndent: 2}, 400)
	plain := NewDocument(text)
	pt, _ := proseTester(t, plain, nil, 400)
	for p, s := range starts(text) {
		if st := text.ParagraphStyleAt(s); st.Align != ui.Start || st.Heading > 0 {
			continue
		}
		got, want := caretAt(t, tt, doc, s), caretAt(t, pt, plain, s)
		if got.Y != want.Y || got.H != want.H {
			t.Errorf("paragraph %d: caret %+v, %+v without the indent: the indent changed its line", p, got, want)
		}
		shift := got.X - want.X
		switch {
		case indented[p] && shift != 20:
			t.Errorf("paragraph %d starts %v right of its unindented start, want 20", p, shift)
		case !indented[p] && shift != 0:
			t.Errorf("paragraph %d is indented by %v", p, shift)
		}
	}
}

// TestFirstLineIndentEditing moves the caret, clicks and types around the
// indent of a paragraph: the indent is no character.
func TestFirstLineIndentEditing(t *testing.T) {
	long := strings.Repeat("word ", 30)
	doc := NewDocument(TextOf(para("One"), para(long), para("Three")))
	tt, _ := proseTester(t, doc, &Theme{FirstLineIndent: 2}, 400)
	start := 4
	first := caretAt(t, tt, doc, start)
	// The second line of the paragraph wraps back to the edge.
	tt.Key(0, ui.KeyDown)
	if _, c := doc.Selection(); c <= start {
		t.Fatalf("down from the start went to %d", c)
	}
	tt.Key(0, ui.KeyHome)
	second, _ := tt.TextCaret()
	if first.X-second.X != 20 || second.Y <= first.Y {
		t.Errorf("first line at %+v, second at %+v: want the first 20 right", first, second)
	}
	// Left at the start goes to the end of the paragraph before, and
	// right comes back.
	caretAt(t, tt, doc, start)
	tt.Key(0, ui.KeyLeft)
	if _, c := doc.Selection(); c != 3 {
		t.Fatalf("left from the start went to %d, want 3", c)
	}
	tt.Key(0, ui.KeyRight)
	if _, c := doc.Selection(); c != start {
		t.Fatalf("right went to %d, want %d", c, start)
	}
	tt.Key(0, ui.KeyRight)
	if _, c := doc.Selection(); c != start+1 {
		t.Fatalf("right went to %d, want %d", c, start+1)
	}
	// A click in the indent puts the caret at the start.
	tt.ClickAt(5, first.Y+first.H/2)
	if a, c := doc.Selection(); a != start || c != start {
		t.Fatalf("a click in the indent selected %d–%d, want %d", a, c, start)
	}
	// Selecting the line from its end to Home selects from the start.
	tt.ClickAt(first.X+3, first.Y+first.H/2)
	tt.Key(ui.Shift, ui.KeyHome)
	if a, c := doc.Selection(); min(a, c) != start {
		t.Fatalf("shift-home selected %d–%d, want from %d", a, c, start)
	}
	// Typing at the start goes before the paragraph's text, after the
	// indent.
	caretAt(t, tt, doc, start)
	tt.Type("A")
	if got := doc.Text().String(); !strings.HasPrefix(got, "One\nAword") {
		t.Fatalf("typed into %q", got)
	}
	if r, _ := tt.TextCaret(); r.X <= first.X {
		t.Errorf("caret at %v after typing, the start at %v", r.X, first.X)
	}
	// An input method composes after the indent, its caret in the
	// composition.
	caretAt(t, tt, doc, start)
	tt.Compose("ni", 2)
	if r, _ := tt.TextCaret(); r.X <= first.X || r.Y != first.Y {
		t.Errorf("composing, the caret at %+v, the start at %+v", r, first)
	}
	tt.Compose("", 0)
	tt.Type("你")
	if got := doc.Text().String(); !strings.HasPrefix(got, "One\n你Aword") {
		t.Fatalf("composed into %q", got)
	}
	// Backspace at the start joins the paragraphs, as without an indent.
	caretAt(t, tt, doc, start)
	tt.Key(0, ui.KeyBackspace)
	if got := doc.Text().String(); !strings.HasPrefix(got, "One你Aword") {
		t.Fatalf("backspace left %q", got)
	}
}

// TestFirstLineIndentAttachment checks that an attachment starting an
// indented paragraph is drawn after the indent, not in it.
func TestFirstLineIndentAttachment(t *testing.T) {
	var drawn []ui.Rect
	at := &Attachment{Width: 30, Height: 8, Draw: func(p *ui.Painter, r ui.Rect) { drawn = append(drawn, r) }}
	text := TextOf(para("One"), Paragraph{Runs: []Run{{Text: string(ObjectReplacement), Attributes: Attributes{Attachment: at}}, {Text: " two"}}})
	doc := NewDocument(text)
	tt, _ := proseTester(t, doc, &Theme{FirstLineIndent: 2}, 200)
	drawn = nil
	tt.Frame()
	tt.Advance(600 * time.Millisecond) // a blink, which paints again
	if len(drawn) == 0 {
		t.Fatal("the attachment was not drawn")
	}
	for _, r := range drawn {
		if r.X != 20 || r.W != 30 {
			t.Fatalf("attachment drawn at %+v, want at x 20, 30 wide", r)
		}
	}
}

// TestParagraphSpacing checks that Theme.ParagraphSpacing goes between
// body paragraphs only, and that the caret and the paragraphs' boxes
// stay with the text painted.
func TestParagraphSpacing(t *testing.T) {
	text := TextOf(
		para("One"),
		para("Two"), // spaced
		Paragraph{Style: ParagraphStyle{Heading: 2}, Runs: []Run{{Text: "Heading"}}},
		para("Three"),
		Paragraph{Style: ParagraphStyle{List: Bullets}, Runs: []Run{{Text: "Item"}}},
		Paragraph{Style: ParagraphStyle{List: Bullets}, Runs: []Run{{Text: "Item"}}},
		para("Four"),
		Paragraph{Style: ParagraphStyle{Align: ui.Center}, Runs: []Run{{Text: "* * *"}}}, // spaced
		Paragraph{Style: ParagraphStyle{Quote: true}, Runs: []Run{{Text: "Quoted"}}},     // spaced
		Paragraph{Style: ParagraphStyle{CodeBlock: true}, Runs: []Run{{Text: "code"}}},
		para("Five"),
	)
	spaced := map[int]bool{1: true, 7: true, 8: true}
	doc := NewDocument(text)
	tt, boxes := proseTester(t, doc, &Theme{ParagraphSpacing: 1}, 400)
	plain := NewDocument(text)
	_, plainBoxes := proseTester(t, plain, nil, 400)
	if len(*boxes) != len(starts(text)) || len(*plainBoxes) != len(*boxes) {
		t.Fatalf("%d and %d paragraphs in view, want %d", len(*boxes), len(*plainBoxes), len(starts(text)))
	}
	added := float32(0)
	for p, b := range *boxes {
		pb := (*plainBoxes)[p]
		if spaced[p] {
			added += 10
		}
		if math.Abs(float64(b.Top-pb.Top-added)) > 0.01 || b.Height != pb.Height {
			t.Errorf("paragraph %d at %v, %v high, and %v, %v high without spacing: want %v lower", p, b.Top, b.Height, pb.Top, pb.Height, added)
		}
		if c := caretAt(t, tt, doc, b.Start); c.Y != b.Top {
			t.Errorf("paragraph %d: caret at %v, its box at %v", p, c.Y, b.Top)
		}
	}
	// A click in the space above a paragraph goes into it.
	two := (*boxes)[1]
	tt.ClickAt(100, two.Top-4)
	if _, c := doc.Selection(); c < two.Start || c > two.Start+3 {
		t.Errorf("a click above the second paragraph put the caret at %d", c)
	}
}

// TestParagraphSpacingQuoteBar checks that the bar of a quotation starts
// at its text, below the spacing above it.
func TestParagraphSpacingQuoteBar(t *testing.T) {
	bar := ui.RGB(255, 0, 0)
	text := TextOf(para("One"),
		Paragraph{Style: ParagraphStyle{Quote: true}, Runs: []Run{{Text: "Quoted"}}},
		Paragraph{Style: ParagraphStyle{Quote: true}, Runs: []Run{{Text: "Quoted again"}}})
	doc := NewDocument(text)
	tt, boxes := proseTester(t, doc, &Theme{ParagraphSpacing: 2, QuoteBar: bar, QuoteBarWidth: 4}, 200)
	img := tt.Image()
	red := func(y float32) bool {
		c := img.RGBAAt(2, int(y))
		return c.R > 200 && c.G < 50 && c.B < 50
	}
	q1, q2 := (*boxes)[1], (*boxes)[2]
	if red(q1.Top - 10) {
		t.Error("the quotation's bar goes up into the spacing above it")
	}
	if !red(q1.Top+q1.Height/2) || !red(q2.Top-10) || !red(q2.Top+q2.Height/2) {
		t.Error("the quotation's bar is not along its paragraphs and the spacing between them")
	}
}

// TestVisibleParagraphs checks that the paragraphs in view, as an app's
// view reads them while it builds, are where the editor paints them, as
// it scrolls, edits and resizes.
func TestVisibleParagraphs(t *testing.T) {
	paras := make([]Paragraph, 200)
	for i := range paras {
		paras[i] = para(fmt.Sprintf("Paragraph %d, long enough to wrap in a window this narrow at times.", i))
	}
	doc := NewDocument(TextOf(paras...))
	tt, boxes := proseTester(t, doc, &Theme{ParagraphSpacing: 0.5, FirstLineIndent: 1.5}, 400)
	doc.SetSelection(0, 0)
	tt.Frame()
	painted := func() []ParagraphBox {
		t.Helper()
		v := doc.shownIn.Value()
		if v == nil {
			t.Fatal("no editor shows the document")
		}
		return v.boxes(nil)
	}
	height := float32(400)
	check := func(what string) {
		t.Helper()
		got, want := *boxes, painted()
		if len(got) == 0 || !slices.Equal(got, want) {
			t.Fatalf("%s: the view read %v, the editor painted %v", what, got, want)
		}
		// The first box may start below the top by the spacing above it.
		if got[0].Top > 5 || got[len(got)-1].Top+got[len(got)-1].Height < height {
			t.Fatalf("%s: %v does not cover the view", what, got)
		}
		text := doc.Text()
		for k, b := range got {
			if s, _ := text.ParagraphRange(b.Start); s != b.Start || text.paraAt(b.Start) != b.Index {
				t.Fatalf("%s: box %+v starts no paragraph", what, b)
			}
			if k > 0 && (b.Index != got[k-1].Index+1 || b.Top < got[k-1].Top+got[k-1].Height) {
				t.Fatalf("%s: boxes %+v and %+v out of order", what, got[k-1], b)
			}
		}
	}
	check("first frame")
	if (*boxes)[0].Index != 0 || (*boxes)[0].Top != 0 {
		t.Fatalf("the first box %+v, want paragraph 0 at the top", (*boxes)[0])
	}
	// The caret's line is at the box's top.
	if c := caretAt(t, tt, doc, (*boxes)[2].Start); c.Y != (*boxes)[2].Top {
		t.Errorf("caret at %v, its paragraph's box at %v", c.Y, (*boxes)[2].Top)
	}
	before := slices.Clone(*boxes)
	tt.Scroll(150, 200, 0, 1000)
	check("scrolled")
	if (*boxes)[0].Index == before[0].Index {
		t.Fatalf("scrolled, the first paragraph in view is still %d", before[0].Index)
	}
	// The app moves the caret far down: the editor reveals it as it lays
	// out, and the view reads where.
	end := doc.Text().Len()
	doc.SetSelection(end, end)
	tt.Frame()
	check("revealed")
	if last := (*boxes)[len(*boxes)-1]; last.Index != 199 {
		t.Fatalf("revealing the end, the last paragraph in view is %d", last.Index)
	}
	// An edit above moves the paragraphs down.
	first := (*boxes)[1]
	doc.Edit(func(e *Edit) { e.ReplaceString(first.Start, first.Start, "New\n", Attributes{}) })
	tt.Frame()
	check("edited")
	if (*boxes)[1].Start != first.Start || (*boxes)[2].Start != first.Start+4 {
		t.Fatalf("after a paragraph inserted at %d, boxes %v", first.Start, (*boxes)[:3])
	}
	// Another width wraps the paragraphs otherwise: the frame that paints
	// them builds another, for the view to follow.
	tt.SetSize(200, 400)
	check("narrower")
	height = 250
	tt.SetSize(300, 250)
	check("shorter")
}

// TestVisibleParagraphsBeforeEditor checks the first frames: the view
// reads nothing before the editor showed the document, then the
// paragraphs in view, with no help from the app.
func TestVisibleParagraphsBeforeEditor(t *testing.T) {
	doc := NewDocument(TextOf(para("One"), para("Two")))
	if b := doc.VisibleParagraphs(); b != nil {
		t.Fatalf("no editor yet, and %v", b)
	}
	var reads [][]ParagraphBox
	tt := ui.NewTester(func(c *ui.Context) {
		reads = append(reads, doc.VisibleParagraphs())
		EditorBase(c, doc, EditorOptions{Font: ui.Font{Size: 10}}).Fill()
	}, 300, 200)
	if len(reads) < 2 || reads[0] != nil {
		t.Fatalf("reads %v, want nil then the paragraphs", reads)
	}
	if got := reads[len(reads)-1]; len(got) != 2 || got[0].Top != 0 || got[1].Top != got[0].Height {
		t.Fatalf("read %v", got)
	}
	// Frames that change nothing ask for none more.
	n := len(reads)
	tt.Frame()
	if len(reads) != n+1 || tt.FramePending() {
		t.Fatalf("%d frames for one", len(reads)-n)
	}
}

// ids returns the IDs of the paragraphs of t.
func ids(t Text) []string {
	var out []string
	for _, p := range t.Paragraphs() {
		out = append(out, p.Style.ID)
	}
	return out
}

// TestParagraphIDEdits checks where IDs go as paragraphs are split and
// joined, by the editor's keys, and that undo and redo bring them back.
func TestParagraphIDEdits(t *testing.T) {
	text := TextOf(
		Paragraph{Style: ParagraphStyle{ID: "a"}, Runs: []Run{{Text: "alpha"}}},
		Paragraph{Style: ParagraphStyle{ID: "b"}, Runs: []Run{{Text: "beta"}}},
		Paragraph{Style: ParagraphStyle{ID: "c", Heading: 2}, Runs: []Run{{Text: "gamma"}}},
	)
	steps := []struct {
		what   string
		caret  int
		key    ui.Key
		text   string
		ids    []string
		strung string
	}{
		{"Enter in the middle", 2, ui.KeyEnter, "", []string{"a", "", "b", "c"}, "al\npha\nbeta\ngamma"},
		{"Enter at the start", 7, ui.KeyEnter, "", []string{"a", "", "", "b", "c"}, "al\npha\n\nbeta\ngamma"},
		{"Enter at the end", 13, ui.KeyEnter, "", []string{"a", "", "", "b", "", "c"}, "al\npha\n\nbeta\n\ngamma"},
		{"Backspace joining", 3, ui.KeyBackspace, "", []string{"a", "", "b", "", "c"}, "alpha\n\nbeta\n\ngamma"},
		{"Backspace after an empty paragraph", 7, ui.KeyBackspace, "", []string{"a", "b", "", "c"}, "alpha\nbeta\n\ngamma"},
		{"Delete joining", 10, ui.KeyDelete, "", []string{"a", "b", "c"}, "alpha\nbeta\ngamma"},
	}
	doc := NewDocument(text)
	tt, _ := richTester(t, doc)
	var versions []Text
	for _, s := range steps {
		versions = append(versions, doc.Text())
		doc.SetSelection(s.caret, s.caret)
		tt.Frame()
		tt.Key(0, s.key)
		if got := doc.Text().String(); got != s.strung {
			t.Fatalf("%s: text %q, want %q", s.what, got, s.strung)
		}
		if got := ids(doc.Text()); !slices.Equal(got, s.ids) {
			t.Fatalf("%s: IDs %q, want %q", s.what, got, s.ids)
		}
	}
	for k := len(steps) - 1; k >= 0; k-- {
		tt.Key(ui.Cmd, ui.KeyZ)
		if !doc.Text().Equal(versions[k]) {
			t.Fatalf("undoing %s: IDs %q, want %q", steps[k].what, ids(doc.Text()), ids(versions[k]))
		}
	}
	for _, s := range steps {
		tt.Key(ui.Cmd|ui.Shift, ui.KeyZ)
		if got := ids(doc.Text()); !slices.Equal(got, s.ids) {
			t.Fatalf("redoing %s: IDs %q, want %q", s.what, got, s.ids)
		}
	}
}

// TestParagraphIDText checks IDs in texts: a newline in a run, a
// replacement whose text has IDs, JSON, formats and the clipboard.
func TestParagraphIDText(t *testing.T) {
	text := TextOf(
		Paragraph{Style: ParagraphStyle{ID: "a"}, Runs: []Run{{Text: "one\ntwo"}}},
		Paragraph{Style: ParagraphStyle{ID: "b"}, Runs: []Run{{Text: "three"}}},
	)
	if got := ids(text); !slices.Equal(got, []string{"a", "", "b"}) {
		t.Fatalf("TextOf made IDs %q", got)
	}
	// Text inserted, IDs and all, brings none.
	in := text.Replace(4, 4, text.Slice(0, text.Len()))
	if got := ids(in); !slices.Equal(got, []string{"a", "", "", "", "b"}) {
		t.Fatalf("inserting a copy made IDs %q", got)
	}
	// JSON keeps them.
	b, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	var back Text
	if err := json.Unmarshal(b, &back); err != nil || !back.Equal(text) {
		t.Fatalf("JSON %s read back as %q", b, ids(back))
	}
	// The clipboard does not, in its format or in HTML.
	clip := clipboardData(text)
	for _, f := range []transfer.Format{Format, transfer.HTML} {
		if data, _ := clip.Read(f); strings.Contains(string(data), `"a"`) || strings.Contains(string(data), `id=`) {
			t.Errorf("the clipboard's %s holds the IDs: %s", f, data)
		}
	}
	// Paragraphs alike but for their IDs are no mix of styles.
	doc := NewDocument(text)
	doc.SetSelection(0, text.Len())
	if f := doc.Formatting(); f.ParagraphsMixed || f.Paragraph.ID != "a" {
		t.Errorf("formatting %+v", f)
	}
}

// TestParagraphIDCopyPaste copies paragraphs with IDs and pastes them:
// the copies have none, and the paragraphs copied keep theirs.
func TestParagraphIDCopyPaste(t *testing.T) {
	doc := NewDocument(TextOf(
		Paragraph{Style: ParagraphStyle{ID: "a"}, Runs: []Run{{Text: "one"}}},
		Paragraph{Style: ParagraphStyle{ID: "b"}, Runs: []Run{{Text: "two"}}},
	))
	tt, _ := richTester(t, doc)
	doc.SetSelection(0, 7)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyC)
	end := doc.Text().Len()
	doc.SetSelection(end, end)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	tt.Key(ui.Cmd, ui.KeyV)
	if got := doc.Text().String(); got != "one\ntwo\none\ntwo" {
		t.Fatalf("pasted into %q", got)
	}
	if got := ids(doc.Text()); !slices.Equal(got, []string{"a", "b", "", ""}) {
		t.Fatalf("IDs %q after the paste", got)
	}
	// Cut and paste back: the text pasted is new.
	doc.SetSelection(0, 4)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyX)
	tt.Key(ui.Cmd, ui.KeyV)
	if got := ids(doc.Text()); got[0] != "" || !slices.Contains(got, "b") {
		t.Fatalf("IDs %q after cutting and pasting", got)
	}
}

// TestIdentifyParagraphs checks that a document naming paragraphs names
// those it has and those edits make, as part of their steps of undo.
func TestIdentifyParagraphs(t *testing.T) {
	doc := NewDocument(TextOf(Paragraph{Style: ParagraphStyle{ID: "keep"}, Runs: []Run{{Text: "one"}}}, para("two")))
	n := 0
	doc.IdentifyParagraphs(func() string { n++; return fmt.Sprint("n", n) })
	if got := ids(doc.Text()); !slices.Equal(got, []string{"keep", "n1"}) {
		t.Fatalf("IDs %q", got)
	}
	tt, _ := richTester(t, doc)
	doc.SetSelection(1, 1)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	after := ids(doc.Text())
	if !slices.Equal(after, []string{"keep", "n2", "n1"}) {
		t.Fatalf("IDs %q after Enter", after)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := ids(doc.Text()); !slices.Equal(got, []string{"keep", "n1"}) {
		t.Fatalf("IDs %q after undo", got)
	}
	doc.Redo()
	if got := ids(doc.Text()); !slices.Equal(got, after) {
		t.Fatalf("IDs %q after redo, want %q", got, after)
	}
	// Pasted paragraphs get new IDs, and a paragraph whose ID the app
	// takes away gets one too.
	doc.SetSelection(0, 0)
	doc.Insert(TextOf(Paragraph{Style: ParagraphStyle{ID: "keep"}, Runs: []Run{{Text: "x\ny\n"}}}))
	doc.Edit(func(e *Edit) { e.FormatParagraphs(0, 0, func(p *ParagraphStyle) { p.ID = "" }) })
	got := ids(doc.Text())
	seen := map[string]bool{}
	for _, id := range got {
		if id == "" || seen[id] {
			t.Fatalf("IDs %q: one missing or twice", got)
		}
		seen[id] = true
	}
	doc.SetText(TextOf(para("new"), para("text")))
	if got := ids(doc.Text()); slices.Contains(got, "") {
		t.Fatalf("IDs %q of a text set", got)
	}
}

// TestVisibleParagraphsGutter lines up a gutter beside an Editor, as the
// guide shows it: its marks are at the paragraphs' first lines, padding
// and scrolling included.
func TestVisibleParagraphsGutter(t *testing.T) {
	paras := make([]Paragraph, 60)
	for i := range paras {
		paras[i] = para(fmt.Sprintf("Paragraph %d", i))
	}
	doc := NewDocument(TextOf(paras...))
	var boxes []ParagraphBox
	var marks []ui.Rect
	tt := ui.NewTester(func(c *ui.Context) {
		boxes = doc.VisibleParagraphs()
		ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			ui.Box(c).Width(40).Clip().Draw(func(p *ui.Painter, r ui.Rect) {
				marks = marks[:0]
				for _, b := range boxes {
					mark := ui.Rect{X: r.X + 16, Y: r.Y + b.Top, W: 8, H: b.Height}
					marks = append(marks, mark)
					p.Fill(mark, ui.RGB(0, 0, 0), 2)
				}
			})
			Editor(c, doc, EditorOptions{Theme: &Theme{ParagraphSpacing: 1}}).Grow(1)
		})
	}, 400, 300)
	tt.ClickAt(200, 150)
	for _, scroll := range []float32{0, 230} {
		tt.Scroll(200, 150, 0, scroll)
		if len(marks) < 3 || len(marks) != len(boxes) {
			t.Fatalf("scrolled %v: marks %v of boxes %v", scroll, marks, boxes)
		}
		// Paragraphs whole in view, so that the caret does not scroll.
		for k := 1; k < 3; k++ {
			b, m := boxes[k], marks[k]
			doc.SetSelection(b.Start, b.Start)
			tt.Frame()
			if r, _ := tt.TextCaret(); r.Y != m.Y || marks[k] != m {
				t.Errorf("scrolled %v: paragraph %d's caret at %v, its mark at %v", scroll, b.Index, r.Y, m.Y)
			}
		}
	}
	if boxes[0].Index == 0 {
		t.Error("the editor did not scroll")
	}
}
