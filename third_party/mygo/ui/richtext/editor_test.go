package richtext

import (
	"bytes"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// richTester builds a RichTextEditor of doc filling a window, focused at
// the end of its text.
func richTester(t *testing.T, doc *Document) (*ui.Tester, *bool) {
	t.Helper()
	changed := new(bool)
	tt := ui.NewTester(func(c *ui.Context) {
		if Editor(c, doc).Fill().Changed() {
			*changed = true
		}
	}, 500, 400)
	tt.Press(250, 380)
	tt.Release(250, 380)
	tt.Key(ui.Cmd, ui.KeyDown)
	return tt, changed
}

// runsOf describes the runs of a text: their text, marked with what they
// are, as "plain|*bold*|_italic_|[link]".
func runsOf(at Text) string {
	var parts []string
	var last Attributes
	for i, r := range at.Runs(0, at.Len()) {
		if i > 0 && r.Attributes == last && len(parts) > 0 {
			// A run of the next paragraph, as the one before.
			parts[len(parts)-1] = joinRun(parts[len(parts)-1], r.Text)
			continue
		}
		last = r.Attributes
		s := r.Text
		a := r.Attributes
		if a.Bold() {
			s = "*" + s + "*"
		}
		if a.Italic {
			s = "_" + s + "_"
		}
		if a.Underline {
			s = "~" + s + "~"
		}
		if a.Code {
			s = "`" + s + "`"
		}
		if a.Link != "" {
			s = "[" + s + "]"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "|")
}

// joinRun puts text inside the marks of a run runsOf described.
func joinRun(run, text string) string {
	i := strings.LastIndexFunc(run, func(r rune) bool { return !strings.ContainsRune("*_~`]", r) })
	return run[:i+1] + text + run[i+1:]
}

// typeRunes types s a rune at a time, as a keyboard does.
func typeRunes(tt *ui.Tester, s string) {
	for _, r := range s {
		tt.Type(string(r))
	}
}

// The caret blinks by the frames' time, and shows again as it moves.
func TestRichTextCaretBlinks(t *testing.T) {
	const blink = 530 * time.Millisecond
	tt, _ := richTester(t, NewDocument(Text{}))
	tt.Type("hello")
	shown := bytes.Clone(tt.Image().Pix)
	tt.Advance(blink)
	if bytes.Equal(tt.Image().Pix, shown) {
		t.Fatal("the caret still shows a blink on")
	}
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyRight)
	if !bytes.Equal(tt.Image().Pix, shown) {
		t.Error("the caret does not show as it moved")
	}
}

func TestRichTextTyping(t *testing.T) {
	doc := NewDocument(Text{})
	tt, changed := richTester(t, doc)
	tt.Type("plain ")
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Type("bold")
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Type(" more")
	if got := runsOf(doc.Text()); got != "plain |*bold*| more" {
		t.Fatalf("runs %q", got)
	}
	if !*changed {
		t.Fatal("typing reports no change")
	}
	// Typing at the end of bold continues it; at its start, it does not.
	doc.SetSelection(10, 10)
	tt.Frame()
	tt.Type("er")
	doc.SetSelection(6, 6)
	tt.Frame()
	tt.Type("x")
	if got := runsOf(doc.Text()); got != "plain x|*bolder*| more" {
		t.Fatalf("runs %q", got)
	}
}

func TestRichTextLinkEdges(t *testing.T) {
	doc := NewDocument(TextOf(Paragraph{Runs: []Run{
		{Text: "see "}, {Text: "docs", Attributes: Attributes{Link: "https://x"}}, {Text: " now"},
	}}))
	tt, _ := richTester(t, doc)
	doc.SetSelection(8, 8)
	tt.Frame()
	tt.Type("!")
	doc.SetSelection(6, 6)
	tt.Frame()
	tt.Type("o")
	if got := runsOf(doc.Text()); got != "see |[doocs]|! now" {
		t.Fatalf("runs %q", got)
	}
}

func TestRichTextCaretFormatIsNoStep(t *testing.T) {
	doc := NewDocument(NewText("hello"))
	tt, _ := richTester(t, doc)
	tt.Key(ui.Cmd, ui.KeyB)
	if doc.CanUndo() {
		t.Fatal("⌘B at a caret made a step of undo")
	}
	if f := doc.Formatting(); f.Bold != FormatOn {
		t.Fatalf("typing attributes not bold: %+v", f)
	}
	// Moving the caret forgets them.
	tt.Key(0, ui.KeyLeft)
	if doc.Formatting().Bold != FormatOff {
		t.Fatal("moving the caret kept the typing attributes")
	}
}

func TestRichTextUndoRestoresAttributes(t *testing.T) {
	doc := NewDocument(TextOf(Paragraph{Runs: []Run{
		{Text: "a "}, {Text: "bold", Attributes: Attributes{Weight: 700}}, {Text: " word"},
	}}))
	tt, _ := richTester(t, doc)
	before := doc.Text()
	doc.SetSelection(2, 6)
	tt.Frame()
	tt.Key(0, ui.KeyBackspace)
	if got := doc.Text().String(); got != "a  word" {
		t.Fatalf("deleted to %q", got)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if !doc.Text().Equal(before) {
		t.Fatalf("undo made %q", runsOf(doc.Text()))
	}
	if a, c := doc.Selection(); a != 2 || c != 6 {
		t.Fatalf("undo selected %d–%d", a, c)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if got := doc.Text().String(); got != "a  word" {
		t.Fatalf("redo made %q", got)
	}
}

func TestRichTextFormatUndo(t *testing.T) {
	doc := NewDocument(NewText("one two"))
	tt, _ := richTester(t, doc)
	doc.SetSelection(4, 7)
	tt.Frame()
	tt.Key(ui.Cmd, ui.KeyI)
	tt.Key(ui.Cmd, ui.KeyU)
	if got := runsOf(doc.Text()); got != "one |~_two_~" {
		t.Fatalf("runs %q", got)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := runsOf(doc.Text()); got != "one |_two_" {
		t.Fatalf("after one undo %q", got)
	}
	if a, c := doc.Selection(); a != 4 || c != 7 {
		t.Fatalf("undo selected %d–%d", a, c)
	}
}

func TestRichTextLists(t *testing.T) {
	doc := NewDocument(Text{})
	tt, _ := richTester(t, doc)
	typeRunes(tt, "- milk")
	tt.Key(0, ui.KeyEnter)
	typeRunes(tt, "eggs")
	tt.Key(0, ui.KeyEnter)
	at := doc.Text()
	if got := at.String(); got != "milk\neggs\n" {
		t.Fatalf("text %q", got)
	}
	for i, want := range []ListKind{Bullets, Bullets, Bullets} {
		if got := at.ParagraphStyleAt(at.paraStart(i)).List; got != want {
			t.Fatalf("paragraph %d: list %d, want %d", i, got, want)
		}
	}
	// Enter on an empty item leaves the list.
	tt.Key(0, ui.KeyEnter)
	at = doc.Text()
	if at.String() != "milk\neggs\n" || at.ParagraphStyleAt(at.Len()).List != NoList {
		t.Fatalf("%q %+v", at.String(), at.ParagraphStyleAt(at.Len()))
	}
	// Tab indents an item, Backspace at its start outdents it, then takes
	// it out of the list.
	doc.SetSelection(5, 5)
	tt.Frame()
	tt.Key(0, ui.KeyTab)
	if got := doc.Text().ParagraphStyleAt(5).Indent; got != 1 {
		t.Fatalf("indent %d", got)
	}
	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyBackspace)
	if st := doc.Text().ParagraphStyleAt(5); st.List != NoList || st.Indent != 0 {
		t.Fatalf("style %+v", st)
	}
}

func TestRichTextHeadings(t *testing.T) {
	doc := NewDocument(Text{})
	tt, _ := richTester(t, doc)
	typeRunes(tt, "## Title")
	tt.Key(0, ui.KeyEnter)
	typeRunes(tt, "body")
	at := doc.Text()
	if at.String() != "Title\nbody" || at.ParagraphStyleAt(0).Heading != 2 || at.ParagraphStyleAt(7).Heading != 0 {
		t.Fatalf("%q %+v %+v", at.String(), at.ParagraphStyleAt(0), at.ParagraphStyleAt(7))
	}
	// Undo takes the shortcut back to what was typed.
	doc2 := NewDocument(Text{})
	tt2, _ := richTester(t, doc2)
	typeRunes(tt2, "# ")
	tt2.Key(ui.Cmd, ui.KeyZ)
	if got := doc2.Text(); got.String() != "# " || got.ParagraphStyleAt(0).Heading != 0 {
		t.Fatalf("undo of the shortcut: %q", got.String())
	}
}

func TestRichTextInlineShortcuts(t *testing.T) {
	doc := NewDocument(Text{})
	tt, _ := richTester(t, doc)
	typeRunes(tt, "a **big** and `x` then")
	if got := runsOf(doc.Text()); got != "a |*big*| and |`x`| then" {
		t.Fatalf("runs %q", got)
	}
}

func TestRichTextCompose(t *testing.T) {
	doc := NewDocument(NewText("ab"))
	tt, _ := richTester(t, doc)
	tt.Key(ui.Cmd, ui.KeyB)
	tt.Compose("ni", 2)
	tt.Compose("你", 1)
	tt.Compose("", 0)
	tt.Type("你")
	if got := runsOf(doc.Text()); got != "ab|*你*" {
		t.Fatalf("runs %q", got)
	}
}

func TestRichTextAppCommands(t *testing.T) {
	doc := NewDocument(NewText("first\nsecond"))
	tt, _ := richTester(t, doc)
	doc.SetSelection(0, 3)
	doc.ToggleBold()
	doc.SetHeading(1)
	tt.Frame()
	at := doc.Text()
	if runsOf(at) != "*fir*|st\nsecond" || at.ParagraphStyleAt(0).Heading != 1 {
		t.Fatalf("%q %+v", runsOf(at), at.ParagraphStyleAt(0))
	}
	// The editor follows: typing replaces the selection the app set.
	tt.Type("X")
	if got := runsOf(doc.Text()); got != "*X*|st\nsecond" {
		t.Fatalf("runs %q", got)
	}
}

func TestRichTextFormatMenu(t *testing.T) {
	doc := NewDocument(NewText("one two"))
	tt, _ := richTester(t, doc)
	doc.SetSelection(0, 3)
	tt.Frame()
	tt.Command("bold")
	tt.Command("alignCenter")
	at := doc.Text()
	if runsOf(at) != "*one*| two" || at.ParagraphStyleAt(0).Align != ui.Center {
		t.Fatalf("%q %+v", runsOf(at), at.ParagraphStyleAt(0))
	}
}

// TestRichTextEdgesAsNSTextView checks the edges a Swift probe measured of
// NSTextView on macOS 27: Enter and Backspace after bold keep typing bold,
// a link extends only inside, never at its start.
func TestRichTextEdgesAsNSTextView(t *testing.T) {
	bold := Attributes{Weight: 700}
	link := Attributes{Link: "https://x"}
	cases := []struct {
		name  string
		text  Text
		caret int
		keys  func(tt *ui.Tester)
		want  string
	}{
		{"enter at the end of bold", TextOf(Paragraph{Runs: []Run{{Text: "plain "}, {Text: "bold", Attributes: bold}}}, Paragraph{Runs: []Run{{Text: "next"}}}), 10,
			func(tt *ui.Tester) { tt.Key(0, ui.KeyEnter); typeRunes(tt, "x") }, "plain |*bold\nx\n*|next"},
		{"backspace a bold character", TextOf(Paragraph{Runs: []Run{{Text: "plain "}, {Text: "B", Attributes: bold}}}), 7,
			func(tt *ui.Tester) { tt.Key(0, ui.KeyBackspace); typeRunes(tt, "x") }, "plain |*x*"},
		{"the start of a link", TextOf(Paragraph{Runs: []Run{{Text: "see "}, {Text: "docs", Attributes: link}}}), 4,
			func(tt *ui.Tester) { typeRunes(tt, "x") }, "see x|[docs]"},
		{"a link starting a paragraph", TextOf(Paragraph{Runs: []Run{{Text: "docs", Attributes: link}, {Text: " now"}}}), 0,
			func(tt *ui.Tester) { typeRunes(tt, "x") }, "x|[docs]| now"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			doc := NewDocument(c.text)
			tt, _ := richTester(t, doc)
			doc.SetSelection(c.caret, c.caret)
			tt.Frame()
			c.keys(tt)
			if got := runsOf(doc.Text()); got != c.want {
				t.Fatalf("runs %q, want %q", got, c.want)
			}
		})
	}
}

// TestRichTextTypingIsOneStep types a word and undoes it whole, as
// NSTextView and TextArea do.
func TestRichTextTypingIsOneStep(t *testing.T) {
	doc := NewDocument(NewText(""))
	tt, _ := richTester(t, doc)
	typeRunes(tt, "abc")
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := doc.Text().String(); got != "" {
		t.Fatalf("one undo left %q", got)
	}
}

// TestRichTextCursor checks the pointer's shape: an arrow over a task's
// box, which a click checks, an I-beam over text and a hand over links.
func TestRichTextCursor(t *testing.T) {
	doc := NewDocument(TextOf(
		Paragraph{Runs: []Run{{Text: "milk"}}, Style: ParagraphStyle{List: Tasks}},
		Paragraph{Runs: []Run{{Text: "see "}, {Text: "the site", Attributes: Attributes{Link: "https://example.com"}}}},
	))
	tt, _ := richTester(t, doc)
	find := func(want ui.Cursor) (float32, float32, bool) {
		for y := float32(2); y < 80; y += 2 {
			for x := float32(0); x < 200; x += 2 {
				tt.Move(x, y)
				if tt.Cursor() == want {
					return x, y, true
				}
			}
		}
		return 0, 0, false
	}
	x, y, ok := find(ui.CursorDefault)
	if !ok {
		t.Fatal("no arrow over the task's box")
	}
	tt.ClickAt(x, y)
	if !doc.Text().ParagraphStyleAt(0).Checked {
		t.Fatalf("a click where the arrow shows doesn't check the task")
	}
	tt.Move(x+60, y)
	if got := tt.Cursor(); got != ui.CursorText {
		t.Fatalf("cursor over the task's text %v, want an I-beam", got)
	}
	if _, _, ok := find(ui.CursorPointer); !ok {
		t.Fatal("no hand over the link")
	}
}

// TestRichTextComposeUndo checks that undo takes back what an input method
// committed at a caret, not the typing before it too.
func TestRichTextComposeUndo(t *testing.T) {
	doc := NewDocument(Text{})
	tt, _ := richTester(t, doc)
	typeRunes(tt, "hello")
	tt.Key(0, ui.KeyLeft)
	tt.Key(0, ui.KeyRight)
	tt.Compose("ni", 2)
	tt.Type("你")
	if got := doc.Text().String(); got != "hello你" {
		t.Fatalf("text %q", got)
	}
	tt.Key(ui.Cmd, ui.KeyZ)
	if got := doc.Text().String(); got != "hello" {
		t.Fatalf("after undo %q, want %q", got, "hello")
	}
}

// TestRichTextComposeDropped checks that a composition the app's selection
// drops leaves the caret where the app put it.
func TestRichTextComposeDropped(t *testing.T) {
	doc := NewDocument(Text{})
	tt, _ := richTester(t, doc)
	start, _ := tt.TextCaret()
	typeRunes(tt, "abc")
	tt.Compose("nihao", 5)
	doc.SetSelection(0, 0)
	tt.Frame()
	if got, _ := tt.TextCaret(); got.X != start.X {
		t.Fatalf("caret at x %v, want %v", got.X, start.X)
	}
	tt.Type("z")
	if got := doc.Text().String(); got != "zabc" {
		t.Fatalf("text %q", got)
	}
}

// TestRichTextEnterAtCheckedTask checks that Enter at the start of a
// checked task keeps it checked, with an empty item above, as Notes.
func TestRichTextEnterAtCheckedTask(t *testing.T) {
	doc := NewDocument(TextOf(Paragraph{Runs: []Run{{Text: "done"}}, Style: ParagraphStyle{List: Tasks, Checked: true}}))
	tt, _ := richTester(t, doc)
	doc.SetSelection(0, 0)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	at := doc.Text()
	if at.String() != "\ndone" || at.ParagraphStyleAt(0).Checked || !at.ParagraphStyleAt(1).Checked {
		t.Fatalf("%q: checked %v, %v", at.String(), at.ParagraphStyleAt(0).Checked, at.ParagraphStyleAt(1).Checked)
	}
}

// TestRichTextTaskBoxOnly checks that a click in the margin away from a
// task's box doesn't check it.
func TestRichTextTaskBoxOnly(t *testing.T) {
	doc := NewDocument(TextOf(Paragraph{Runs: []Run{{Text: "todo"}}, Style: ParagraphStyle{List: Tasks}}))
	tt, _ := richTester(t, doc)
	for x := float32(0); x < 40; x += 2 {
		tt.ClickAt(x, 300)
	}
	if doc.Text().ParagraphStyleAt(0).Checked {
		t.Fatal("a click below the task checked it")
	}
}

// TestRichTextUnderscoreInWord checks that snake_case stays as typed.
func TestRichTextUnderscoreInWord(t *testing.T) {
	doc := NewDocument(Text{})
	tt, _ := richTester(t, doc)
	typeRunes(tt, "snake_case_name and _this_")
	if got := runsOf(doc.Text()); got != "snake_case_name and |_this_" {
		t.Fatalf("runs %q", got)
	}
}

// TestRichTextCodeBlockSpace checks that a code block's last line gets its
// space below back when the line after it leaves the block.
func TestRichTextCodeBlockSpace(t *testing.T) {
	doc := NewDocument(TextOf(
		Paragraph{Runs: []Run{{Text: "code"}}, Style: ParagraphStyle{CodeBlock: true}},
		Paragraph{Style: ParagraphStyle{CodeBlock: true}},
		Paragraph{Runs: []Run{{Text: "after"}}},
	))
	tt, _ := richTester(t, doc)
	doc.SetSelection(5, 5)
	tt.Frame()
	tt.Key(0, ui.KeyEnter)
	doc.SetSelection(doc.Text().Len(), doc.Text().Len())
	tt.Frame()
	got, _ := tt.TextCaret()
	fresh := NewDocument(doc.Text())
	ft, _ := richTester(t, fresh)
	want, _ := ft.TextCaret()
	if got.Y != want.Y {
		t.Fatalf("last line at y %v, want %v as laid out afresh", got.Y, want.Y)
	}
}

// TestRichTextParagraphKeys checks moving by paragraphs: ⌥↑↓ on macOS,
// Ctrl+↑↓ elsewhere.
func TestRichTextParagraphKeys(t *testing.T) {
	doc := NewDocument(NewText("one two\nthree four\nfive"))
	tt, _ := richTester(t, doc)
	mod := ui.Ctrl
	if runtime.GOOS == "darwin" {
		mod = ui.Alt
	}
	doc.SetSelection(12, 12)
	tt.Frame()
	tt.Key(mod, ui.KeyUp)
	if _, c := doc.Selection(); c != 8 {
		t.Fatalf("up to %d, want 8", c)
	}
	tt.Key(mod, ui.KeyUp)
	if _, c := doc.Selection(); c != 0 {
		t.Fatalf("up again to %d, want 0", c)
	}
	tt.Key(mod, ui.KeyDown)
	want := 7 // the end of the paragraph on macOS
	if runtime.GOOS != "darwin" {
		want = 8 // the start of the next elsewhere
	}
	if _, c := doc.Selection(); c != want {
		t.Fatalf("down to %d, want %d", c, want)
	}
}

// TestRichTextLinkOpensSafely clicks links: only those ui.SafeToOpen
// allows open, as "calc" would run a program on Windows.
func TestRichTextLinkOpensSafely(t *testing.T) {
	doc := NewDocument(TextOf(
		Paragraph{Runs: []Run{{Text: "run", Attributes: Attributes{Link: "calc"}}}},
		Paragraph{Runs: []Run{{Text: "site", Attributes: Attributes{Link: "https://example.com"}}}},
	))
	tt, _ := richTester(t, doc)
	for y := float32(2); y < 80; y += 6 { // apart, so that no click is a double click
		tt.Move(10, y)
		if tt.Cursor() == ui.CursorPointer {
			tt.ClickAt(10, y)
		}
	}
	got := tt.OpenedURLs()
	if len(got) == 0 || slices.ContainsFunc(got, func(u string) bool { return u != "https://example.com" }) {
		t.Errorf("opened %q", got)
	}
}

// TestRichTextLinkCanceledPress presses a link and has the system cancel
// the press: the link opens only for a click.
func TestRichTextLinkCanceledPress(t *testing.T) {
	doc := NewDocument(TextOf(
		Paragraph{Runs: []Run{{Text: "site", Attributes: Attributes{Link: "https://example.com"}}}},
	))
	tt, _ := richTester(t, doc)
	var x, y float32
	for y = 2; y < 40; y += 2 {
		if tt.Move(10, y); tt.Cursor() == ui.CursorPointer {
			x = 10
			break
		}
	}
	if x == 0 {
		t.Fatal("no link under the pointer")
	}
	tt.Press(x, y)
	tt.CancelPress()
	if got := tt.OpenedURLs(); len(got) != 0 {
		t.Fatalf("a canceled press opened %q", got)
	}
	tt.Advance(time.Second) // no double click
	tt.ClickAt(x, y)
	if got := tt.OpenedURLs(); !slices.Equal(got, []string{"https://example.com"}) {
		t.Fatalf("a click opened %q", got)
	}
}

// TestRichTextCJKWords deletes and moves by words in Chinese and
// Japanese: a character at a time, as they have no spaces (#668).
func TestRichTextCJKWords(t *testing.T) {
	word := ui.Ctrl
	if runtime.GOOS == "darwin" {
		word = ui.Alt
	}
	doc := NewDocument(TextOf(Paragraph{Runs: []Run{{Text: "Go语言，今日は"}}}))
	tt, _ := richTester(t, doc)
	tt.Key(word, ui.KeyBackspace)
	if got := doc.Text().String(); got != "Go语言，今日" {
		t.Fatalf("word backspace left %q", got)
	}
	for _, want := range []int{6, 5, 3, 2, 0} {
		tt.Key(word, ui.KeyLeft)
		if a, c := doc.Selection(); a != want || c != want {
			t.Fatalf("word left went to %d–%d, want %d", a, c, want)
		}
	}
}

// The scroll bar's thumb shows while the pointer is over the editor, as a
// scroll container's does, and not otherwise.
func TestScrollbarOnHover(t *testing.T) {
	doc := NewDocument(NewText(strings.Repeat("line\n", 200)))
	tt := ui.NewTester(func(c *ui.Context) { Editor(c, doc).Fill() }, 500, 400)
	thumb := func() bool {
		img := tt.Image()
		s := img.Bounds().Dx() / 500
		// Clear of the border's rounded corners.
		for y := 10 * s; y < 390*s; y++ {
			if img.RGBAAt(494*s, y) != img.RGBAAt(480*s, y) {
				return true
			}
		}
		return false
	}
	tt.Move(-10, -10)
	tt.Frame()
	if thumb() {
		t.Fatal("the thumb shows with the pointer away")
	}
	tt.Move(250, 200)
	tt.Frame()
	if !thumb() {
		t.Fatal("the thumb doesn't show with the pointer over the editor")
	}
	tt.Move(-10, -10)
	tt.Frame()
	if thumb() {
		t.Fatal("the thumb stayed as the pointer left")
	}
}
